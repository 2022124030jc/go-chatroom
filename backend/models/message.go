package models

import "gorm.io/gorm"

type Message struct {
    gorm.Model
    UserID    uint   `gorm:"index;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;->"` // 添加 -> 禁用外键约束
    Username  string `gorm:"size:50"`
    Content   string `gorm:"type:text"`
    Channel   string `gorm:"index;size:50"`
    MessageType string `gorm:"size:20;default:'message'"` // message, system
}