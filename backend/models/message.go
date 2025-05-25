package models

import (
	"gorm.io/gorm"
)

type Message struct {
    gorm.Model
    UserID    uint   `gorm:"column:sender_id;index;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;->"`
    Username  string `gorm:"size:50"`
    Content   string `gorm:"type:text"`
    Channel   string `gorm:"index;size:50"`
    MessageType string `gorm:"size:20;default:'message'"` // message, system
}