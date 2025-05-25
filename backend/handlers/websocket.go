package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "backend/models"
    "backend/config"
    "context"
    "log"
    "sync"
    "time"
    "encoding/json"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

var (
    messageQueue = make(chan models.Message, 100)
    clients      = make(map[*websocket.Conn]*Client)
    clientsMutex = &sync.Mutex{}
)

// Client 表示一个WebSocket客户端
type Client struct {
    Username string
    Channels map[string]bool
}

// MessageType 定义消息类型
type MessageType string

const (
    MessageTypeMessage   MessageType = "message"
    MessageTypeSystem    MessageType = "system"
    MessageTypeHistory   MessageType = "history"
    MessageTypeSubscribe MessageType = "subscribe"
)

// ClientMessage 客户端发送的消息
type ClientMessage struct {
    Type      MessageType `json:"type"`
    Content   string      `json:"content,omitempty"`
    Channel   string      `json:"channel,omitempty"`
    Username  string      `json:"username,omitempty"`
    CreatedAt string      `json:"created_at,omitempty"`
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(c *gin.Context) {
    // 获取用户名
    username := c.Query("username")
    if username == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
        return
    }

    // 升级HTTP连接为WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Println("Failed to upgrade connection:", err)
        return
    }

    // 创建新客户端
    client := &Client{
        Username: username,
        Channels: make(map[string]bool),
    }

    // 注册客户端
    clientsMutex.Lock()
    clients[conn] = client
    clientsMutex.Unlock()

    // 发送系统消息通知用户已连接
    systemMsg := models.Message{
        Content: username + " 加入了聊天室",
        Channel: "general", // 默认频道
    }
    messageQueue <- systemMsg

    // 更新在线用户列表
    updateOnlineUsers()

    // 处理连接关闭
    defer func() {
        clientsMutex.Lock()
        delete(clients, conn)
        clientsMutex.Unlock()

        conn.Close()

        // 发送系统消息通知用户已断开连接
        systemMsg := models.Message{
            Content: username + " 离开了聊天室",
            Channel: "general", // 默认频道
        }
        messageQueue <- systemMsg

        // 更新在线用户列表
        updateOnlineUsers()
    }()

    // 处理接收到的消息
    for {
        var clientMsg ClientMessage
        err := conn.ReadJSON(&clientMsg)
        if err != nil {
            log.Println("Error reading message:", err)
            break
        }

        switch clientMsg.Type {
        case MessageTypeMessage:
            // 处理普通消息
            msg := models.Message{
                UserID:  0, // 这里可以设置实际的用户ID
                Content: clientMsg.Content,
                Channel: clientMsg.Channel,
            }
            messageQueue <- msg

        case MessageTypeHistory:
            // 处理历史消息请求
            sendHistoryMessages(conn, clientMsg.Channel)

        case MessageTypeSubscribe:
            // 处理频道订阅
            client.Channels[clientMsg.Channel] = true
            log.Printf("User %s subscribed to channel %s", client.Username, clientMsg.Channel)
        }
    }
}

// StartMessageDispatcher 启动消息分发器
func StartMessageDispatcher() {
    for {
        select {
        case msg := <-messageQueue:
            // 保存到数据库
            if err := config.DB.Create(&msg).Error; err != nil {
                log.Println("Failed to save message:", err)
            }

            // 发布到Redis
            msgJSON, _ := json.Marshal(msg)
            err := config.RDB.Publish(context.Background(), msg.Channel, string(msgJSON)).Err()
            if err != nil {
                log.Println("Failed to publish message to Redis:", err)
            }

            // 广播消息给订阅该频道的客户端
            broadcastMessage(msg)
        }
    }
}

// broadcastMessage 广播消息给订阅了指定频道的客户端
func broadcastMessage(msg models.Message) {
    clientMessage := ClientMessage{
        Type:      MessageTypeMessage,
        Content:   msg.Content,
        Channel:   msg.Channel,
        Username:  getUsernameByID(msg.UserID), // 这里需要根据UserID获取用户名
        CreatedAt: time.Now().Format(time.RFC3339),
    }

    clientsMutex.Lock()
    defer clientsMutex.Unlock()

    for conn, client := range clients {
        // 检查客户端是否订阅了该频道
        if client.Channels[msg.Channel] {
            err := conn.WriteJSON(clientMessage)
            if err != nil {
                log.Println("Error sending message to client:", err)
                conn.Close()
                delete(clients, conn)
            }
        }
    }
}

// sendHistoryMessages 发送历史消息给客户端
func sendHistoryMessages(conn *websocket.Conn, channel string) {
    var messages []models.Message
    result := config.DB.Where("channel = ?", channel).Order("created_at desc").Limit(50).Find(&messages)
    if result.Error != nil {
        log.Println("Failed to fetch history messages:", result.Error)
        return
    }

    // 反转消息顺序，使最早的消息在前
    for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
        messages[i], messages[j] = messages[j], messages[i]
    }

    for _, msg := range messages {
        clientMessage := ClientMessage{
            Type:      MessageTypeMessage,
            Content:   msg.Content,
            Channel:   msg.Channel,
            Username:  getUsernameByID(msg.UserID),
            CreatedAt: msg.CreatedAt.Format(time.RFC3339),
        }

        err := conn.WriteJSON(clientMessage)
        if err != nil {
            log.Println("Error sending history message to client:", err)
            return
        }
    }
}

// updateOnlineUsers 更新在线用户列表并广播给所有客户端
func updateOnlineUsers() {
    clientsMutex.Lock()
    defer clientsMutex.Unlock()

    var usernames []string
    for _, client := range clients {
        usernames = append(usernames, client.Username)
    }

    // 创建用户列表消息
    userListMsg := struct {
        Type  string   `json:"type"`
        Users []string `json:"users"`
    }{
        Type:  "userlist",
        Users: usernames,
    }

    // 广播给所有客户端
    for conn := range clients {
        err := conn.WriteJSON(userListMsg)
        if err != nil {
            log.Println("Error sending user list to client:", err)
        }
    }
}

// getUsernameByID 根据用户ID获取用户名
func getUsernameByID(userID uint) string {
    // 这里应该查询数据库获取用户名
    // 简化起见，这里直接返回一个默认值
    return "用户" + string(userID)
}