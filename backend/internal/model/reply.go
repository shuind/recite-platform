package model

import (
	"time"

	"gorm.io/gorm"
)

// Reply 模型代表对一个帖子的回复
type Reply struct {
	// gorm.Model 展开以添加 json 标签
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 在 JSON 响应中隐藏

	PostID  uint   `json:"post_id" gorm:"not null;index"` // 所属帖子的ID
	UserID  uint   `json:"user_id" gorm:"not null;index"` // 回复者的ID
	Content string `json:"content" gorm:"type:text;not null"`

	// --- 楼中楼回复 (可选) ---
	// ParentReplyID 指向它所回复的那条评论的ID
	// 同样使用指针以允许 NULL 值 (一级回复的 ParentReplyID 为 NULL)
	ParentReplyID *uint `json:"parent_reply_id,omitempty" gorm:"index"`

	// --- GORM 关联关系 ---

	// 一个回复属于一个帖子
	// omitempty: 如果没有预加载 Post，则在 JSON 中省略此字段
	Post Post `json:"post,omitempty" gorm:"foreignKey:PostID"`

	// 一个回复属于一个用户
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`

	// (可选) 楼中楼关系的自引用
	ParentReply  *Reply  `json:"parent_reply,omitempty" gorm:"foreignKey:ParentReplyID"`
	ChildReplies []Reply `json:"child_replies,omitempty" gorm:"foreignKey:ParentReplyID"`
}
