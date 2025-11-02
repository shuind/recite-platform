// internal/model/content.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type Asset struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at"`
	UserID    uint   `gorm:"index;not null" json:"user_id"`
	Bucket    string `gorm:"type:varchar(128)" json:"bucket"`
	ObjectKey string `gorm:"type:text" json:"object_key"`
	URL       string `gorm:"type:text" json:"url"`
	MimeType  string `gorm:"type:varchar(128)" json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `gorm:"type:char(64)" json:"sha256"`
}

type NodeContent struct {
	ID          uint              `gorm:"primarykey" json:"id"`
	CreatedAt   int64             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   int64             `gorm:"autoUpdateTime" json:"updated_at"`
	NodeID      uint              `gorm:"uniqueIndex;not null" json:"node_id"`
	ContentType string            `gorm:"type:varchar(20);not null" json:"content_type"` // plain|markdown|image|code
	Body        string            `gorm:"type:text" json:"body"`
	CodeLang    string            `gorm:"type:varchar(32)" json:"code_lang"`
	AssetID     *uint             `json:"asset_id"`
	AssetURL    string            `gorm:"type:text" json:"asset_url"`
	Meta        datatypes.JSONMap `gorm:"type:jsonb" json:"meta,omitempty"`
}

type ExportTask struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	JobID      string    `gorm:"type:varchar(64);uniqueIndex" json:"job_id"` // 任务的唯一ID
	RootNodeID uint      `gorm:"not null" json:"root_node_id"`
	Format     string    `gorm:"type:varchar(10);not null" json:"format"` // md, txt, pdf
	Status     string    `gorm:"type:varchar(20);not null" json:"status"` // pending, processing, completed, failed
	OutputPath string    `gorm:"type:text" json:"output_path"`            // 在MinIO中的路径
	ErrorMsg   string    `gorm:"type:text" json:"error_msg"`
}
