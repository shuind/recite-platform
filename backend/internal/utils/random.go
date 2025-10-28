package utils

import (
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Go 1.20 之前，需要手动设置种子
// var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// Go 1.20+ rand.NewSource is deprecated and rand is safe for concurrent use.
// We'll write it in a way that's simple and modern.

// GenerateRandomString 创建一个指定长度的随机字符串
func GenerateRandomString(length int) string {
	// 使用新的 rand.NewSource 以确保每次程序启动时的随机性
	// 对于高并发场景，可以考虑创建一个全局或池化的 rand.Rand 实例
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
