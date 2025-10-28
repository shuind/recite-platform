package main

import (
	"bytes"
	"context" // <-- 新增：用于 MinIO 操作的上下文
	"encoding/json"
	"errors"
	"fmt" // <-- 新增：用于格式化字符串 (如文件名和URL)
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"

	//"path/filepath" // <-- 新增：用于获取文件扩展名
	"strconv" // <-- 新增：用于将字符串转换为数字 (解析 UserID)
	"time"    // <-- 新增：用于生成时间戳文件名

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/minio/minio-go/v7"                 // <-- 新增：MinIO 主 SDK
	"github.com/minio/minio-go/v7/pkg/credentials" // <-- 新增：MinIO 凭证管理
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/shuind/language-learner/backend/internal/middleware"
	"github.com/shuind/language-learner/backend/internal/model"
	"github.com/shuind/language-learner/backend/internal/mq"
	"github.com/shuind/language-learner/backend/internal/utils"
)

// 全局数据库变量
var (
	DB            *gorm.DB
	minioClient   *minio.Client // 假设你已有
	jwtKey        []byte
	minioBucket   = "recordings"
	thinkTagRegex = regexp.MustCompile(`(?s)<think>.*?</think>`)
)
var mqManager *mq.RabbitMQManager // 使用新的管理器

// --- 通用 DTO ---
type AuthorResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// --- Recording 相关 DTO ---
type RecordingWithLikeStatus struct {
	model.Recording
	User        AuthorResponse `json:"user"`
	IsLikedByMe bool           `json:"is_liked_by_me"`
}

// --- Comment 相关 DTO ---
type CreateCommentInput struct {
	Content string `json:"content" binding:"required,min=1"`
}

type CommentResponse struct {
	model.Comment
	User AuthorResponse `json:"user"`
}

// --- Post 相关 DTO ---
type CreatePostInput struct {
	Title   string `json:"title" binding:"required,min=1,max=255"`
	Content string `json:"content" binding:"required,min=1"`
}

type PostListResponse struct {
	model.Post
	User              AuthorResponse  `json:"user"`
	LastRepliedByUser *AuthorResponse `json:"last_replied_by_user,omitempty"`
}

// --- Reply 相关 DTO ---
type CreateReplyInput struct {
	Content string `json:"content" binding:"required,min=1"`
}

type ReplyResponse struct {
	model.Reply
	User AuthorResponse `json:"user"`
}

// TranscribeRecordingHandler 手动触发对一个已存在录音的识别
func TranscribeRecordingHandler(c *gin.Context) {
	//log.Printf(">>> [HANDLER] Entering TranscribeRecordingHandler. Global rabbitChannel is: %p", rabbitChannel)
	userID, _ := c.Get("userID")
	recordingIDStr := c.Param("id")
	recordingID, _ := strconv.ParseUint(recordingIDStr, 10, 32)

	// 1. 查找录音，并验证所有权
	var recording model.Recording
	if err := DB.Where("id = ? AND user_id = ?", recordingID, userID).First(&recording).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found or permission denied"})
		return
	}
	// 【新增诊断步骤】在获取对象之前，先检查它的状态
	urlParts := strings.Split(recording.AudioURL, "/")
	if len(urlParts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid audio URL format"})
		return
	}
	objectName := urlParts[len(urlParts)-1]
	objInfo, err := minioClient.StatObject(context.Background(), minioBucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		log.Printf("!!!!!! [StatObject ERROR] Failed to get stats for object '%s': %v", objectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stat audio file in storage"})
		return
	}
	// 如果 StatObject 成功，我们可以打印出文件大小
	log.Printf("[StatObject INFO] Object '%s' found. Size: %d bytes.", objectName, objInfo.Size)
	// 2. 从 MinIO 下载录音文件内容
	// (需要 MinIO 客户端实例，也应在 main 函数中初始化)
	object, err := minioClient.GetObject(context.Background(), minioBucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audio file from storage"})
		return
	}

	defer object.Close()
	fileContent, err := io.ReadAll(object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read audio file content"})
		return
	}

	// 3. 创建一个新的 Task
	task := Task{
		RecordingID: uint(recordingID),
		FileContent: fileContent,
	}
	taskJSON, _ := json.Marshal(task)

	// 4. 将任务发布到 RabbitMQ
	ch := mqManager.GetChannel()
	if ch == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Message queue is not available right now"})
		return
	}
	err = ch.PublishWithContext(
		c.Request.Context(),
		"",                 // exchange
		"audio_processing", // routing key (queue name)
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        taskJSON,
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue transcription task"})
		return
	}

	// 5. 立即更新状态，并返回成功响应
	DB.Model(&model.Recording{}).Where("id = ?", recordingID).Update("ai_status", "pending")
	c.JSON(http.StatusAccepted, gin.H{"message": "Transcription task has been queued."})
}

type AIChatInput struct {
	Prompt string `json:"prompt" binding:"required"`
}

// AIChatHandler 处理与 Ollama 的对话
func AIChatHandler(c *gin.Context) {
	var input AIChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ollamaURL := os.Getenv("OLLAMA_API_URL")
	if ollamaURL == "" {
		ollamaURL = "http://host.docker.internal:11434/api/generate"
	}

	ollamaReqBody := map[string]interface{}{
		"model":  "deepseek-r1:1.5b",
		"prompt": "用中文会答. The user's query is: " + input.Prompt,
		"stream": false,
	}
	jsonData, _ := json.Marshal(ollamaReqBody)

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error connecting to Ollama: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to connect to AI service"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Ollama returned non-200 status: %d. Body: %s", resp.StatusCode, string(body))
		c.JSON(http.StatusBadGateway, gin.H{"error": "The AI service returned an error.", "details": string(body)})
		return
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		log.Printf("Failed to unmarshal Ollama success response: %v. Body: %s", err, string(body))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI service response"})
		return
	}

	// --- 4. 【核心清理逻辑】 ---
	// 在解析出原始回复后，返回给前端之前，进行清理
	rawAiReply := ollamaResp.Response

	// 使用正则表达式，将所有匹配 <think>...</think> 的部分替换为空字符串
	cleanedReply := thinkTagRegex.ReplaceAllString(rawAiReply, "")

	// 再修剪一下结果字符串，去掉可能多余的前后空格和换行符
	finalReply := strings.TrimSpace(cleanedReply)

	// 5. 将清理后的干净文本返回给前端
	c.JSON(http.StatusOK, gin.H{
		"reply": finalReply,
	})
}

func DomainOwnerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("userID")
		domainIDStr := c.Param("id") // 有些路由是 :id，有些是 :domainID，需要统一或兼容
		if domainIDStr == "" {
			domainIDStr = c.Param("domainId")
		}

		domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID"})
			return
		}

		var domain model.Domain
		// 核心检查：圈子是否存在，且 owner_id 是当前登录用户
		if err := DB.Where("id = ? AND owner_id = ?", domainID, userID).First(&domain).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Permission denied: you are not the owner of this domain"})
			return
		}

		// 将圈子信息存入 context，方便后续使用
		c.Set("domain", domain)
		c.Next()
	}
}

// 初始化数据库连接
func initDB() {

	var err error

	// 从 Docker Compose 提供的环境变量中读取数据库连接字符串 (DSN)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL 环境变量未设置") // 增加一个检查，更稳妥
	}

	// 使用这个 DSN 来连接数据库
	var attempts = 5 // 最多尝试5次
	for i := 0; i < attempts; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("Successfully connected to database.")
			// 成功连接，跳出循环
			break
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, attempts, err)
		if i < attempts-1 {
			log.Println("Retrying in 5 seconds...")
			time.Sleep(5 * time.Second) // 等待5秒后重试
		}
	}
	if err != nil {
		// 这里的日志信息可以更详细
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 自动迁移模型，这部分保持不变
	err = DB.AutoMigrate(&model.User{}, &model.Text{}, &model.Recording{}, &model.Node{}, &model.Domain{}, &model.DomainMember{}, &model.DomainNode{}, &model.Like{}, &model.Follower{}, &model.Post{}, &model.Reply{}, &model.DomainNodeComment{})
	if err != nil {
		log.Fatalf("Failed to auto migrate: %v", err)
	}
	seedTexts(DB)
	seedNodes(DB)
}
func seedNodes(db *gorm.DB) {
	// 检查用户1是否存在，如果不存在则不植入数据
	var user model.User
	if err := db.First(&user, 1).Error; err != nil {
		log.Println("User with ID 1 not found, skipping node seeding.")
		return
	}

	var count int64
	db.Model(&model.Node{}).Where("user_id = ?", 1).Count(&count)
	if count == 0 {
		log.Println("Seeding nodes for user 1...")
		nodes := []model.Node{
			{UserID: 1, NodeType: "folder", Title: "我的工作区"},
			{UserID: 1, NodeType: "folder", Title: "个人笔记"},
			{UserID: 1, NodeType: "text", Title: "快速便签", Content: "今天天气不错！"},
		}
		if err := db.Create(&nodes).Error; err != nil {
			log.Fatalf("Could not seed nodes: %v", err)
		}
		log.Println("Nodes seeded successfully.")
	}
}

type CreateNodeInput struct {
	ParentID *uint  `json:"parent_id"` // 使用指针以接受 null
	NodeType string `json:"node_type" binding:"required,oneof=folder text"`
	Title    string `json:"title" binding:"required,min=1,max=255"`
	Content  string `json:"content"`
}

func ListNodesHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	parentID := c.Query("parent_id") // 获取查询参数

	query := DB.Where("user_id = ?", userID)

	if parentID == "" || parentID == "null" {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}

	var nodes []model.Node
	if err := query.Order("node_type DESC, title ASC").Find(&nodes).Error; err != nil { // 文件夹排前面
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve nodes"})
		return
	}

	if nodes == nil {
		nodes = make([]model.Node, 0)
	}
	c.JSON(http.StatusOK, nodes)
}
func CreateNodeHandler(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 2. 绑定并验证输入
	var input CreateNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. 安全验证：如果指定了 parent_id，必须验证该父节点存在且属于当前用户
	if input.ParentID != nil {
		var parentNode model.Node
		// 查找父节点，条件是 ID 匹配 且 userID 匹配
		if err := DB.Where("id = ? AND user_id = ?", *input.ParentID, userID).First(&parentNode).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Parent folder not found or you don't have permission"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		// 额外验证：父节点必须是 'folder' 类型
		if parentNode.NodeType != "folder" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot create a node under a text file"})
			return
		}
	}

	// 4. 创建节点实例
	newNode := model.Node{
		UserID:   userID.(uint), // 从 context 取出的 userID 是 interface{}，需要类型断言
		ParentID: input.ParentID,
		NodeType: input.NodeType,
		Title:    input.Title,
		Content:  input.Content,
	}

	// 如果是 folder，强制清空 content
	if newNode.NodeType == "folder" {
		newNode.Content = ""
	}

	// 5. 保存到数据库
	if err := DB.Create(&newNode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create node"})
		return
	}

	// 6. 返回新创建的节点，状态码为 201 Created
	c.JSON(http.StatusCreated, newNode)
}

type UpdateNodeInput struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

// UpdateNodeHandler 更新一个节点 (标题或内容)
func UpdateNodeHandler(c *gin.Context) {
	// 1. 获取用户ID和URL中的节点ID
	userID, _ := c.Get("userID")
	nodeID := c.Param("id")

	// 2. 绑定输入
	var input UpdateNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. 安全查找：必须确保节点存在且属于当前用户
	var node model.Node
	if err := DB.Where("id = ? AND user_id = ?", nodeID, userID).First(&node).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found or permission denied"})
		return
	}

	// 4. 应用更新
	// 检查 title 是否被传入
	if input.Title != nil {
		// 可以增加验证，比如标题不能为空
		if strings.TrimSpace(*input.Title) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title cannot be empty"})
			return
		}
		node.Title = *input.Title
	}

	// 检查 content 是否被传入
	if input.Content != nil {
		// 业务规则：文件夹不能有内容
		if node.NodeType == "folder" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot set content for a folder"})
			return
		}
		node.Content = *input.Content
	}

	// 5. 保存更新
	if err := DB.Save(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update node"})
		return
	}

	// 6. 返回更新后的节点
	c.JSON(http.StatusOK, node)
}

// DeleteNodeHandler 删除一个节点
func DeleteNodeHandler(c *gin.Context) {
	// 1. 获取用户ID和URL中的节点ID
	userID, _ := c.Get("userID")
	nodeID := c.Param("id")

	// 2. 执行带权限检查的删除
	// GORM 的 Where().Delete() 会返回一个结果对象
	// 我们不需要先 Find() 再 Delete()，可以一步到位
	result := DB.Where("id = ? AND user_id = ?", nodeID, userID).Delete(&model.Node{})

	// 3. 检查是否有错误
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete node"})
		return
	}

	// 4. 检查是否真的删除了记录
	// 如果 RowsAffected == 0，说明没有找到匹配的记录（即节点不存在或不属于该用户）
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found or permission denied"})
		return
	}

	// 5. 返回成功响应
	// HTTP 规范中，成功的 DELETE 操作通常返回 204 No Content
	c.Status(http.StatusNoContent)
}

// SearchNodesHandler 搜索用户的节点
func SearchNodesHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	query := c.Query("q") // 获取搜索关键词，例如: /nodes/search?q=开发计划

	if strings.TrimSpace(query) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query cannot be empty"})
		return
	}

	var results []model.Node

	// 使用 to_tsquery 进行查询，它会将用户输入也转换为查询向量
	// @@ 是 "matches" 操作符
	// 我们还可以在这里添加排序，比如按更新时间
	err := DB.Where("user_id = ?", userID).
		Where("tsv @@ to_tsquery('simple', ?)", query).
		Order("updated_at DESC").
		Find(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to perform search"})
		return
	}

	if results == nil {
		results = make([]model.Node, 0)
	}

	c.JSON(http.StatusOK, results)
}

type MoveNodeInput struct {
	NewParentID *uint `json:"new_parent_id"`
}

