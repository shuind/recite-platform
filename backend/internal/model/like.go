package model

import "time"

// Like 模型代表了用户对录音的点赞关系
type Like struct {
	UserID      uint      `gorm:"primaryKey" json:"user_id"`
	RecordingID uint      `gorm:"primaryKey" json:"recording_id"`
	CreatedAt   time.Time `json:"created_at"`
	// 可以选择性地添加关联，但对于这个简单的表不是必须的
	// User        User      `gorm:"foreignKey:UserID"`
	// Recording   Recording `gorm:"foreignKey:RecordingID"`
}
