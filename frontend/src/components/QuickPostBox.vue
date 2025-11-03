<template>
  <el-card class="quick-post-box" shadow="never">
    <div class="input-section">
      <el-avatar class="user-avatar" :size="40" src="https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png" />
      <el-input
        ref="postInputRef"
        v-model="newPost.content"
        class="post-input"
        type="textarea"
        :autosize="{ minRows: 2, maxRows: 6 }"
        placeholder="分享此刻的想法..."
      />
    </div>

    <!-- 媒体预览区 -->
    <div v-if="newPost.image_url || newPost.video_url" class="media-preview">
      <img v-if="newPost.image_url" :src="newPost.image_url" alt="图片预览"/>
      <video v-if="newPost.video_url" :src="newPost.video_url" controls></video>
      <el-button class="remove-media-btn" type="danger" :icon="Close" circle @click="removeMedia" />
    </div>

    <div class="actions-section">
      <div class="action-icons">
        <el-tooltip content="添加话题" placement="top">
          <span @click="addTopic"><el-icon><PriceTag /></el-icon> #</span>
        </el-tooltip>
        
        <!-- 【核心修改】使用 el-upload 组件来处理图片和视频的上传 -->
        <el-upload
          :action="uploadURL"
          :show-file-list="false"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :before-upload="beforeUpload"
          :headers="uploadHeaders"
          name="file"
        >
          <el-tooltip content="上传图片或视频" placement="top">
            <!-- 将图片和视频图标合并为一个上传触发器 -->
            <span class="upload-trigger">
              <el-icon><Picture /></el-icon> / <el-icon><VideoCamera /></el-icon>
            </span>
          </el-tooltip>
        </el-upload>
      </div>
      <el-button class="post-button" type="primary" @click="handleSubmit" :loading="isSubmitting" :disabled="!newPost.content.trim()">发想法</el-button>
    </div>
  </el-card>
</template>

<script setup>
import { ref, computed } from 'vue';
import { PriceTag, Picture, VideoCamera, Close } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import apiClient from '@/api'; // 确保您有一个统一的 apiClient 实例

const emit = defineEmits(['post-created']);

const newPost = ref({
  content: '',
  post_type: 'thought',
  image_url: '', // 字段名与 Go 模型对应
  video_url: '', // 字段名与 Go 模型对应
});
const isSubmitting = ref(false);
const postInputRef = ref(null);

// 定义上传API的地址
const uploadURL = '/api/v1/uploads/file'; // 指向您在 Go 后端创建的通用上传接口

// 如果上传需要认证，动态计算请求头
const uploadHeaders = computed(() => {
  const token = localStorage.getItem('token'); // 从 localStorage 或其他地方获取 token
  return token ? { 'Authorization': `Bearer ${token}` } : {};
});

// 上传成功的回调函数
const handleUploadSuccess = (response, uploadFile) => {
  removeMedia(); // 清除旧的媒体文件
  const url = response.url;
  const contentType = uploadFile.raw.type;

  // 根据文件的 MIME 类型，将返回的 URL 存到对应的字段中
  if (contentType.startsWith('image/')) {
    newPost.value.image_url = url;
  } else if (contentType.startsWith('video/')) {
    newPost.value.video_url = url;
  }
  ElMessage.success('文件上传成功!');
};

// 上传失败的回调函数
const handleUploadError = (error) => {
    ElMessage.error('文件上传失败，请检查文件大小或网络。');
    console.error("Upload Error:", error);
}

// 上传前的客户端校验
const beforeUpload = (rawFile) => {
  const isImage = rawFile.type.startsWith('image/');
  const isVideo = rawFile.type.startsWith('video/');
  
  if (!isImage && !isVideo) {
    ElMessage.error('只能上传图片 (JPG, PNG, GIF) 或视频 (MP4, WEBM) 文件!');
    return false;
  }
  
  const isLt20M = rawFile.size / 1024 / 1024 < 20; // 示例：限制为 20MB
  if (!isLt20M) {
    ElMessage.error('文件大小不能超过 20MB!');
    return false;
  }
  return true;
};

// 提交整个帖子的函数
const handleSubmit = async () => {
  if (!newPost.value.content.trim()) return;
  isSubmitting.value = true;
  try {
    const response = await apiClient.post('/posts', newPost.value); // 假设创建帖子的API是这个
    ElMessage.success('发布成功!');
    emit('post-created', response.data);
    
    // 重置表单
    newPost.value.content = '';
    newPost.value.image_url = '';
    newPost.value.video_url = '';
  } catch (error) {
    console.error("发布失败:", error.response?.data || error.message);
    ElMessage.error(error.response?.data?.error || '发布失败，请稍后再试');
  } finally {
    isSubmitting.value = false;
  }
};

// 移除已添加的媒体
const removeMedia = () => {
  newPost.value.image_url = '';
  newPost.value.video_url = '';
};

// 添加话题符号
const addTopic = () => {
  postInputRef.value?.focus();
  newPost.value.content += ' #在此输入话题# ';
};
</script>

<style scoped>
/* 您的样式 + 针对上传触发器的一些微调 */
.quick-post-box { /* ... */ }
.input-section { /* ... */ }
.user-avatar { /* ... */ }
.post-input { /* ... */ }
.actions-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
}
.action-icons {
  display: flex;
  align-items: center;
  color: #8a919f;
  gap: 20px;
  font-size: 18px;
  margin-left: 55px;
}
.action-icons span {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
}
.upload-trigger { /* 让图标看起来更像一个整体的按钮 */
    display: inline-flex;
    align-items: center;
    gap: 5px;
}
.post-button { /* ... */ }
.media-preview {
  position: relative;
  margin-top: 15px;
  margin-left: 55px;
  width: 150px;
  height: 150px;
  border: 1px dashed #dcdfe6;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.media-preview img, .media-preview video {
  max-width: 100%;
  max-height: 100%;
}
.remove-media-btn {
  position: absolute;
  top: -10px;
  right: -10px;
}
</style>