// MoveNodeHandler 移动一个节点
func MoveNodeHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	nodeID := c.Param("id")

	var input MoveNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找要移动的节点，确保它属于当前用户
	var nodeToMove model.Node
	if err := DB.Where("id = ? AND user_id = ?", nodeID, userID).First(&nodeToMove).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node to move not found"})
		return
	}

	// 如果目标是某个文件夹，验证目标文件夹也属于当前用户
	if input.NewParentID != nil {
		// 防止移动到自身
		if *input.NewParentID == nodeToMove.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move a node into itself"})
			return
		}

		var targetParent model.Node
		if err := DB.Where("id = ? AND user_id = ?", *input.NewParentID, userID).First(&targetParent).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target folder not found"})
			return
		}
		if targetParent.NodeType != "folder" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target must be a folder"})
			return
		}
		// TODO: 还需要验证不能将父节点移动到其子节点下（防止循环）
	}

	// 更新 ParentID
	nodeToMove.ParentID = input.NewParentID
	DB.Save(&nodeToMove)

	c.JSON(http.StatusOK, gin.H{"message": "Node moved successfully"})
}
func initMinIO() {
	var err error
	ctx := context.Background()

	// 从环境变量读取配置 (这些已在 docker-compose.yml 中定义)
	endpoint := os.Getenv("MINIO_ENDPOINT") // e.g., "localhost:9000" or "minio:9000"
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := false // 在开发环境中通常不使用 SSL

	// 初始化 MinIO 客户端对象
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// 检查存储桶是否存在，如果不存在则创建
	bucketName := "recordings"
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalln(err)
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("Successfully created bucket %s\n", bucketName)

		// 设置存储桶策略为公开读，这样用户可以直接通过 URL 访问音频
		policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::` + bucketName + `/*"]}]}`
		err = minioClient.SetBucketPolicy(ctx, bucketName, policy)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("Successfully set bucket policy for %s\n", bucketName)
	}
}

func seedTexts(db *gorm.DB) {
	var count int64
	db.Model(&model.Text{}).Count(&count)
	if count == 0 {
		log.Println("No texts found, seeding database...")
		texts := []model.Text{
			{Title: "The North Wind and the Sun", Content: "The North Wind and the Sun were disputing which was the stronger, when a traveler came along wrapped in a warm cloak.", Difficulty: 1},
			{Title: "A Fox and a Crane", Content: "A Fox invited a Crane to supper and provided nothing for his entertainment but some soup in a very shallow dish.", Difficulty: 2},
			{Title: "The Ant and the Grasshopper", Content: "In a field one summer's day a Grasshopper was hopping about, chirping and singing to its heart's content.", Difficulty: 1},
		}
		if err := db.Create(&texts).Error; err != nil {
			log.Fatalf("Could not seed texts: %v", err)
		}
		log.Println("Texts seeded successfully.")
	}
}

// --- Handler ---
type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func RegisterHandler(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	var existingUser model.User
	if err := DB.Where("username = ?", input.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// 创建用户
	user := model.User{Username: input.Username, Password: string(hashedPassword)}
	result := DB.Create(&user)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Registration successful"})
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 根据用户名查找用户
	var user model.User
	if err := DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 2. 比较哈希密码和用户输入的密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 3. 密码正确，生成 JWT Token
	expirationTime := time.Now().Add(24 * time.Hour)

	// 创建 claims (声明)，包含用户 ID 和标准声明
	claims := &jwt.RegisteredClaims{
		// --- 这里是修正的地方 ---
		Subject: strconv.FormatUint(uint64(user.ID), 10), // 正确地将 uint 转换为 string
		// -----------------------
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}

	// 创建 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用我们设置的密钥签名 Token
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	// 4. 返回 Token
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":       user.ID, // 注意 GORM 默认是 ID (大写)
			"username": user.Username,
			// 如果有 role 等其他安全信息，也可以在这里返回
		},
	})
}

func ListTextsHandler(c *gin.Context) {
	var texts []struct { // 定义一个临时结构体，避免传输大的 content 字段
		ID         uint   `json:"id"`
		Title      string `json:"title"`
		Difficulty int    `json:"difficulty"`
	}

	// 只选择需要的字段
	if err := DB.Model(&model.Text{}).Select("id, title, difficulty").Find(&texts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve texts"})
		return
	}

	c.JSON(http.StatusOK, texts)
}

// GetTextHandler 获取单个文本的完整内容
func GetTextHandler(c *gin.Context) {
	var text model.Text
	textID := c.Param("id") // 从 URL 路径中获取 "id"

	if err := DB.First(&text, textID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Text not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve text"})
		return
	}

	c.JSON(http.StatusOK, text)
}

type Task struct {
	RecordingID uint   `json:"recording_id"`
	FileContent []byte `json:"file_content"`
}

// UploadHandler 统一处理所有来源的录音上传
func UploadHandler(c *gin.Context) {
	// === 临时调试代码 开始 ===
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB max memory
		log.Printf("Debug: Error parsing multipart form: %v", err)
	}
	log.Println("--- Debug: Received Upload Form Data ---")
	for key, values := range c.Request.PostForm {
		for _, value := range values {
			log.Printf("Field[%s]: %s", key, value)
		}
	}
	log.Println("------------------------------------")
	// 1. 从认证中间件获取用户ID
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userIDVal.(uint)

	// 2. 灵活获取表单数据：text_id, node_id, 或 domain_node_id
	textIDStr := c.PostForm("text_id")
	nodeIDStr := c.PostForm("node_id")
	domainNodeIDStr := c.PostForm("domain_node_id")

	// 检查是否提供了多个ID，这是不允许的
	providedCount := 0
	if textIDStr != "" {
		providedCount++
	}
	if nodeIDStr != "" {
		providedCount++
	}
	if domainNodeIDStr != "" {
		providedCount++
	}

	if providedCount > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot provide more than one of text_id, node_id, or domain_node_id"})
		return
	}
	if providedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "One of text_id, node_id, or domain_node_id is required"})
		return
	}

	// 使用指针类型，因为它们在模型中是可选的
	var textID, nodeID, domainNodeID *uint

	// --- 情况A: 上传到公共文本 (text_id) ---
	if textIDStr != "" {
		id, err := strconv.ParseUint(textIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid text_id format"})
			return
		}
		val := uint(id)
		textID = &val
		// (可选) 检查 text_id 是否真的存在于 texts 表中
	} else if nodeIDStr != "" {
		id, err := strconv.ParseUint(nodeIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node_id format"})
			return
		}
		val := uint(id)

		// 【安全检查】验证个人节点的所有权和类型
		var node model.Node
		if err := DB.Where("id = ? AND user_id = ?", val, userID).First(&node).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied or personal node not found"})
			return
		}
		if node.NodeType != "text" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot record for a folder node"})
			return
		}
		nodeID = &val
	} else if domainNodeIDStr != "" {
		id, err := strconv.ParseUint(domainNodeIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain_node_id format"})
			return
		}
		val := uint(id)

		// 【安全检查】验证用户是否是该圈子节点的成员
		var domainNode model.DomainNode
		if err := DB.Where("id = ?", val).First(&domainNode).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Domain node not found"})
			return
		}
		if domainNode.NodeType != "text" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot record for a folder node in a domain"})
			return
		}
		var member model.DomainMember
		if err := DB.Where("domain_id = ? AND user_id = ?", domainNode.DomainID, userID).First(&member).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this domain"})
			return
		}
		domainNodeID = &val
	}

	// 3. 获取音频文件 (逻辑不变)
	file, _, err := c.Request.FormFile("audio_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not get audio file"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
		return
	}

	// 4. 在数据库中创建记录，包含所有可能的 ID
	newRecording := model.Recording{
		UserID:       userID,
		TextID:       textID,
		NodeID:       nodeID,
		DomainNodeID: domainNodeID, // 确保模型中有这个字段
		Status:       "processing",
		// Title 可以在转码后由 worker 根据关联的文本标题填充
	}
	if err := DB.Create(&newRecording).Error; err != nil {
		log.Printf("Database create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save recording metadata"})
		return
	}
	log.Printf("Created new recording record with ID: %d", newRecording.ID)

	// 5. 将任务发布到 RabbitMQ (逻辑不变)
	task := Task{ // 假设你的 Task 结构体是这样
		RecordingID: newRecording.ID,
		FileContent: fileBytes,
	}
	taskJSON, err := json.Marshal(task)
	if err != nil {
		// ... 错误处理
		return
	}

	ch := mqManager.GetChannel()

	// 2. 【重要】检查 Channel 是否真的可用
	if ch == nil {
		log.Println("FATAL: Could not get RabbitMQ channel from manager.")
		DB.Delete(&newRecording) // 回滚数据库操作
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Message queue service is currently unavailable"})
		return
	}

	// 3. 使用这个健康的 Channel 来发布消息
	err = ch.PublishWithContext(c.Request.Context(), "", "audio_processing", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         taskJSON,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		log.Printf("Failed to publish message to RabbitMQ: %v", err)
		DB.Delete(&newRecording) // 回滚数据库操作
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue file for processing"})
		return
	}

	log.Printf("Successfully published task for RecordingID: %d", newRecording.ID)

	// 6. 立即返回 202 Accepted 响应
	c.JSON(http.StatusAccepted, gin.H{
		"message":      "File uploaded and is being processed.",
		"recording_id": newRecording.ID,
	})
}

// ListRecordingsForNodeHandler 获取某个节点的所有录音
func ListRecordingsForNodeHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	nodeID := c.Param("id")

	// 再次验证节点所有权
	var node model.Node
	if err := DB.Where("id = ? AND user_id = ?", nodeID, userID).First(&node).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	var recordings []model.Recording
	// 查询条件：user_id 和 node_id 都匹配
	DB.Where("user_id = ? AND node_id = ?", userID, nodeID).Order("created_at desc").Find(&recordings)

	c.JSON(http.StatusOK, recordings)
}

// ListMyRecordingsHandler 获取当前登录用户的所有录音记录（包括公共和私有的）
func ListMyRecordingsHandler(c *gin.Context) {
	// 1. 从 Gin Context 中获取用户 ID (这部分逻辑不变)
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userIDVal.(uint)

	// 2. 【修改】查询数据库时，同时预加载 Text 和 Node 的信息
	var recordings []model.Recording
	if err := DB.
		Preload("Text"). // 预加载关联的公共文本
		Preload("Node"). // 预加载关联的私有节点
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&recordings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve recordings"})
		return
	}

	// 3. 【修改】格式化返回的数据 DTO，使其能容纳两种来源
	type RecordingResponse struct {
		ID         uint      `json:"id"`
		AudioURL   string    `json:"audio_url"`
		Status     string    `json:"status"`
		CreatedAt  time.Time `json:"created_at"`
		Title      string    `json:"title"`       // 通用标题字段
		SourceType string    `json:"source_type"` // 'public' or 'private'
		SourceID   uint      `json:"source_id"`   // 来源的 ID (可能是 text_id 或 node_id)
	}

	var response []RecordingResponse
	for _, r := range recordings {
		var title string
		var sourceType string
		var sourceID uint

		// 【核心逻辑】根据关联关系判断来源和标题
		if r.TextID != nil && r.Text.ID != 0 { // 检查 Text 是否被成功加载
			title = r.Text.Title
			sourceType = "public"
			sourceID = *r.TextID
		} else if r.NodeID != nil && r.Node.ID != 0 { // 检查 Node 是否被成功加载
			title = r.Node.Title
			sourceType = "private"
			sourceID = *r.NodeID
		} else {
			// 兜底情况，可能关联的数据已被删除但录音记录还在
			title = "未知来源"
			sourceType = "unknown"
			sourceID = 0
		}

		response = append(response, RecordingResponse{
			ID:         r.ID,
			AudioURL:   r.AudioURL,
			Status:     r.Status,
			CreatedAt:  r.CreatedAt,
			Title:      title,
			SourceType: sourceType,
			SourceID:   sourceID,
		})
	}

	// 如果没有录音，返回一个空数组而不是 null (这部分逻辑不变)
	if response == nil {
		response = make([]RecordingResponse, 0)
	}

	c.JSON(http.StatusOK, response)
}

// UpdateRecordingInput 定义重命名录音的输入
type UpdateRecordingInput struct {
	Title string `json:"title" binding:"required,max=255"`
}

// UpdateRecordingHandler 重命名一个录音
func UpdateRecordingHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	recordingID := c.Param("id")

	var input UpdateRecordingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 安全查找：确保录音存在且属于当前用户
	var recording model.Recording
	if err := DB.Where("id = ? AND user_id = ?", recordingID, userID).First(&recording).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found or permission denied"})
		return
	}

	// 更新标题并保存
	recording.Title = input.Title
	DB.Save(&recording)

	c.JSON(http.StatusOK, recording)
}

// DeleteRecordingHandler 删除一个录音
func DeleteRecordingHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	recordingID := c.Param("id")

	// 安全删除：确保只删除属于自己的录音
	result := DB.Where("id = ? AND user_id = ?", recordingID, userID).Delete(&model.Recording{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete recording"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found or permission denied"})
		return
	}

	c.Status(http.StatusNoContent)
}

// === Create Domain Handler ===
type CreateDomainInput struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
}

func CreateDomainHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	var input CreateDomainInput
	if err := c.ShouldBindJSON(&input); err != nil { /* ... */
	}

	// 创建圈子
	newDomain := model.Domain{
		OwnerID:     userID.(uint),
		Name:        input.Name,
		Description: input.Description,
		JoinCode:    utils.GenerateRandomString(8), // 生成唯一邀请码
	}
	// TODO: 需要循环检查确保邀请码唯一性，虽然碰撞概率极低

	// 使用事务：创建 domain 和将 owner 添加为 member 必须同时成功
	tx := DB.Begin()
	if err := tx.Create(&newDomain).Error; err != nil {
		tx.Rollback()
		// ... 返回错误
		return
	}

	// 将创建者作为 "owner" 加入成员表
	ownerMember := model.DomainMember{
		DomainID: newDomain.ID,
		UserID:   userID.(uint),
		Role:     "owner",
	}
	if err := tx.Create(&ownerMember).Error; err != nil {
		tx.Rollback()
		// ... 返回错误
		return
	}

	tx.Commit()
	c.JSON(http.StatusCreated, newDomain)
}

// === Join Domain Handler ===
type JoinDomainInput struct {
	DomainID uint   `json:"domain_id" binding:"required"`
	JoinCode string `json:"join_code" binding:"required"`
}

