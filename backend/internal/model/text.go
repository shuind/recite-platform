package model

import (
	"time"

	"gorm.io/gorm"
)

type Text struct {
	// gorm.Model 展开以添加 json 标签
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 在 JSON 响应中隐藏

	Title      string `json:"title" gorm:"not null"`
	Content    string `json:"content" gorm:"type:text;not null"`
	Difficulty int    `json:"difficulty" gorm:"default:1"`
}
