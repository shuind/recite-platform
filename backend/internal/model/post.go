package model

import (
	"time"

	"gorm.io/gorm"
)

// Post 模型代表论坛中的一个帖子
type Post struct {
	// --- 【核心修正】将 gorm.Model 展开并为每个字段添加 json 标签 ---
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 在 JSON 响应中隐藏 DeletedAt

	// --- 您已正确添加的字段 ---
	UserID              uint       `json:"user_id"`
	Title               string     `json:"title"`
	Content             string     `json:"content" gorm:"type:text"` // 建议为长内容指定 gorm 类型
	IsPinned            bool       `json:"is_pinned" gorm:"default:false"`
	IsFeatured          bool       `json:"is_featured" gorm:"default:false"`
	RepliesCount        int        `json:"replies_count" gorm:"default:0"`
	ViewsCount          int        `json:"views_count" gorm:"default:0"`
	LastRepliedAt       *time.Time `json:"last_replied_at"`
	LastRepliedByUserID *uint      `json:"last_replied_by_user_id"`

	// --- 关联关系 (保持不变) ---
	// 为了API响应的一致性，确保 User 和 Reply 模型也遵循了同样的 json 标签规范
	User              User    `json:"user" gorm:"foreignKey:UserID"`
	Replies           []Reply `json:"replies,omitempty" gorm:"foreignKey:PostID"` // 使用 omitempty 避免在未预加载时返回 null
	LastRepliedByUser User    `json:"last_replied_by_user" gorm:"foreignKey:LastRepliedByUserID"`
}
