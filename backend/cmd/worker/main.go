// /cmd/worker/main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	// PDF 渲染
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	// MinIO
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	// RabbitMQ
	amqp "github.com/rabbitmq/amqp091-go"

	// Markdown 渲染
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html" // 建议用别名，避免和变量名/包名冲突

	// DB
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// 你的模型
	"github.com/shuind/language-learner/backend/internal/model"
)

// Task 定义了从消息队列接收的任务结构
type Task struct {
	RecordingID uint   `json:"recording_id"`
	FileContent []byte `json:"file_content"`
}

// 导出任务
type ExportJob struct {
	ID         string // 用于命名导出文件；可用 uuid 或时间戳
	UserID     uint
	RootNodeID uint
	Format     string // "md" | "txt" | "pdf"
}

// 全局变量
var (
	DB          *gorm.DB
	minioClient *minio.Client
	minioBucket = "recordings"
)

// initDB 初始化数据库连接（带重试机制）
func initDB() {
	// ... 你的 initDB 函数保持不变 ...
	var err error
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=user password=password dbname=mydatabase port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	}

	maxRetries := 5 // 增加重试次数
	for i := 1; i <= maxRetries; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("Database connection established.")
			return
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v", i, maxRetries, err)
		log.Println("Retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	log.Fatalf("Could not connect to the database after %d attempts.", maxRetries)
}

// initMinio 初始化 MinIO 客户端
func initMinio() {
	// ... 你的 initMinio 函数保持不变 ...
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false

	var err error
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	log.Printf("MinIO client connected to %s", endpoint)

	err = minioClient.MakeBucket(context.Background(), minioBucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), minioBucket)
		if errBucketExists == nil && exists {
			log.Printf("Bucket '%s' already exists.", minioBucket)
		} else {
			log.Fatalf("Could not create or verify bucket: %v", err)
		}
	} else {
		log.Printf("Successfully created bucket '%s'", minioBucket)
	}
}

// ===================================================================
// =================== 核心修改在这里 ================================
// ===================================================================

// processTask 是处理单个任务的核心函数
func processTask(task Task) error {
	log.Printf("Processing task for RecordingID: %d", task.RecordingID)

	// --- 步骤 1: 上传文件到 MinIO ---
	objectName := fmt.Sprintf("%d-%d.webm", task.RecordingID, time.Now().Unix())
	fileReader := bytes.NewReader(task.FileContent)
	fileSize := int64(len(task.FileContent))

	_, err := minioClient.PutObject(context.Background(), minioBucket, objectName, fileReader, fileSize, minio.PutObjectOptions{
		ContentType: "audio/webm",
	})
	if err != nil {
		// 【修改】只返回错误，让上层处理
		return fmt.Errorf("minio upload failed for RecordingID %d: %w", task.RecordingID, err)
	}

	minioPublicEndpoint := os.Getenv("MINIO_PUBLIC_ENDPOINT")
	if minioPublicEndpoint == "" {
		minioPublicEndpoint = "http://localhost:9000"
	}
	audioURL := fmt.Sprintf("%s/%s/%s", minioPublicEndpoint, minioBucket, objectName)
	log.Printf("RecordingID %d uploaded to MinIO: %s", task.RecordingID, audioURL)

	// 【修改】将 status, audio_url 和初始 ai_status 一起更新
	DB.Model(&model.Recording{}).Where("id = ?", task.RecordingID).Updates(map[string]interface{}{
		"status":    "completed",
		"audio_url": audioURL,
		"ai_status": "pending", // 设置初始状态为 pending
	})

	// --- 步骤 2: 调用 AI 服务 ---
	log.Printf("RecordingID %d: Calling AI service for transcription...", task.RecordingID)
	DB.Model(&model.Recording{}).Where("id = ?", task.RecordingID).Update("ai_status", "processing")

	asrApiUrl := os.Getenv("ASR_API_URL")
	if asrApiUrl == "" {
		//【建议】使用 host.docker.internal 可能不稳定，可以直接指向宿主机 IP 或另一个服务名
		asrApiUrl = "http://host.docker.internal:8000/transcribe"
	}

	recognizedText, err := callAsrApi(asrApiUrl, task.FileContent)
	if err != nil {
		// AI 识别失败，只更新 AI 相关状态
		log.Printf("WARN: AI processing failed for RecordingID %d: %v", task.RecordingID, err)
		DB.Model(&model.Recording{}).Where("id = ?", task.RecordingID).Updates(map[string]interface{}{
			"ai_status":       "failed",
			"recognized_text": err.Error(), // 可以将错误信息存起来供排查
		})
		//【修改】不返回错误，因为核心任务已完成
		return nil
	}

	// --- 步骤 3: 将 AI 结果更新到数据库 ---
	log.Printf("RecordingID %d: Transcription result: \"%s\"", task.RecordingID, recognizedText)
	DB.Model(&model.Recording{}).Where("id = ?", task.RecordingID).Updates(map[string]interface{}{
		"ai_status":       "completed",
		"recognized_text": recognizedText,
	})

	log.Printf("Successfully processed task for RecordingID: %d. AI part completed.", task.RecordingID)
	return nil
}
func callAsrApi(url string, fileContent []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "audio.webm")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return "", fmt.Errorf("failed to copy file content to form: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 模型处理可能较慢，设置一个合理的超时时间
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI service returned an error. Status: %d, Body: %s", resp.StatusCode, string(bodyBytes))
	}

	var aiResult struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResult); err != nil {
		return "", fmt.Errorf("failed to decode AI service response: %w", err)
	}
	return aiResult.Text, nil
}

