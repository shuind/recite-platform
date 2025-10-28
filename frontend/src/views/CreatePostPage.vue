<template>
  <div class="create-post-page">
    <el-card class="editor-card">
      <template #header>
        <div class="card-header">
          <span>发布新帖子</span>
        </div>
      </template>
      
      <el-form :model="postForm" label-position="top">
        <el-form-item label="帖子标题">
          <el-input 
            v-model="postForm.title" 
            placeholder="请输入一个吸引人的标题"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item label="帖子内容">
          <!-- 未来这里可以替换为富文本编辑器 -->
          <el-input 
            v-model="postForm.content"
            type="textarea"
            :rows="15"
            placeholder="在这里输入你的想法、问题或分享..."
          />
        </el-form-item>
        
        <el-form-item>
          <el-button type="primary" @click="submitPost" :loading="isSubmitting">
            立即发布
          </el-button>
          <el-button @click="goBack">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';

const router = useRouter();
const postForm = reactive({
  title: '',
  content: '',
});
const isSubmitting = ref(false);

const submitPost = async () => {
  if (!postForm.title.trim() || !postForm.content.trim()) {
    ElMessage.warning('标题和内容都不能为空');
    return;
  }
  
  isSubmitting.value = true;
  try {
    const response = await apiClient.post('/posts', postForm);
    ElMessage.success('帖子发布成功！');
    
    // 发布成功后，跳转到新帖子的详情页
    const newPostId = response.data.Post.id; // 根据你后端返回的结构获取ID
    router.push({ name: 'post-detail', params: { postId: newPostId } });

  } catch (error) {
    ElMessage.error('发布失败，请稍后重试');
  } finally {
    isSubmitting.value = false;
  }
};

const goBack = () => {
  router.back(); // 返回上一页
};
</script>

<style scoped>
.create-post-page {
  max-width: 900px;
  margin: 20px auto;
  padding: 20px;
}
.editor-card {
  border: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}
.card-header {
  font-size: 20px;
  font-weight: 600;
}
</style>