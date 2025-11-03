package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	// gorm.Model 展开以添加 json 标签
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 在 JSON 响应中隐藏
	AvatarURL string `json:"avatar_url"`
	Username string `gorm:"unique;not null" json:"username"`
	// Password 不应该被返回给前端
	Password string `gorm:"not null" json:"-"`

	// --- 【新增】社交计数缓存字段 ---

	// FollowersCount 存储该用户的粉丝数量。
	// `gorm:"default:0"`: 设置默认值为0。
	// `gorm:"column:followers_count"`: 明确映射到数据库列。
	FollowersCount int `gorm:"default:0;column:followers_count" json:"followers_count"`

	// FollowingCount 存储该用户关注的人的数量。
	FollowingCount int `gorm:"default:0;column:following_count" json:"following_count"`

	// --- GORM 关联关系 (Has Many) ---
	// 这部分也是可选的，但有助于构建更复杂的查询。

	// Followers 字段代表“关注我的人”的关系列表。
	// `gorm:"foreignKey:FollowingID"`: 指明在 Follower 模型中，是通过 FollowingID 字段来关联到当前 User 的。
	// 简单说就是：在 followers 表里，所有 following_id 等于我ID的记录，都是我的粉丝。
	// `json:"followers,omitempty"`: 如果没有预加载此关系，则在JSON中省略。
	Followers []Follower `gorm:"foreignKey:FollowingID" json:"followers,omitempty"`

	// Followings 字段代表“我关注的人”的关系列表。
	// `gorm:"foreignKey:FollowerID"`: 指明在 Follower 模型中，是通过 FollowerID 字段来关联到当前 User 的。
	// 简单说就是：在 followers 表里，所有 follower_id 等于我ID的记录，都是我关注的人。
	// `json:"followings,omitempty"`: 如果没有预加载此关系，则在JSON中省略。
	Followings []Follower `gorm:"foreignKey:FollowerID" json:"followings,omitempty"`
}