// === Join Domain Handler ===
func JoinDomainHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	var input JoinDomainInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 【核心修改】根据 圈子ID 和 邀请码 同时查找
	var domain model.Domain
	if err := DB.Where("id = ? AND join_code = ?", input.DomainID, input.JoinCode).First(&domain).Error; err != nil {
		// 这里的错误原因可能是 ID 不对，也可能是 code 不对，统一返回一个模糊的错误
		c.JSON(http.StatusNotFound, gin.H{"error": "圈子信息或邀请码不正确"})
		return
	}

	// 检查用户是否已是成员 (逻辑不变)
	var existingMember model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ?", domain.ID, userID).First(&existingMember).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "你已经是该圈子成员"})
		return
	}

	// 添加新成员 (逻辑不变)
	newMember := model.DomainMember{
		DomainID: domain.ID,
		UserID:   userID.(uint),
		Role:     "member",
	}
	DB.Create(&newMember)

	c.JSON(http.StatusOK, gin.H{"message": "成功加入圈子", "domain": domain})
}

// === List My Domains Handler ===
func ListMyDomainsHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	var domains []model.Domain
	// 通过 domain_members 表进行 JOIN 查询
	DB.Joins("join domain_members on domain_members.domain_id = domains.id").
		Where("domain_members.user_id = ?", userID).
		Find(&domains)

	c.JSON(http.StatusOK, domains)
}

type PublishNodeInput struct {
	SourceNodeID uint `json:"source_node_id" binding:"required"`
	// MVP 版本先默认发布到圈子根目录，未来可扩展 target_parent_id
}

// PublishNodeToDomainHandler 是 API 的入口
func PublishNodeToDomainHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	domainID, _ := strconv.ParseUint(c.Param("domainId"), 10, 32)

	var input PublishNodeInput
	if err := c.ShouldBindJSON(&input); err != nil { /* ... */
	}

	// --- 安全性与权限检查 ---
	// 1. 检查用户是否是该圈子的圈主 (只有圈主能发布内容)
	var domain model.Domain
	if err := DB.Where("id = ? AND owner_id = ?", domainID, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: you are not the owner of this domain"})
		return
	}

	// 2. 检查源节点是否存在且属于该用户
	var sourceNode model.Node
	if err := DB.Where("id = ? AND user_id = ?", input.SourceNodeID, userID).First(&sourceNode).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source node not found"})
		return
	}

	// --- 核心逻辑：使用事务执行递归复制 ---
	tx := DB.Begin()
	err := recursiveCopyNode(tx, sourceNode.ID, uint(domainID), nil) // nil 表示发布到根目录
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish content", "details": err.Error()})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Content published successfully"})
}

// recursiveCopyNode 是真正的递归复制函数
func recursiveCopyNode(tx *gorm.DB, sourceNodeID uint, domainID uint, targetParentID *uint) error {
	// 1. 获取源节点信息
	var sourceNode model.Node
	if err := tx.First(&sourceNode, sourceNodeID).Error; err != nil {
		return err
	}

	// 2. 创建一个新的 domain_node (副本)
	newDomainNode := model.DomainNode{
		DomainID: domainID,
		ParentID: targetParentID,
		NodeType: sourceNode.NodeType,
		Title:    sourceNode.Title,
		Content:  sourceNode.Content,
	}
	if err := tx.Create(&newDomainNode).Error; err != nil {
		return err
	}

	// 3. 【递归部分】如果源节点是文件夹，则递归复制其所有子节点
	if sourceNode.NodeType == "folder" {
		var children []model.Node
		tx.Where("parent_id = ?", sourceNode.ID).Find(&children)

		for _, child := range children {
			// 将新创建的 domain_node 的 ID 作为下一轮递归的 targetParentID
			err := recursiveCopyNode(tx, child.ID, domainID, &newDomainNode.ID)
			if err != nil {
				return err // 如果任何一个子节点复制失败，整个事务都会回滚
			}
		}
	}
	return nil
}

func ListOwnedDomainsHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	var domains []model.Domain
	// 直接查询 domains 表，条件是 owner_id 匹配
	DB.Where("owner_id = ?", userID).Find(&domains)
	c.JSON(http.StatusOK, domains)
}

// GetDomainDetailsHandler 获取圈子详情和用户角色
func GetDomainDetailsHandler(c *gin.Context) {
	userIDInterface, _ := c.Get("userID")
	domainIDStr := c.Param("domainId") // 假设你已统一为 domainID

	// --- 详细日志打印 ---
	log.Printf("--- Enter GetDomainDetailsHandler ---")
	log.Printf("Raw userID from context (interface{}): %v (Type: %T)", userIDInterface, userIDInterface)
	log.Printf("Raw domainID from URL (string): %s", domainIDStr)

	// 进行类型转换
	userID, ok := userIDInterface.(uint)
	if !ok {
		log.Printf("ERROR: Failed to assert userID to uint")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: user ID format is incorrect"})
		return
	}

	domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
	if err != nil {
		log.Printf("ERROR: Failed to parse domainID string to uint. Error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID format"})
		return
	}

	log.Printf("Parsed userID (uint): %d", userID)
	log.Printf("Parsed domainID (uint): %d", domainID)
	log.Printf("Executing DB query: SELECT * FROM domain_members WHERE domain_id = %d AND user_id = %d", domainID, userID)

	var member model.DomainMember
	// 使用转换后的、类型确定的变量进行查询
	result := DB.Where("domain_id = ? AND user_id = ?", uint(domainID), userID).First(&member)

	if result.Error != nil {
		log.Printf("DB query failed. Records found: %d. Error: %v", result.RowsAffected, result.Error)
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this domain"})
		return
	}

	log.Printf("DB query successful. Found member record: %+v", member)
	log.Printf("--- Exit GetDomainDetailsHandler ---")

	// 2. 获取圈子信息
	var domain model.Domain
	DB.First(&domain, domainID)

	// 3. 返回信息，包含角色
	c.JSON(http.StatusOK, gin.H{
		"domain": domain,
		"role":   member.Role,
	})
}

// ListDomainNodesHandler 获取圈子内的节点
func ListDomainNodesHandler(c *gin.Context) {
	// --- 1. 参数获取与校验 ---
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	domainIDStr := c.Param("domainId")
	domainID, err := strconv.ParseUint(domainIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID format"})
		return
	}

	parentIDStr := c.Query("parent_id")

	// --- 2. 权限验证 ---
	var member model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ?", domainID, userID).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: you are not a member of this domain"})
			return
		}
		log.Printf("Error checking domain membership: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// --- 3. 构建数据库查询 ---
	query := DB.Where("domain_id = ?", domainID)

	if parentIDStr == "" || parentIDStr == "null" || parentIDStr == "0" {
		log.Printf("Querying root nodes for domain ID: %d", domainID)
		query = query.Where("parent_id IS NULL")
	} else {
		parentID, err := strconv.ParseUint(parentIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent_id format"})
			return
		}
		log.Printf("Querying child nodes for parent ID: %d in domain ID: %d", parentID, domainID)
		query = query.Where("parent_id = ?", parentID)
	}

	// --- 4. 执行查询并返回结果 ---
	var nodes []model.DomainNode
	if err := query.Order("node_type, title").Find(&nodes).Error; err != nil {
		log.Printf("Error finding domain nodes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve domain content"})
		return
	}

	// GORM 在找不到记录时，Find 不会报错，而是返回一个空切片。
	// 所以这里的检查可以确保前端总是收到一个数组 `[]`，而不是 `null`。
	if nodes == nil {
		nodes = make([]model.DomainNode, 0)
	}

	log.Printf("Found %d nodes.", len(nodes))
	c.JSON(http.StatusOK, nodes)
}

// === Create Domain Node Handler ===
// POST /domains/:id/nodes
func CreateDomainNodeHandler(c *gin.Context) {
	// 从中间件获取圈子信息，c.MustGet 会在 key 不存在时 panic，更安全
	domain := c.MustGet("domain").(model.Domain)
	domainID := domain.ID

	// 复用 CreateNodeInput 结构体
	var input CreateNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果指定了 parent_id，必须验证该父节点存在且属于当前圈子
	if input.ParentID != nil {
		var parentNode model.DomainNode
		// 核心验证：父节点ID存在 且 属于当前 domain_id
		if err := DB.Where("id = ? AND domain_id = ?", *input.ParentID, domainID).First(&parentNode).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Parent folder not found in this domain"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking parent node"})
			return
		}
		// 额外验证：父节点必须是 'folder' 类型
		if parentNode.NodeType != "folder" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot create a node under a text file"})
			return
		}
	}

	// 创建新的 domain_node 实例
	newDomainNode := model.DomainNode{
		DomainID: domainID,
		ParentID: input.ParentID,
		NodeType: input.NodeType,
		Title:    input.Title,
		Content:  input.Content,
	}

	// 如果是 folder，强制清空 content
	if newDomainNode.NodeType == "folder" {
		newDomainNode.Content = ""
	}

	// 保存到数据库
	if err := DB.Create(&newDomainNode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create node in domain"})
		return
	}

	// 返回新创建的节点，状态码 201 Created
	c.JSON(http.StatusCreated, newDomainNode)
}

