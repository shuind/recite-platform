package model

import "time"

type TaskStatus string

const (
	TaskTodo       TaskStatus = "todo"
	TaskInProgress TaskStatus = "in_progress"
	TaskDone       TaskStatus = "done"
	TaskArchived   TaskStatus = "archived"
)

// 任务规划的核心实体
type TaskItem struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	UserID      uint       `json:"user_id" gorm:"index;not null"`
	Title       string     `json:"title" gorm:"size:200;not null"`
	Description string     `json:"description"`
	Priority    int16      `json:"priority" gorm:"not null;default:2"` // 0=P0..3=P3
	Status      TaskStatus `json:"status"  gorm:"type:varchar(20);not null;default:'todo'"`

	// 新增👇
	Score       int        `json:"score" gorm:"not null;default:0"` // 完成后获得的分数
	CompletedAt *time.Time `json:"completed_at"`
	ArchivedAt  *time.Time `json:"archived_at"` // 每周一0点设置，用于“归档”

	StartAt     *time.Time `json:"start_at"`
	DueAt       *time.Time `json:"due_at"`
	EstimateMin *int       `json:"estimate_min"`
	ManualOrder *int64     `json:"manual_order"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"index"`
}
