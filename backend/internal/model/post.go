package model

import (
	"time"

	"gorm.io/gorm"
)

// Post 模型代表论坛中的一个帖子
// 这个结构现在完全支持前端所需的所有功能
type Post struct {
	// --- 基础字段 (来自 gorm.Model) ---
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 在 JSON 响应中隐藏 DeletedAt

	// --- 核心内容与作者信息 ---
	UserID  uint   `json:"user_id"`
	Title   string `json:"title" gorm:"type:varchar(255)"`
	Content string `json:"content" gorm:"type:text;not null"`

	// --- 新增: 为 PostCard 提供内容摘要 ---
	Excerpt string `json:"excerpt" gorm:"type:text"` // 对应 PostCard 的摘要显示, 建议在创建帖子时自动生成并存储

	// --- 新增: 帖子状态与类型 ---
	Status   string `json:"status" gorm:"type:varchar(20);default:'published';index"`  // 帖子状态: published, draft. 用于实现草稿箱功能
	PostType string `json:"post_type" gorm:"type:varchar(20);default:'article';index"` // 帖子类型: article, question, thought. 对应 QuickPostBox

	// --- 统计与计数 ---
	ViewsCount   int `json:"views_count" gorm:"default:0"`
	RepliesCount int `json:"replies_count" gorm:"default:0"`
	// --- 【新增】媒体链接字段 ---
	ImageURL string `json:"image_url" gorm:"type:varchar(255)"`
	VideoURL string `json:"video_url" gorm:"type:varchar(255)"`
	// --- 新增: 赞同/反对数 ---
	VotesCount int `json:"votes_count" gorm:"default:0"` // 对应 PostCard 的赞同数
	// 【新增】关注数 (用于“问题”类型)
	FollowersCount int `json:"followers_count" gorm:"default:0"`
	// 【新增】回答数 (用于“问题”类型)
	AnswersCount int `json:"answers_count" gorm:"default:0"`

	// 【新增】父帖子ID (用于将“回答”关联到“问题”)
	// 使用指针 *uint 以允许其为 NULL
	ParentID *uint `json:"parent_id" gorm:"index"`
	// --- 用于排序和显示的额外字段 ---
	IsPinned            bool       `json:"is_pinned" gorm:"default:false"`
	IsFeatured          bool       `json:"is_featured" gorm:"default:false"`
	LastRepliedAt       *time.Time `json:"last_replied_at,omitempty"` // omitempty 表示如果为空则不在json中显示
	LastRepliedByUserID *uint      `json:"last_replied_by_user_id,omitempty"`

	// --- GORM 关联关系 (非常重要) ---
	// 确保在 API 响应中嵌套所需的用户信息
	User User `json:"user" gorm:"foreignKey:UserID"` // 作者信息，PostCard 显示所需

	// 注意：在帖子列表 API 中，通常不需要预加载 Replies 来避免性能问题
	Replies []Reply `json:"replies,omitempty" gorm:"foreignKey:PostID"`
	// 【新增】一个问题可以有多个回答 (Post本身)
	Answers []Post `json:"answers,omitempty" gorm:"foreignKey:ParentID"`
	// 最后一个回复的用户信息 (可选，但在帖子详情页可能有用)
	LastRepliedByUser User `json:"last_replied_by_user,omitempty" gorm:"foreignKey:LastRepliedByUserID"`
}
