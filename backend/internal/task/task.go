package task

// AudioProcessingTask 是我们放入消息队列的任务定义
type AudioProcessingTask struct {
	RecordingID      uint   `json:"recording_id"`
	UserID           uint   `json:"user_id"`
	FileContent      []byte `json:"-"` // 文件内容，JSON中忽略
	OriginalFileName string `json:"original_file_name"`
}

// in your worker or a shared types package
type AudioProcessingPayload struct {
	RecordingID uint   `json:"recording_id"`
	UserID      uint   `json:"user_id"`
	FileContent []byte `json:"file_content"` // 在简单场景下，直接传递字节
	ContentType string `json:"content_type"`
	FileExt     string `json:"file_ext"`
}
