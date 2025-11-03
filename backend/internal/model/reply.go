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
	// 【新增】
	// ReplyToUserID 指向这条回复直接@的用户ID。
	// 只有当它是二级回复时，这个字段才可能有值。
	ReplyToUserID *uint `json:"reply_to_user_id,omitempty" gorm:"index"`
	// --- GORM 关联关系 ---
	// 【新增】用于在返回JSON时携带子评论总数
	// `gorm:"-"` 标签告诉 GORM，这个字段不属于数据库表
	ChildRepliesCount int64 `json:"child_replies_count" gorm:"-"`
	// 一个回复属于一个帖子
	// omitempty: 如果没有预加载 Post，则在 JSON 中省略此字段
	Post Post `json:"post,omitempty" gorm:"foreignKey:PostID"`
	// 【补充】增加一个点赞计数字段
	LikesCount int `json:"likes_count" gorm:"default:0"`
	// 一个回复属于一个用户
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	// 【新增】预加载被回复的用户信息
	ReplyToUser User `json:"reply_to_user,omitempty" gorm:"foreignKey:ReplyToUserID"`
	// (可选) 楼中楼关系的自引用
	ParentReply  *Reply  `json:"parent_reply,omitempty" gorm:"foreignKey:ParentReplyID"`
	ChildReplies []Reply `json:"child_replies,omitempty" gorm:"foreignKey:ParentReplyID"`
}
