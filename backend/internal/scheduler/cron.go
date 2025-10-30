// internal/scheduler/cron.go
package scheduler

import (
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/shuind/language-learner/backend/internal/model"
)

type Config struct {
	DB     *gorm.DB
	Logger *log.Logger // 可为 nil，默认用标准 logger
	// 环境：dev | prod，dev 下默认用每分钟触发，prod 用每周一 00:00
	Env string
	// 时区，默认 "Asia/Shanghai"
	Timezone string
	// 手动覆盖归档任务的 Cron 表达式（优先级最高），例如 "0 * * * * *"
	ArchiveSpecOverride string
}

// Start 启动调度器，返回停止函数
func Start(cfg Config) (stop func()) {
	if cfg.DB == nil {
		panic("scheduler.Config.DB is nil")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	// 时区
	tz := cfg.Timezone
	if tz == "" {
		tz = os.Getenv("APP_TZ")
		if tz == "" {
			tz = "Asia/Shanghai"
		}
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logger.Printf("[CRON] invalid timezone %q, fallback Asia/Shanghai: %v", tz, err)
		loc, _ = time.LoadLocation("Asia/Shanghai")
	}

	// 选择表达式
	spec := cfg.ArchiveSpecOverride
	if spec == "" {
		env := cfg.Env
		if env == "" {
			env = os.Getenv("APP_ENV") // dev | prod
		}
		if env == "dev" {
			spec = "0 * * * * *" // 每分钟 0 秒触发（方便本地验证）
		} else {
			spec = "0 0 0 * * 1" // 生产：每周一 00:00:00
		}
	}

	// 用秒级解析器
	c := cron.New(cron.WithSeconds(), cron.WithLocation(loc))

	// === 任务：归档已完成的任务 ===
	_, err = c.AddFunc(spec, func() {
		now := time.Now().In(loc)
		logger.Printf("[CRON] Archiving completed tasks... (%s)", now.Format(time.RFC3339))
		res := cfg.DB.Model(&model.TaskItem{}).
			Where("status = ? AND archived_at IS NULL", model.TaskDone).
			Update("archived_at", now)

		if res.Error != nil {
			logger.Printf("[CRON] archive failed: %v", res.Error)
			return
		}
		logger.Printf("[CRON] archived tasks: %d", res.RowsAffected)
	})
	if err != nil {
		logger.Fatalf("[CRON] failed to register archive job: %v", err)
	}

	c.Start()
	logger.Printf("[CRON] started (spec=%q, tz=%s)", spec, loc)

	return func() {
		ctx := c.Stop() // 返回 context，等待正在执行的 job 结束
		<-ctx.Done()
		logger.Printf("[CRON] stopped")
	}
}