// failTask 标记任务为失败
func failTask(recordingID uint, reason string) {
	log.Printf("Marking task as failed for RecordingID %d. Reason: %s", recordingID, reason)
	// 在更新时只选择 status 字段，避免覆盖其他可能已更新的字段（如 audio_url）
	DB.Model(&model.Recording{}).Where("id = ?", recordingID).Update("status", "failed")
}

// main 函数基本保持不变
func main() {
	// ... 你的 main 函数前半部分保持不变 ...
	log.Println("Starting Worker Service...")

	initDB()
	initMinio()

	var conn *amqp.Connection
	var err error
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	for {
		conn, err = amqp.Dial(rabbitURL)
		if err == nil {
			log.Println("RabbitMQ connection established.")
			break
		}
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// 声明并消费导出队列
	exportQueue, err := ch.QueueDeclare(
		"content_export", // 队列名
		true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare export queue: %v", err)
	}

	exportMsgs, err := ch.Consume(
		exportQueue.Name, "", false, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to register export consumer: %v", err)
	}

	go func() {
		for d := range exportMsgs {
			var task model.ExportTask
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Printf("ERROR: Failed to unmarshal export task: %v", err)
				d.Ack(false)
				continue
			}
			processExportTask(task)
			d.Ack(false)
		}
	}()

	q, err := ch.QueueDeclare(
		"audio_processing",
		true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, "", false, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			log.Printf("Received a message with body size: %d", len(d.Body))
			var task Task
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Printf("ERROR: Failed to unmarshal message: %v. Discarding message.", err)
				d.Ack(false)
				continue
			}

			// 【修改】简化错误处理逻辑
			if err := processTask(task); err != nil {
				// 只有当 processTask 返回错误时，才意味着是致命错误
				log.Printf("FATAL ERROR during task processing for RecordingID %d: %v", task.RecordingID, err)
				// 更新主状态为 failed，AI 状态也一并失败
				DB.Model(&model.Recording{}).Where("id = ?", task.RecordingID).Updates(map[string]interface{}{
					"status":    "failed",
					"ai_status": "failed",
				})
			}

			// 无论成功与否，都确认消息，表示已经处理过
			// 这样可以防止坏消息（比如无法解析的JSON）无限循环
			d.Ack(false)
		}
	}()

	log.Println("Worker is waiting for messages. To exit press CTRL+C")
	<-forever
}

// NodeWithChildren 用于在内存中构建节点树
type NodeWithChildren struct {
	model.Node
	Children []NodeWithChildren `json:"children"`
}

// buildTreeRecursively 递归地从数据库加载整个节点树
func buildTreeRecursively(db *gorm.DB, nodeID uint, userID uint) (NodeWithChildren, error) {
	var node model.Node
	if err := db.Where("id = ? AND user_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return NodeWithChildren{}, err
	}

	result := NodeWithChildren{Node: node}

	if node.NodeType == "folder" {
		var children []model.Node
		db.Where("parent_id = ? AND user_id = ?", node.ID, userID).Order("title ASC").Find(&children)
		for _, child := range children {
			childTree, err := buildTreeRecursively(db, child.ID, userID)
			if err == nil {
				result.Children = append(result.Children, childTree)
			}
		}
	}
	return result, nil
}

