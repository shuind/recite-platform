package handler

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// UploadHandler 处理器结构体，包含 MinIO 客户端
type UploadHandler struct {
	MinioClient *minio.Client
}

// NewUploadHandler 构造函数
func NewUploadHandler(minioClient *minio.Client) *UploadHandler {
	return &UploadHandler{MinioClient: minioClient}
}

// HandleFileUpload 是一个通用的文件上传函数
func (h *UploadHandler) HandleFileUpload(c *gin.Context) {
	// ... 这部分代码与上一版完全相同，无需修改 ...
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file is received"})
		return
	}

	fileType, isValid := validateFileType(file)
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only images (jpg, png, gif) and videos (mp4, webm) are allowed."})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer src.Close()

	url, err := h.uploadToMinio(src, file.Size, file.Filename, fileType)
	if err != nil {
		log.Printf("ERROR: Failed to upload to MinIO: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to storage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"url":     url,
	})
}

// uploadToMinio 封装了上传到 MinIO 的核心逻辑
func (h *UploadHandler) uploadToMinio(src multipart.File, size int64, filename, contentType string) (string, error) {
	// ... 这部分代码与上一版完全相同，无需修改 ...
	bucketName := os.Getenv("MINIO_BUCKET_NAME")
	publicEndpoint := os.Getenv("MINIO_PUBLIC_ENDPOINT")
	objectName := uuid.New().String() + filepath.Ext(filename)

	_, err := h.MinioClient.PutObject(context.Background(), bucketName, objectName, src, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	fileURL := fmt.Sprintf("%s/%s/%s", publicEndpoint, bucketName, objectName)
	return fileURL, nil
}

// validateFileType 校验文件类型
func validateFileType(file *multipart.FileHeader) (string, bool) {
	// ... 这部分代码与上一版完全相同，无需修改 ...
	contentType := file.Header.Get("Content-Type")
	allowedImages := map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true}
	allowedVideos := map[string]bool{"video/mp4": true, "video/webm": true}

	if allowedImages[contentType] || allowedVideos[contentType] {
		return contentType, true
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".mp4" || ext == ".webm" {
		return contentType, true
	}

	return "", false
}
