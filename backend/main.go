package main

import (
    "backend/config"
    "backend/handlers"
    "backend/models"
    "github.com/gin-gonic/gin"
    "log"
    "net/http"
    "path/filepath"
)

func main() {
    if err := config.InitDB(); err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    config.InitRedis()

    // 自动迁移数据库表结构
    config.DB.Exec("DROP TABLE IF EXISTS messages")
    if err := config.DB.AutoMigrate(&models.Message{}); err != nil {
        log.Fatal("Failed to migrate database:", err)
    }

    go handlers.StartMessageDispatcher()

    r := gin.Default()

    // 设置前端文件路径
    frontendDir := "../frontend"
    absPath, _ := filepath.Abs(frontendDir)
    log.Printf("Serving frontend files from: %s", absPath)

    // 静态文件服务
    r.StaticFS("/static", http.Dir(filepath.Join(frontendDir, "static")))
    r.StaticFile("/", filepath.Join(frontendDir, "index.html"))
    r.StaticFile("/chat.js", filepath.Join(frontendDir, "chat.js"))
    r.StaticFile("/style.css", filepath.Join(frontendDir, "style.css"))

    // WebSocket路由
    r.GET("/ws", handlers.HandleWebSocket)

    if err := r.Run(":8080"); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}