// === Update Domain Node Handler ===
// PUT /domains/:id/nodes/:nodeId
func UpdateDomainNodeHandler(c *gin.Context) {
	domainID := c.MustGet("domain").(model.Domain).ID
	nodeID := c.Param("nodeId")

	// 复用 UpdateNodeInput 结构体
	var input UpdateNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找节点，确保它在正确的 domain 内
	var node model.DomainNode
	if err := DB.Where("id = ? AND domain_id = ?", nodeID, domainID).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found in this domain"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error finding node"})
		return
	}

	// 应用更新
	if input.Title != nil {
		if strings.TrimSpace(*input.Title) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title cannot be empty"})
			return
		}
		node.Title = *input.Title
	}

	if input.Content != nil {
		if node.NodeType == "folder" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot set content for a folder"})
			return
		}
		node.Content = *input.Content
	}

	// 保存更新
	if err := DB.Save(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update node"})
		return
	}

	c.JSON(http.StatusOK, node)
}

// === Delete Domain Node Handler ===
// DELETE /domains/:id/nodes/:nodeId
func DeleteDomainNodeHandler(c *gin.Context) {
	domainID := c.MustGet("domain").(model.Domain).ID
	nodeID := c.Param("nodeId")

	// GORM 的级联删除 (ON DELETE CASCADE) 会在数据库层面处理子节点的删除
	// 我们只需要删除目标节点即可
	result := DB.Where("id = ? AND domain_id = ?", nodeID, domainID).Delete(&model.DomainNode{})

	// 检查是否有错误
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete node"})
		return
	}

	// 检查是否真的删除了记录
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found in this domain"})
		return
	}

	c.Status(http.StatusNoContent)
}

// === Move Domain Node Handler ===
// PUT /domains/:id/nodes/:nodeId/move
func MoveDomainNodeHandler(c *gin.Context) {
	domainID := c.MustGet("domain").(model.Domain).ID
	nodeIDStr := c.Param("nodeId")
	nodeID, _ := strconv.ParseUint(nodeIDStr, 10, 32)

	// 复用 MoveNodeInput 结构体
	var input MoveNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找要移动的节点，确保它在当前 domain 内
	var nodeToMove model.DomainNode
	if err := DB.Where("id = ? AND domain_id = ?", nodeID, domainID).First(&nodeToMove).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node to move not found in this domain"})
		return
	}

	// 如果目标是某个文件夹 (NewParentID is not null)
	if input.NewParentID != nil {
		newParentID := *input.NewParentID

		// 规则1：不能移动到自身
		if newParentID == uint(nodeID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move a node into itself"})
			return
		}

		// 规则2：验证目标父节点存在于当前 domain 且是文件夹类型
		var targetParent model.DomainNode
		if err := DB.Where("id = ? AND domain_id = ? AND node_type = 'folder'", newParentID, domainID).First(&targetParent).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target folder not found or is not a folder in this domain"})
			return
		}

		// 规则3：【重要】防止将父节点移动到其子孙节点下，避免形成循环
		// 我们可以从 targetParent 向上遍历，检查其祖先链中是否包含 nodeToMove
		currentParentID := targetParent.ParentID
		for currentParentID != nil {
			if *currentParentID == uint(nodeID) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move a parent node into its own descendant"})
				return
			}
			var tempParent model.DomainNode
			if err := DB.Select("parent_id").First(&tempParent, *currentParentID).Error; err != nil {
				break // 到达根节点或出错
			}
			currentParentID = tempParent.ParentID
		}
	}

	// 所有验证通过，更新 ParentID
	nodeToMove.ParentID = input.NewParentID
	if err := DB.Save(&nodeToMove).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node moved successfully"})
}

// === List Recordings for Domain Node Handler ===
// GET /domain-nodes/:nodeId/recordings
func ListRecordingsForDomainNodeHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	nodeID := c.Param("id")

	log.Printf("--- Handling request for Domain Node Recordings ---")
	log.Printf("Attempting to find recordings for nodeId: %s", nodeID)
	log.Printf("Request initiated by userID: %v", userID)

	// 1. 安全检查：用户必须是该节点所在圈子的成员
	var domainNode model.DomainNode
	if err := DB.First(&domainNode, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	var member model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ?", domainNode.DomainID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this domain"})
		return
	}

	// 2. 查询录音
	// 注意：查询条件是 domain_node_id 和 user_id
	// 这确保了用户只能看到自己对这个圈子内容的录音
	var recordings []model.Recording
	if err := DB.Where("domain_node_id = ? AND user_id = ?", nodeID, userID).Order("created_at desc").Find(&recordings).Error; err != nil {
		// 增加错误处理，这是一种好习惯
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve recordings"})
		return
	}
	c.JSON(http.StatusOK, recordings)
}

// GetNodeDetailsHandler 获取单个个人节点的详细信息
func GetNodeDetailsHandler(c *gin.Context) {
	// 1. 从认证中间件中获取当前登录用户的 ID
	userID, exists := c.Get("userID")
	if !exists {
		// 理论上中间件会处理，但作为双重保障
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 2. 从 URL 路径中获取要查询的节点 ID
	nodeID := c.Param("id")

	// 3. 定义一个变量来存储查询结果
	var node model.Node

	// 4. 执行数据库查询
	// 【核心安全逻辑】查询条件必须同时包含 nodeID 和 userID
	// 这可以防止用户通过猜测 ID 来获取不属于自己的节点信息
	if err := DB.Where("id = ? AND user_id = ?", nodeID, userID).First(&node).Error; err != nil {
		// 如果查询出错，判断错误类型
		if err == gorm.ErrRecordNotFound {
			// 这是最常见的情况：节点不存在，或者节点存在但不属于当前用户
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found or permission denied"})
			return
		}
		// 其他类型的数据库错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching node details"})
		return
	}

	// 5. 如果查询成功，返回找到的节点信息
	c.JSON(http.StatusOK, node)
}

func FeatureRecordingHandler(c *gin.Context) {
	recordingID := c.Param("id")
	// Body: { "feature": true }
	var input struct {
		Feature bool `json:"feature"`
	}
	c.ShouldBindJSON(&input)

	result := DB.Model(&model.Recording{}).Where("id = ?", recordingID).Update("is_featured", input.Feature)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Feature status updated"})
}

// LikeRecordingHandler 点赞一个录音
func LikeRecordingHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	recordingID_str := c.Param("id")
	recordingID, _ := strconv.ParseUint(recordingID_str, 10, 32)

	// 使用事务保证数据一致性
	tx := DB.Begin()

	// 1. 尝试插入 like 记录
	newLike := model.Like{
		UserID:      userID.(uint),
		RecordingID: uint(recordingID),
	}
	// ON CONFLICT DO NOTHING 避免重复点赞时报错
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&newLike).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like recording"})
		return
	}

	// 2. 更新 likes_count
	// 使用 gorm.Expr 来执行原子更新操作
	result := tx.Model(&model.Recording{}).Where("id = ?", recordingID).
		Update("likes_count", gorm.Expr("likes_count + ?", 1))

	if result.Error != nil || result.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update likes count"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Liked successfully"})
}

// UnlikeRecordingHandler 取消点赞
func UnlikeRecordingHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	recordingID_str := c.Param("id")
	recordingID, _ := strconv.ParseUint(recordingID_str, 10, 32)

	tx := DB.Begin()

	// 1. 删除 like 记录
	result := tx.Where("user_id = ? AND recording_id = ?", userID, recordingID).Delete(&model.Like{})
	if result.Error != nil {
		tx.Rollback()
		// 即使出错，也可能只是因为记录不存在，所以返回一个通用的成功信息可能更好
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process unlike request"})
		return
	}

	// 如果 RowsAffected 为 0，说明用户本来就没点赞，可以直接返回成功
	if result.RowsAffected == 0 {
		tx.Rollback() // 虽然没有更改，但最好还是回滚
		c.JSON(http.StatusOK, gin.H{"message": "Already unliked"})
		return
	}

	// 2. 更新 likes_count (防止计数为负)
	if err := tx.Model(&model.Recording{}).Where("id = ?", recordingID).
		Update("likes_count", gorm.Expr("GREATEST(0, likes_count - 1)")).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update likes count after unliking"})
		return
	}

	// --- 修正点 ---
	// 提交事务以保存更改
	if err := tx.Commit().Error; err != nil {
		tx.Rollback() // 如果提交失败，也需要回滚
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize unlike operation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unliked successfully"})
}
func ListAllRecordingsForNodeInDomainHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	domainID := c.Param("domainID")
	nodeID := c.Param("nodeID")

	// 1. 【权限检查】验证用户是该圈子的圈主/管理员
	var member model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ? AND role IN ?", domainID, userID, []string{"owner", "admin"}).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// 2. 查询该圈子该节点下的所有录音
	var recordings []model.Recording
	DB.Preload("User").Where("domain_node_id = ?", nodeID).Order("created_at desc").Find(&recordings)
	// 注意：这里需要修改 recordings 表，增加一个 domain_node_id 字段来明确关联

	c.JSON(http.StatusOK, recordings)
}
func FeatureRecordingInDomainHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	recordingID := c.Param("id")
	var input struct {
		Feature bool `json:"feature"`
	}
	c.ShouldBindJSON(&input)

	// 1. 找到这个录音，并确认它属于哪个圈子
	var recording model.Recording
	DB.First(&recording, recordingID)
	if recording.DomainNodeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This recording does not belong to any domain"})
		return
	}

	// 2. 【权限检查】验证操作者是该圈子的圈主/管理员
	var domainNode model.DomainNode
	DB.First(&domainNode, recording.DomainNodeID) // 找到圈子ID
	var member model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ? AND role IN ?", domainNode.DomainID, userID, []string{"owner", "admin"}).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// 3. 更新状态
	DB.Model(&recording).Update("is_domain_featured", input.Feature)
	c.JSON(http.StatusOK, gin.H{"message": "Feature status updated"})
}

