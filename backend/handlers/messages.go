package handlers

import (
    "backend/config"
    "backend/models"
    "github.com/gin-gonic/gin"
    "net/http"
    "strconv"
)

/**
 * 获取历史消息
 * @param c - Gin上下文
 */
func GetMessages(c *gin.Context) {
    channel := c.DefaultQuery("channel", "general")
    limitStr := c.DefaultQuery("limit", "50")
    limit, _ := strconv.Atoi(limitStr)
    
    if limit > 100 {
        limit = 100 // 限制最大返回数量
    }
    
    var messages []models.Message
    result := config.DB.Where("channel = ?", channel).Order("created_at desc").Limit(limit).Find(&messages)
    
    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
        return
    }
    
    c.JSON(http.StatusOK, messages)
}