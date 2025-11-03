package model

import (
	"time"
)

type Message struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time `json:"created_at"`
	SenderID    uint      `json:"sender_id" gorm:"index"`
	RecipientID uint      `json:"recipient_id" gorm:"index"`
	Content     string    `json:"content" gorm:"type:text"`
	IsRead      bool      `json:"is_read" gorm:"default:false"`
}
