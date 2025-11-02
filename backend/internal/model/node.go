package model

import "gorm.io/gorm"

type Node struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID   uint  `gorm:"not null;index" json:"user_id"`
	ParentID *uint `gorm:"index" json:"parent_id"`

	// 【修改】NodeType 统一使用 'text' 代表可编辑节点
	NodeType string `gorm:"type:varchar(20);not null;check:node_type IN ('folder', 'text')" json:"node_type"`

	Title string `gorm:"type:varchar(255);not null" json:"title"`

	// 【修改】只保留 Content 字段，用于存储所有 Markdown 内容
	Content string `gorm:"type:text" json:"content"`

	// 【移除】ContentType, CodeLang, AssetURLs, MarkdownTOC 字段

	// 关联关系保持不变
	Parent   *Node  `gorm:"foreignKey:ParentID;references:ID"`
	Children []Node `gorm:"foreignKey:ParentID;references:ID"`
}
