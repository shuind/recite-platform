package model

import "gorm.io/gorm"

// DomainNodeComment 模型代表对一个圈子内容节点的评论
type DomainNodeComment struct {
	// gorm.Model 内嵌了 ID, CreatedAt, UpdatedAt, DeletedAt
	// GORM 默认会将这些字段序列化为蛇形 (id, created_at, etc.)
	gorm.Model

	// `gorm:"not null;index"`: 定义数据库约束
	// `json:"domain_node_id"`: 【核心】明确指定序列化为 JSON 时的字段名
	DomainNodeID uint   `gorm:"not null;index" json:"domain_node_id"`
	UserID       uint   `gorm:"not null;index" json:"user_id"`
	Content      string `gorm:"type:text;not null" json:"content"`

	// --- GORM 关联关系 ---
	// `json:"user"`: 指定关联对象在 JSON 中的字段名
	User User `gorm:"foreignKey:UserID" json:"user"`
}