type FeaturedRecordingResponse struct {
	// 嵌入原始的 Recording 模型所有字段，例如 ID, CreatedAt, AudioURL 等
	model.Recording

	// 附加的社交信息
	IsLikedByMe bool `json:"is_liked_by_me"`

	// 附加的关联模型信息 (通过 Preload 获得)
	// GORM 的 Preload 会自动填充这些结构体
	// 注意：这里的 User 和 DomainNode 应该只包含公开信息，避免泄露敏感数据
	// 我们可以通过在模型中添加 `json:"-"` 标签来隐藏字段，或者定义专门的 DTO
	User       model.User       `json:"user"`
	DomainNode model.DomainNode `json:"domain_node"`
}

// ListDomainFeaturedRecordingsHandler 获取一个圈子内所有的精选录音
func ListDomainFeaturedRecordingsHandler(c *gin.Context) {
	// --- 1. 获取上下文信息 ---
	// 从认证中间件获取当前登录用户的 ID
	userID_interface, _ := c.Get("userID")
	userID := userID_interface.(uint) // 类型断言为 uint

	// 从 URL 路径中获取圈子 ID
	domainID_str := c.Param("id")
	domainID, err := strconv.ParseUint(domainID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID"})
		return
	}

	// --- 2. 权限检查 ---
	// 验证当前用户是否是该圈子的成员，如果不是，则无权查看
	var member model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ?", domainID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. You are not a member of this domain."})
		return
	}

	// --- 3. 核心数据查询 ---
	// 查询该圈子下所有被标记为 is_domain_featured = true 的录音
	var recordings []model.Recording

	// 使用 Joins 来关联 recordings 和 domain_nodes 表，以便通过 domain_id 进行过滤
	// 使用 Preload 来预加载关联的 User 和 DomainNode 信息，避免 N+1 查询
	if err := DB.
		// 【修改】明确 Select 我们需要的所有字段
		Select("recordings.*"). // 关键：告诉 GORM 我们需要 recordings 表的所有字段
		Joins("JOIN domain_nodes ON domain_nodes.id = recordings.domain_node_id").
		Where("domain_nodes.domain_id = ? AND recordings.is_domain_featured = ?", domainID, true).
		Preload("User").
		Preload("DomainNode").
		Order("recordings.updated_at DESC").
		Find(&recordings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query featured recordings"})
		return
	}

	// 如果没有精选录音，直接返回一个空数组
	if len(recordings) == 0 {
		c.JSON(http.StatusOK, []FeaturedRecordingResponse{})
		return
	}

	// --- 4. 附加社交信息 (is_liked_by_me) ---
	// 这是提升性能的关键一步，避免在循环中查询数据库

	// a. 提取所有录音的 ID
	recordingIDs := make([]uint, len(recordings))
	for i, r := range recordings {
		recordingIDs[i] = r.ID
	}

	// b. 一次性查询当前用户在这些录音中的所有点赞记录
	var userLikes []model.Like
	DB.Where("user_id = ? AND recording_id IN ?", userID, recordingIDs).Find(&userLikes)

	// c. 将点赞记录转换为一个查找效率高的 map (Set)
	likedMap := make(map[uint]bool)
	for _, like := range userLikes {
		likedMap[like.RecordingID] = true
	}

	// --- 5. 组装最终的响应数据 ---
	// 创建一个用于返回的 response 切片
	response := make([]FeaturedRecordingResponse, len(recordings))

	for i, r := range recordings {
		response[i] = FeaturedRecordingResponse{
			Recording:   r,
			IsLikedByMe: likedMap[r.ID], // 从 map 中高效地获取点赞状态
			User:        r.User,         // GORM 已经通过 Preload 填充了
			DomainNode:  r.DomainNode,   // GORM 已经通过 Preload 填充了
		}
	}

	// --- 6. 返回结果 ---
	c.JSON(http.StatusOK, response)
}

// ListFeaturedRecordingsForNode 获取单个圈子节点下的所有精选录音
func ListFeaturedRecordingsForNode(c *gin.Context) {
	// --- 1. 获取参数并进行基本验证 ---
	userID_interface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userID_interface.(uint) // 从中间件获取 userID

	nodeID_str := c.Param("nodeId")
	nodeID, err := strconv.ParseUint(nodeID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID format"})
		return
	}

	// --- 2. 权限检查: 验证用户有权访问此节点 ---
	var node model.DomainNode
	// a. 检查节点是否存在
	if err := DB.First(&node, uint(nodeID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while fetching node"})
		return
	}
	// b. 检查用户是否是该节点所属圈子的成员
	var member model.DomainMember
	if err := DB.Where("domain_id = ? AND user_id = ?", node.DomainID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: you are not a member of this domain"})
		return
	}

	// --- 3. 查询该节点下的所有精选录音 ---
	var recordings []model.Recording
	err = DB.Where("domain_node_id = ? AND is_domain_featured = ?", nodeID, true).
		Preload("User").           // 预加载录音的作者信息
		Order("likes_count DESC"). // 按点赞数降序排序
		Find(&recordings).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve featured recordings"})
		return
	}

	// 如果没有精选录音，直接返回空数组
	if len(recordings) == 0 {
		c.JSON(http.StatusOK, []RecordingWithLikeStatus{})
		return
	}

	// --- 4. 【核心】附加 is_liked_by_me 状态 (N+1 查询优化) ---

	// a. 收集所有录音的 ID
	recordingIDs := make([]uint, len(recordings))
	for i, r := range recordings {
		recordingIDs[i] = r.ID
	}

	// b. 一次性查询当前用户在这些录音中的所有点赞记录
	var userLikes []model.Like
	DB.Where("user_id = ? AND recording_id IN ?", userID, recordingIDs).Find(&userLikes)

	// c. 将点赞记录转换为一个 Map，便于快速查找
	likedMap := make(map[uint]bool)
	for _, like := range userLikes {
		likedMap[like.RecordingID] = true
	}

	// --- 5. 组装最终的响应数据 ---
	response := make([]RecordingWithLikeStatus, 0, len(recordings))
	for _, r := range recordings {
		response = append(response, RecordingWithLikeStatus{
			Recording: r,
			User: AuthorResponse{ // 只返回安全的作者信息
				ID:       r.User.ID,
				Username: r.User.Username,
			},
			IsLikedByMe: likedMap[r.ID], // 从 Map 中高效获取点赞状态
		})
	}

	c.JSON(http.StatusOK, response)
}

// CreateCommentHandler 发表新评论
func CreateCommentHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	recordingID_str := c.Param("id")
	recordingID, _ := strconv.ParseUint(recordingID_str, 10, 32)

	var input CreateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil { /* ... */
	}

	// 检查录音是否存在
	var recording model.Recording
	if err := DB.First(&recording, uint(recordingID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found"})
		return
	}

	newComment := model.Comment{
		RecordingID: uint(recordingID),
		UserID:      userID.(uint),
		Content:     input.Content,
	}

	tx := DB.Begin()
	// 1. 创建评论
	if err := tx.Create(&newComment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post comment"})
		return
	}
	// 2. 更新评论计数
	tx.Model(&model.Recording{}).Where("id = ?", recordingID).
		Update("comments_count", gorm.Expr("comments_count + 1"))

	tx.Commit()

	// 返回新创建的评论，并附带作者信息
	DB.Preload("User").First(&newComment, newComment.ID)

	response := CommentResponse{
		Comment: newComment,
		User:    AuthorResponse{ID: newComment.User.ID, Username: newComment.User.Username},
	}

	c.JSON(http.StatusCreated, response)
}