// walkAndAppendMarkdown 遍历内存中的树，生成 Markdown 字符串
func walkAndAppendMarkdown(buf *strings.Builder, n NodeWithChildren, baseLevel int) {
	// 确保标题级别在 1 到 6 之间
	if baseLevel > 6 {
		baseLevel = 6
	}
	if baseLevel < 1 {
		baseLevel = 1
	}

	// 1. 写入当前节点的标题，作为 Markdown 的一级、二级...标题
	heading := strings.Repeat("#", baseLevel)
	fmt.Fprintf(buf, "%s %s\n\n", heading, n.Title)

	// 2. 如果是 'text' 类型的节点，直接将其 Content 写入
	// 因为 Content 本身就是完整的 Markdown 内容 (包含文字、图片链接、代码块等)
	if n.NodeType == "text" {
		buf.WriteString(n.Content)
		buf.WriteString("\n\n") // 在每个文件内容后添加换行，确保格式清晰
	}

	// 3. 如果是 'folder' 类型的节点，递归处理其所有子节点
	for _, ch := range n.Children {
		walkAndAppendMarkdown(buf, ch, baseLevel+1)
	}
}

// stripMarkdown 移除 Markdown 标记，用于生成 .txt
func stripMarkdown(md string) string {
	// 移除标题、粗体、斜体、链接等
	re := regexp.MustCompile(`(?m)^#{1,6}\s*|[*_]{1,2}|\[(.*?)\]\(.*?\)|` + "`" + ``)
	s := re.ReplaceAllString(md, "$1")
	return s
}

// markdownToHTML 使用 goldmark 将 markdown 转换为 HTML
func markdownToHTML(md string) string {
	var buf bytes.Buffer
	goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	).Convert([]byte(md), &buf)

	// 添加基础样式以提高PDF可读性
	return fmt.Sprintf(`
        <!DOCTYPE html><html><head><meta charset="UTF-8">
        <style>
            body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; line-height: 1.6; padding: 2em; }
            h1, h2, h3 { border-bottom: 1px solid #eaecef; padding-bottom: .3em; }
            code { font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace; }
            pre { background-color: #f6f8fa; padding: 16px; border-radius: 3px; overflow: auto; }
        </style>
        </head><body>%s</body></html>`, buf.String())
}

// htmlToPDF 使用 chromedp 将 HTML 转换为 PDF
func htmlToPDF(html string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var pdfBuffer []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("data:text/html,"+url.PathEscape(html)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuffer, _, err = page.PrintToPDF().Do(ctx)
			return err
		}),
	)
	return pdfBuffer, err
}

// processExportTask 是处理导出任务的核心函数
func processExportTask(task model.ExportTask) {
	log.Printf("Processing export task for JobID: %s", task.JobID)
	DB.Model(&task).Update("status", "processing")

	// 1. 从数据库递归加载节点树
	tree, err := buildTreeRecursively(DB, task.RootNodeID, task.UserID)
	if err != nil {
		log.Printf("ERROR: Failed to build tree for job %s: %v", task.JobID, err)
		DB.Model(&task).Updates(map[string]interface{}{"status": "failed", "error_msg": "Failed to load content tree."})
		return
	}

	// 2. 生成 Markdown 内容
	var markdownBuilder strings.Builder
	walkAndAppendMarkdown(&markdownBuilder, tree, 1) // 从一级标题开始
	markdownContent := markdownBuilder.String()

	// 3. 根据格式转换内容
	var fileContent []byte
	var contentType string

	switch task.Format {
	case "md":
		fileContent = []byte(markdownContent)
		contentType = "text/markdown; charset=utf-8"
	case "txt":
		fileContent = []byte(stripMarkdown(markdownContent))
		contentType = "text/plain; charset=utf-8"
	case "pdf":
		htmlContent := markdownToHTML(markdownContent)
		pdfBytes, err := htmlToPDF(htmlContent)
		if err != nil {
			log.Printf("ERROR: Failed to convert HTML to PDF for job %s: %v", task.JobID, err)
			DB.Model(&task).Updates(map[string]interface{}{"status": "failed", "error_msg": "PDF generation failed."})
			return
		}
		fileContent = pdfBytes
		contentType = "application/pdf"
	default:
		// Should not happen due to frontend validation
		DB.Model(&task).Updates(map[string]interface{}{"status": "failed", "error_msg": "Unsupported format."})
		return
	}

	// 4. 上传到 MinIO
	objectName := fmt.Sprintf("exports/%d/%s.%s", task.UserID, task.JobID, task.Format)
	_, err = minioClient.PutObject(context.Background(), minioBucket, objectName, bytes.NewReader(fileContent), int64(len(fileContent)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		log.Printf("ERROR: Failed to upload exported file for job %s: %v", task.JobID, err)
		DB.Model(&task).Updates(map[string]interface{}{"status": "failed", "error_msg": "Failed to save exported file."})
		return
	}

	// 5. 更新任务状态为完成
	DB.Model(&task).Updates(map[string]interface{}{"status": "completed", "output_path": objectName})
	log.Printf("Successfully processed export task for JobID: %s", task.JobID)
}
