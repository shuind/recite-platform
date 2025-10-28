package model

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	// gorm.Model 的字段通常也需要 json 标签
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 在 json 中忽略 DeletedAt

	RecordingID uint   `json:"recording_id" gorm:"not null;index"`
	UserID      uint   `json:"user_id" gorm:"not null"`
	Content     string `json:"content" gorm:"type:text;not null"`

	User User `json:"user" gorm:"foreignKey:UserID"` // 方便预加载作者信息
}