// ListCommentsHandler 获取评论列表
func ListCommentsHandler(c *gin.Context) {
	recordingID := c.Param("id")

	var comments []model.Comment
	DB.Where("recording_id = ?", recordingID).
		Preload("User").         // 预加载作者信息
		Order("created_at ASC"). // 按时间正序排列
		Find(&comments)

	// 组装安全的 Response DTO
	response := make([]CommentResponse, len(comments))
	for i, comment := range comments {
		response[i] = CommentResponse{
			Comment: comment,
			User:    AuthorResponse{ID: comment.User.ID, Username: comment.User.Username},
		}
	}

	c.JSON(http.StatusOK, response)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// === FollowUserHandler 关注一个用户 ===
func FollowUserHandler(c *gin.Context) {
	// 1. 获取 ID
	// a. followerID 是当前操作者，从中间件获取
	followerID_interface, _ := c.Get("userID")
	followerID := followerID_interface.(uint)

	// b. followingID 是被关注者，从 URL 参数获取
	followingID_str := c.Param("id")
	followingID_uint64, err := strconv.ParseUint(followingID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	followingID := uint(followingID_uint64)

	// 2. 业务规则校验
	if followerID == followingID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot follow yourself"})
		return
	}

	// 3. 【核心】使用数据库事务执行操作
	err = DB.Transaction(func(tx *gorm.DB) error {
		// a. 创建关注关系记录
		follow := model.Follower{
			FollowerID:  followerID,
			FollowingID: followingID,
		}
		// 使用 OnConflict Do Nothing 避免重复关注时报错，实现幂等性
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&follow).Error; err != nil {
			return err // 返回错误，事务将回滚
		}

		// 如果上面 Create 成功 (RowsAffected > 0)，才更新计数
		if tx.RowsAffected > 0 {
			// b. 增加关注者的 following_count
			if err := tx.Model(&model.User{}).Where("id = ?", followerID).UpdateColumn("following_count", gorm.Expr("following_count + 1")).Error; err != nil {
				return err
			}
			// c. 增加被关注者的 followers_count
			if err := tx.Model(&model.User{}).Where("id = ?", followingID).UpdateColumn("followers_count", gorm.Expr("followers_count + 1")).Error; err != nil {
				return err
			}
		}

		return nil // 返回 nil，事务将提交
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Followed successfully"})
}

// === UnfollowUserHandler 取消关注一个用户 ===
func UnfollowUserHandler(c *gin.Context) {
	// 1. 获取 ID (同上)
	followerID_interface, _ := c.Get("userID")
	followerID := followerID_interface.(uint)
	followingID_str := c.Param("id")
	followingID_uint64, _ := strconv.ParseUint(followingID_str, 10, 32)
	followingID := uint(followingID_uint64)

	// 2. 使用事务执行操作
	err := DB.Transaction(func(tx *gorm.DB) error {
		// a. 删除关注关系记录
		result := tx.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&model.Follower{})
		if result.Error != nil {
			return result.Error
		}

		// 只有在确实删除了记录 (之前是关注状态) 的情况下，才更新计数
		if result.RowsAffected > 0 {
			// b. 减少关注者的 following_count
			if err := tx.Model(&model.User{}).Where("id = ?", followerID).UpdateColumn("following_count", gorm.Expr("GREATEST(0, following_count - 1)")).Error; err != nil {
				return err
			}
			// c. 减少被关注者的 followers_count
			if err := tx.Model(&model.User{}).Where("id = ?", followingID).UpdateColumn("followers_count", gorm.Expr("GREATEST(0, followers_count - 1)")).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfollow user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unfollowed successfully"})
}

// GetUserProfileHandler 获取用户公开信息
func GetUserProfileHandler(c *gin.Context) {
	// a. 获取目标用户的 ID
	targetUserID_str := c.Param("id")
	targetUserID, _ := strconv.ParseUint(targetUserID_str, 10, 32)

	// b. 查找目标用户
	var userProfile model.User
	if err := DB.Select("id", "username", "created_at", "followers_count", "following_count").First(&userProfile, uint(targetUserID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// c. 检查当前访问者与目标用户的关注关系 (如果已登录)
	isFollowing := false
	visitorID_interface, exists := c.Get("userID") // c.Get() 不会报错，只会返回 nil 和 false
	if exists {
		visitorID := visitorID_interface.(uint)
		var follow model.Follower
		// 尝试查找关注记录
		if err := DB.Where("follower_id = ? AND following_id = ?", visitorID, targetUserID).First(&follow).Error; err == nil {
			isFollowing = true
		}
	}

	// d. 组装并返回数据
	c.JSON(http.StatusOK, gin.H{
		"id":                userProfile.ID,
		"username":          userProfile.Username,
		"created_at":        userProfile.CreatedAt,
		"followers_count":   userProfile.FollowersCount,
		"following_count":   userProfile.FollowingCount,
		"is_followed_by_me": isFollowing, // 【关键】返回关注状态
	})
}

// ListPostsHandler 获取帖子列表，支持分页和排序
func ListPostsHandler(c *gin.Context) {
	// 1. 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// 2. 获取排序参数
	sortBy := c.DefaultQuery("sort_by", "last_replied_at")
	order := "DESC" // 默认降序

	// 构建排序字符串，防止SQL注入
	// 只允许按特定字段排序
	allowedSorts := map[string]bool{
		"last_replied_at": true,
		"created_at":      true,
		"views_count":     true,
	}
	if !allowedSorts[sortBy] {
		sortBy = "last_replied_at" // 如果参数不合法，使用默认值
	}

	// last_replied_at 为 NULL 的帖子应该排在后面
	orderString := sortBy + " " + order + " NULLS LAST"

	// 3. 查询数据库
	var posts []model.Post
	err := DB.
		Preload("User").              // 预加载作者信息
		Preload("LastRepliedByUser"). // 预加载最后回复者信息
		Order(orderString).
		Offset(offset).
		Limit(limit).
		Find(&posts).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve posts"})
		return
	}

	// （可选但推荐）查询总数以方便前端分页
	var total int64
	DB.Model(&model.Post{}).Count(&total)

	// 4. 组装安全的 DTO 返回
	type PostListResponse struct {
		// 嵌入 Post 模型的所有字段
		model.Post
		// 覆盖 User 和 LastRepliedByUser 字段，使用安全的 AuthorResponse
		User              AuthorResponse  `json:"User"`
		LastRepliedByUser *AuthorResponse `json:"LastRepliedByUser,omitempty"`
	}

	response := make([]PostListResponse, len(posts))
	for i, p := range posts {
		respItem := PostListResponse{
			Post: p,
			User: AuthorResponse{ID: p.User.ID, Username: p.User.Username},
		}
		// 因为 LastRepliedByUser 可能为 nil，需要检查
		if p.LastRepliedByUser.ID != 0 {
			respItem.LastRepliedByUser = &AuthorResponse{
				ID:       p.LastRepliedByUser.ID,
				Username: p.LastRepliedByUser.Username,
			}
		}
		response[i] = respItem
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"limit": limit,
		"posts": response,
	})
}

// CreatePostHandler 创建一个新帖子
func CreatePostHandler(c *gin.Context) {
	userID_interface, _ := c.Get("userID")
	userID := userID_interface.(uint)

	var input CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newPost := model.Post{
		UserID:  userID,
		Title:   input.Title,
		Content: input.Content,
		// LastRepliedAt 默认设置为帖子的创建时间，这样新帖子会排在前面
		LastRepliedAt:       &time.Time{}, // 需要初始化指针
		LastRepliedByUserID: &userID,
	}
	*newPost.LastRepliedAt = time.Now()

	if err := DB.Create(&newPost).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	// 返回新创建的帖子详情
	DB.Preload("User").First(&newPost, newPost.ID)

	// 同样使用 DTO 返回
	response := PostListResponse{
		Post: newPost,
		User: AuthorResponse{ID: newPost.User.ID, Username: newPost.User.Username},
		// 注意：新帖子的 LastRepliedByUser 就是作者自己
		LastRepliedByUser: &AuthorResponse{ID: newPost.User.ID, Username: newPost.User.Username},
	}

	c.JSON(http.StatusCreated, response)
}

// GetPostHandler 获取单个帖子的详情，并增加浏览量
func GetPostHandler(c *gin.Context) {
	postID_str := c.Param("id")
	postID, err := strconv.ParseUint(postID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// --- 1. 使用事务来获取帖子并增加浏览量 ---
	var post model.Post
	err = DB.Transaction(func(tx *gorm.DB) error {
		// a. 查找帖子，预加载作者信息
		if err := tx.Preload("User").First(&post, uint(postID)).Error; err != nil {
			return err // 如果找不到，事务会回滚，外部会捕获到 gorm.ErrRecordNotFound
		}

		// b. 原子性地增加浏览量
		if err := tx.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve post"})
		}
		return
	}

	// --- 2. 获取该帖子的第一页回复 ---
	var replies []model.Reply
	DB.Where("post_id = ?", postID).
		Preload("User").
		Order("created_at ASC").
		Limit(20). // 默认加载第一页，每页20条
		Find(&replies)

	// --- 3. 组装安全的 DTO 返回 ---
	// (复用之前定义的 DTO 结构)

	// a. 组装帖子的 DTO
	postResponse := PostListResponse{ // PostListResponse 刚好适用
		Post: post,
		User: AuthorResponse{ID: post.User.ID, Username: post.User.Username},
	}
	if post.LastRepliedByUser.ID != 0 {
		postResponse.LastRepliedByUser = &AuthorResponse{
			ID:       post.LastRepliedByUser.ID,
			Username: post.LastRepliedByUser.Username,
		}
	}

	repliesResponse := make([]ReplyResponse, len(replies))
	for i, r := range replies {
		repliesResponse[i] = ReplyResponse{
			Reply: r,
			User:  AuthorResponse{ID: r.User.ID, Username: r.User.Username},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"post":    postResponse,
		"replies": repliesResponse,
	})
}

// CreateReplyHandler 为一个帖子创建新回复
func CreateReplyHandler(c *gin.Context) {
	userID_interface, _ := c.Get("userID")
	userID := userID_interface.(uint)

	postID_str := c.Param("id")
	postID, _ := strconv.ParseUint(postID_str, 10, 32)

	var input CreateReplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// --- 使用事务来创建回复并更新帖子信息 ---
	var newReply model.Reply
	err := DB.Transaction(func(tx *gorm.DB) error {
		// a. 检查帖子是否存在
		var post model.Post
		if err := tx.First(&post, uint(postID)).Error; err != nil {
			return errors.New("post not found") // 返回一个自定义错误
		}

		// b. 创建回复记录
		newReply = model.Reply{
			PostID:  uint(postID),
			UserID:  userID,
			Content: input.Content,
		}
		if err := tx.Create(&newReply).Error; err != nil {
			return err
		}

		// c. 更新帖子的计数和最后回复信息
		now := time.Now()
		updateData := map[string]interface{}{
			"replies_count":           gorm.Expr("replies_count + 1"),
			"last_replied_at":         &now,
			"last_replied_by_user_id": userID,
		}
		if err := tx.Model(&model.Post{}).Where("id = ?", postID).Updates(updateData).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if err.Error() == "post not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reply"})
		}
		return
	}

	// --- 返回新创建的回复（带作者信息）---
	DB.Preload("User").First(&newReply, newReply.ID)

	// (使用之前定义的 ReplyResponse DTO)
	response := ReplyResponse{
		Reply: newReply,
		User:  AuthorResponse{ID: newReply.User.ID, Username: newReply.User.Username},
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyProfileHandler 获取当前登录用户自己的信息
func GetMyProfileHandler(c *gin.Context) {
	userID_interface, _ := c.Get("userID")
	userID := userID_interface.(uint)

	var user model.User
	// 只选择需要的、安全的字段返回
	if err := DB.Select("id", "username", "created_at", "role", "followers_count", "following_count").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user) // 直接返回 user 对象
}

// checkDomainMembership 检查一个用户是否是指定圈子的成员
// 如果是，则返回成员信息；如果不是，则返回错误。
func checkDomainMembership(db *gorm.DB, userID uint, domainID uint) (*model.DomainMember, error) {
	var member model.DomainMember
	if err := db.Where("domain_id = ? AND user_id = ?", domainID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("access denied: you are not a member of this domain")
		}
		return nil, err // 其他数据库错误
	}
	return &member, nil
}

// CreateDomainNodeCommentHandler 为圈子内容节点创建新评论
func CreateDomainNodeCommentHandler(c *gin.Context) {
	// a. 获取参数
	userID_interface, _ := c.Get("userID")
	userID := userID_interface.(uint)

	nodeID_str := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID format"})
		return
	}

	// b. 绑定输入
	var input CreateCommentInput // 复用之前的 CreateCommentInput 结构体
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// c. 【核心】权限与存在性检查
	var node model.DomainNode
	if err := DB.Select("domain_id").First(&node, uint(nodeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Domain node not found"})
		return
	}
	if _, err := checkDomainMembership(DB, userID, node.DomainID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// d. 创建评论并更新计数（使用事务）
	var newComment model.DomainNodeComment
	err = DB.Transaction(func(tx *gorm.DB) error {
		// 1. 创建评论记录
		newComment = model.DomainNodeComment{
			DomainNodeID: uint(nodeID),
			UserID:       userID,
			Content:      input.Content,
		}
		if err := tx.Create(&newComment).Error; err != nil {
			return err
		}

		// 2. 更新 domain_nodes 表的 comments_count
		if err := tx.Model(&model.DomainNode{}).Where("id = ?", nodeID).UpdateColumn("comments_count", gorm.Expr("comments_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	// e. 返回新创建的评论（带作者信息）
	DB.Preload("User").First(&newComment, newComment.ID)

	// 使用 DTO 组装安全的响应数据
	type DomainNodeCommentResponse struct {
		model.DomainNodeComment
		User AuthorResponse `json:"User"`
	}
	response := DomainNodeCommentResponse{
		DomainNodeComment: newComment,
		User:              AuthorResponse{ID: newComment.User.ID, Username: newComment.User.Username},
	}

	c.JSON(http.StatusCreated, response)
}

// ListDomainNodeCommentsHandler 获取指定圈子内容节点的所有评论
func ListDomainNodeCommentsHandler(c *gin.Context) {
	// a. 获取参数
	userID_interface, _ := c.Get("userID")
	userID := userID_interface.(uint)

	nodeID_str := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID format"})
		return
	}

	// b. 【核心】权限与存在性检查
	var node model.DomainNode
	if err := DB.Select("domain_id").First(&node, uint(nodeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Domain node not found"})
		return
	}
	if _, err := checkDomainMembership(DB, userID, node.DomainID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// c. 查询评论列表
	var comments []model.DomainNodeComment
	DB.Where("domain_node_id = ?", nodeID).
		Preload("User").
		Order("created_at ASC"). // 按时间正序排列，旧的在上面
		Find(&comments)

	// d. 组装安全的 DTO 列表返回
	type DomainNodeCommentResponse struct {
		model.DomainNodeComment
		User AuthorResponse `json:"User"`
	}
	response := make([]DomainNodeCommentResponse, len(comments))
	for i, comment := range comments {
		response[i] = DomainNodeCommentResponse{
			DomainNodeComment: comment,
			User:              AuthorResponse{ID: comment.User.ID, Username: comment.User.Username},
		}
	}

	c.JSON(http.StatusOK, response)
}

func main() {
	// 读取 JWT Secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	jwtKey = []byte(secret)
	// 1. 初始化数据库
	initDB()
	initMinIO()
	mqManager = mq.NewRabbitMQManager("audio_processing")

	// 2. 创建 Gin 引擎
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		// 将原始请求的详细信息打印到日志
		dump, err := httputil.DumpRequest(c.Request, true)
		if err != nil {
			fmt.Println("Error dumping request:", err)
		} else {
			fmt.Printf("--- INCOMING REQUEST ---\n%s\n----------------------\n", string(dump))
		}
		c.Next() // 继续处理请求
	})
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.MaxAge = 12 * time.Hour

	r.Use(cors.New(config))

	// 设置路由
	apiV1 := r.Group("/api/v1")
	{
		// --- 公开路由 ---
		apiV1.POST("/register", RegisterHandler)
		apiV1.POST("/login", LoginHandler)

		apiV1.GET("/posts", ListPostsHandler)
		apiV1.GET("/posts/:id", GetPostHandler)
		// --- 所有需要登录的路由都放在这个组里 ---
		auth := apiV1.Group("/")
		auth.Use(middleware.AuthMiddleware()) // 应用普通用户认证
		{
			// === 个人资源 ===
			auth.POST("/users/:id/follow", FollowUserHandler)
			auth.DELETE("/users/:id/follow", UnfollowUserHandler)
			auth.GET("/users/:id", GetUserProfileHandler)
			auth.GET("/profile", GetMyProfileHandler)
			//文章
			auth.POST("/posts", CreatePostHandler)
			auth.POST("/posts/:id/replies", CreateReplyHandler)
			// 个人内容节点 (Nodes)
			auth.GET("/nodes", ListNodesHandler)
			auth.POST("/nodes", CreateNodeHandler)
			auth.GET("/nodes/search", SearchNodesHandler)
			auth.GET("/nodes/:id", GetNodeDetailsHandler)
			auth.PUT("/nodes/:id", UpdateNodeHandler)
			auth.DELETE("/nodes/:id", DeleteNodeHandler)
			auth.PUT("/nodes/:id/move", MoveNodeHandler)
			auth.GET("/nodes/:id/recordings", ListRecordingsForNodeHandler) // 获取个人节点下的个人录音

			// 个人录音 (Recordings)
			auth.GET("/recordings", ListMyRecordingsHandler)
			auth.POST("/recordings/upload", UploadHandler)
			auth.PUT("/recordings/:id", UpdateRecordingHandler)
			auth.DELETE("/recordings/:id", DeleteRecordingHandler)
			auth.POST("/recordings/:id/transcribe", TranscribeRecordingHandler)

			// --- 社交互动 (作用于录音) ---
			// 【补全】点赞功能路由
			auth.POST("/recordings/:id/like", LikeRecordingHandler)
			auth.DELETE("/recordings/:id/like", UnlikeRecordingHandler)
			auth.POST("/recordings/:id/comments", CreateCommentHandler)
			auth.GET("/recordings/:id/comments", ListCommentsHandler)

			// --- AI 助手 ---
			auth.POST("/ai/chat", AIChatHandler)

			// === 圈子资源 (Domains) ===
			// 圈子本身的创建、加入、列表
			auth.POST("/domains", CreateDomainHandler)
			auth.POST("/domains/join", JoinDomainHandler) // 注意：这个 join 的逻辑可能需要调整
			auth.GET("/domains/my", ListMyDomainsHandler)
			auth.GET("/domain-nodes/:id/recordings", ListRecordingsForDomainNodeHandler)
			auth.POST("/domain-nodes/:id/comments", CreateDomainNodeCommentHandler)
			auth.GET("/domain-nodes/:id/comments", ListDomainNodeCommentsHandler)
			// --- 单个圈子内部的资源 ---
			// 凡是涉及到具体某个圈子内部的操作，都放在 /domains/:domainId 下
			domainSpecific := auth.Group("/domains/:domainId")

			{
				// 成员即可访问的
				domainSpecific.GET("/details", GetDomainDetailsHandler)
				domainSpecific.GET("/nodes", ListDomainNodesHandler) // 获取圈子内容树

				// 【补全】获取圈子所有精选录音
				domainSpecific.GET("/featured-recordings", ListDomainFeaturedRecordingsHandler)

				// 【补全】获取单个圈子节点下的精选录音
				// 注意：这里的 :nodeId 指的是 domain_node_id
				domainSpecific.GET("/nodes/:nodeId/featured-recordings", ListFeaturedRecordingsForNode)

				// 【补全】获取单个圈子节点下所有成员的录音 (供圈主审核)
				// 这个需要圈主权限，我们在 Handler 内部检查
				domainSpecific.GET("/nodes/:nodeId/all-recordings", ListAllRecordingsForNodeInDomainHandler)

				// --- 圈主才能管理的内容 ---
				domainContent := domainSpecific.Group("/nodes")
				domainContent.Use(DomainOwnerMiddleware()) // 对这个组应用圈主权限
				{
					domainContent.POST("", CreateDomainNodeHandler)
					domainContent.PUT("/:nodeId", UpdateDomainNodeHandler)
					domainContent.DELETE("/:nodeId", DeleteDomainNodeHandler)
					domainContent.PUT("/:nodeId/move", MoveDomainNodeHandler)
				}

				// --- 圈主发布内容到圈子 ---
				// 这个也需要圈主权限，可以在Handler内部检查，或者也用中间件
				domainSpecific.POST("/publish", PublishNodeToDomainHandler)
			}

			// 【补全】圈主精选录音的操作
			// 这个操作不依赖于 domainId，而是直接作用于 recordingId，但权限需要验证
			// 所以放在 auth 根级别下
			auth.POST("/recordings/:id/feature-in-domain", FeatureRecordingInDomainHandler)
		}
	}

	r.Run(":8080")
}
