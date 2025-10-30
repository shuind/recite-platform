package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/shuind/language-learner/backend/internal/model"
)

type TaskHandler struct {
	DB *gorm.DB
}

// ---------------------- 输入结构 ----------------------

type CreateTaskInput struct {
	Title       string     `json:"title" binding:"required,min=1,max=200"`
	Description string     `json:"description"`
	Priority    *int16     `json:"priority"`
	StartAt     *time.Time `json:"start_at"`
	DueAt       *time.Time `json:"due_at"`
	EstimateMin *int       `json:"estimate_min"`
	Score       *int       `json:"score"` // 可选积分
}

type UpdateTaskInput struct {
	Title       *string           `json:"title"`
	Description *string           `json:"description"`
	Priority    *int16            `json:"priority"`
	StartAt     *time.Time        `json:"start_at"`
	DueAt       *time.Time        `json:"due_at"`
	EstimateMin *int              `json:"estimate_min"`
	Score       *int              `json:"score"`
	Status      *model.TaskStatus `json:"status"`
}

// ---------------------- 创建 ----------------------

func (h *TaskHandler) Create(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var in CreateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := model.TaskItem{
		UserID:      userID,
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Priority:    2,
		StartAt:     in.StartAt,
		DueAt:       in.DueAt,
		EstimateMin: in.EstimateMin,
		Score:       0,
	}

	if in.Priority != nil {
		item.Priority = *in.Priority
	}
	if in.Score != nil {
		item.Score = *in.Score
	}

	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// ---------------------- 查询 ----------------------

func scopeAutoOrder(db *gorm.DB) *gorm.DB {
	return db.Order(`
	(3.0*(3-priority)
	 + 2.0*COALESCE(GREATEST(0, 1 - EXTRACT(EPOCH FROM (due_at - NOW()))/86400.0/7.0), 0)
	 + 1.0*CASE WHEN start_at IS NULL OR start_at <= NOW() THEN 1 ELSE 0 END
	) DESC, COALESCE(due_at, 'infinity') ASC, id ASC`)
}

func (h *TaskHandler) List(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	view := c.DefaultQuery("view", "auto")
	status := c.Query("status")
	scope := c.DefaultQuery("scope", "active")

	db := h.DB.Model(&model.TaskItem{}).Where("user_id = ? AND deleted_at IS NULL", userID)

	switch scope {
	case "archived":
		db = db.Where("archived_at IS NOT NULL")
	case "all":
	default:
		db = db.Where("archived_at IS NULL")
	}

	if status != "" {
		db = db.Where("status = ?", status)
	} else {
		switch scope {
		case "archived":
			db = db.Where("status = ?", model.TaskDone)
		case "all":
		default:
			db = db.Where("status = ?", model.TaskTodo)
		}
	}

	if view == "manual" {
		db = db.Order("COALESCE(manual_order, 9223372036854775807) ASC").Order("id ASC")
	} else {
		db = scopeAutoOrder(db)
	}

	var items []model.TaskItem
	if err := db.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}
	if items == nil {
		items = make([]model.TaskItem, 0)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ---------------------- 单条读取 ----------------------

func (h *TaskHandler) Get(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var item model.TaskItem
	if err := h.DB.Where("id = ? AND user_id = ?", uint(id), userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// ---------------------- 更新 ----------------------

func (h *TaskHandler) Update(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var in UpdateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var item model.TaskItem
	if err := h.DB.Where("id = ? AND user_id = ? AND archived_at IS NULL", uint(id), userID).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or archived"})
		return
	}

	if in.Title != nil {
		item.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		item.Description = *in.Description
	}
	if in.Priority != nil {
		item.Priority = *in.Priority
	}
	if in.Status != nil {
		item.Status = *in.Status
	}
	if in.StartAt != nil {
		item.StartAt = in.StartAt
	}
	if in.DueAt != nil {
		item.DueAt = in.DueAt
	}
	if in.EstimateMin != nil {
		item.EstimateMin = in.EstimateMin
	}
	if in.Score != nil {
		item.Score = *in.Score
	}

	if err := h.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// ---------------------- 删除 ----------------------

func (h *TaskHandler) Delete(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	res := h.DB.Where("id = ? AND user_id = ? AND archived_at IS NULL", uint(id), userID).
		Delete(&model.TaskItem{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or archived"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------------------- 手动排序 ----------------------

type ReorderReq struct {
	IDs []uint `json:"ids" binding:"required"`
}

func (h *TaskHandler) Reorder(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req ReorderReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	tx := h.DB.Begin()
	for i, id := range req.IDs {
		if err := tx.Model(&model.TaskItem{}).
			Where("id = ? AND user_id = ? AND archived_at IS NULL", id, userID).
			Update("manual_order", i).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder"})
			return
		}
	}
	tx.Commit()
	c.Status(http.StatusNoContent)
}

// ---------------------- 完成 / 撤销 / 延后 ----------------------

func (h *TaskHandler) Complete(c *gin.Context) {
	uid := c.MustGet("userID").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	now := time.Now()
	err := h.DB.Model(&model.TaskItem{}).
		Where("id = ? AND user_id = ? AND archived_at IS NULL", uint(id), uid).
		Updates(map[string]any{
			"status":       model.TaskDone,
			"completed_at": &now,
		}).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to complete task"})
		return
	}
	c.Status(204)
}

type SnoozeReq struct {
	Minutes *int       `json:"minutes"`
	Until   *time.Time `json:"until"`
}

func (h *TaskHandler) Snooze(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req SnoozeReq
	if err := c.ShouldBindJSON(&req); err != nil || (req.Minutes == nil && req.Until == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide minutes or until"})
		return
	}
	update := map[string]any{}
	if req.Until != nil {
		update["start_at"] = req.Until
	} else {
		update["start_at"] = gorm.Expr("COALESCE(start_at, NOW()) + (? || ' minutes')::interval", *req.Minutes)
	}
	if err := h.DB.Model(&model.TaskItem{}).
		Where("id = ? AND user_id = ? AND archived_at IS NULL", uint(id), userID).
		Updates(update).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to snooze"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TaskHandler) Undo(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.DB.Model(&model.TaskItem{}).
		Where("id = ? AND user_id = ? AND archived_at IS NULL", uint(id), userID).
		Update("status", model.TaskTodo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to undo task"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------------------- 每周积分统计 ----------------------

func WeekStartMonday(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	off := time.Duration(wd-1) * 24 * time.Hour
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.Add(-off)
}

func (h *TaskHandler) WeeklyScore(c *gin.Context) {
	uid := c.MustGet("userID").(uint)
	tz := os.Getenv("APP_TZ")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}

	now := time.Now().In(loc)
	weekStart := WeekStartMonday(now)
	weekEnd := weekStart.Add(7 * 24 * time.Hour)

	type row struct {
		Day   string
		Score int64
	}
	var rows []row

	sql := `
	SELECT to_char((completed_at AT TIME ZONE ?), 'YYYY-MM-DD') AS day,
	       COALESCE(SUM(score), 0) AS score
	FROM task_items
	WHERE user_id = ?
	  AND deleted_at IS NULL
	  AND status = 'done'
	  AND completed_at >= ?
	  AND completed_at < ?
	GROUP BY 1
	ORDER BY 1;
	`
	if err := h.DB.Raw(sql, tz, uid, weekStart, weekEnd).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	scoreMap := map[string]int64{}
	for _, r := range rows {
		scoreMap[r.Day] = r.Score
	}

	type item struct {
		Date  string `json:"date"`
		Score int64  `json:"score"`
	}
	out := make([]item, 0, 7)
	for i := 0; i < 7; i++ {
		d := weekStart.Add(time.Duration(i) * 24 * time.Hour)
		key := d.Format("2006-01-02")
		out = append(out, item{
			Date:  key,
			Score: scoreMap[key],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tz":        tz,
		"week_from": weekStart.Format(time.RFC3339),
		"week_to":   weekEnd.Format(time.RFC3339),
		"items":     out,
	})
}

// GET /api/v1/tasks/score-trend?period=day|week|month
// GET /api/v1/tasks/score-trend?period=day|week|month
func (h *TaskHandler) ScoreTrend(c *gin.Context) {
	uid := c.MustGet("userID").(uint)
	period := c.DefaultQuery("period", "day") // day|week|month

	tz := os.Getenv("APP_TZ")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}

	now := time.Now().In(loc)

	// 对齐基准时间到周期起点
	var base time.Time                                                          // 对齐后的“本周期起点”（今天0点 / 本周一0点 / 本月1日0点）
	var trunc string                                                            // date_trunc 的粒度
	steps := 30                                                                 // 默认 30 天
	addStep := func(t time.Time, n int) time.Time { return t.AddDate(0, 0, n) } // 默认按天

	switch period {
	case "week":
		// 本周一 00:00
		wd := int(now.Weekday())
		if wd == 0 {
			wd = 7
		}
		base = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(wd - 1))
		trunc = "week"
		steps = 12
		addStep = func(t time.Time, n int) time.Time { return t.AddDate(0, 0, 7*n) }
	case "month":
		// 本月 1 日 00:00
		base = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		trunc = "month"
		steps = 12
		addStep = func(t time.Time, n int) time.Time { return t.AddDate(0, n, 0) }
	default: // day
		// 今天 00:00
		base = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		trunc = "day"
		steps = 30
		addStep = func(t time.Time, n int) time.Time { return t.AddDate(0, 0, n) }
	}

	// 查询区间：[start, end)
	start := addStep(base, -(steps - 1)) // 回溯 steps-1 个周期，含今天 => 共 steps 个点
	end := addStep(base, 1)              // 下一个周期的起点（闭开上界）

	type row struct {
		Period string
		Score  int64
	}
	var rows []row

	// Postgres：按时区截断并格式化为 YYYY-MM-DD
	sql := `
        SELECT to_char(date_trunc(?, completed_at AT TIME ZONE ?), 'YYYY-MM-DD') AS period,
               COALESCE(SUM(score), 0) AS score
        FROM task_items
        WHERE user_id = ?
          AND deleted_at IS NULL
          AND status = 'done'
          AND completed_at >= ?
          AND completed_at <  ?
        GROUP BY 1
        ORDER BY 1
    `
	if err := h.DB.Raw(sql, trunc, tz, uid, start, end).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	// 补桶
	scoreMap := make(map[string]int64, len(rows))
	for _, r := range rows {
		scoreMap[r.Period] = r.Score
	}

	type item struct {
		Date  string `json:"date"` // 与前端一致
		Score int64  `json:"score"`
	}
	out := make([]item, 0, steps)
	for i := 0; i < steps; i++ {
		t := addStep(start, i)
		key := t.Format("2006-01-02")
		out = append(out, item{
			Date:  key,
			Score: scoreMap[key],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"period":     period,
		"tz":         tz,
		"range_from": start.Format(time.RFC3339),
		"range_to":   end.Format(time.RFC3339),
		"items":      out,
	})
}
