<template>
  <div class="create-post-page">
    <el-card class="editor-card">
      <template #header>
        <div class="card-header">
          <!-- 【修改】标题根据是否在编辑模式动态变化 -->
          <span>{{ isEditing ? '编辑帖子' : '发布新帖子' }}</span>
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
          <el-input 
            v-model="postForm.content"
            type="textarea"
            :rows="15"
            placeholder="在这里输入你的想法、问题或分享..."
          />
        </el-form-item>
        
        <el-form-item>
          <!-- 【修改】将原来的 submitPost 改为 handlePublish -->
          <el-button type="primary" @click="handlePublish" :loading="isSubmitting">
            立即发布
          </el-button>
          <!-- 【新增】存为草稿按钮 -->
          <el-button @click="handleSaveDraft" :loading="isSaving">
            存为草稿
          </el-button>
          <el-button @click="goBack">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed } from 'vue'; // 【新增】引入 onMounted, computed
import { useRoute, useRouter } from 'vue-router'; // 【新增】引入 useRoute
import apiClient from '@/api';
import { ElMessage } from 'element-plus';

const route = useRoute(); // 【新增】
const router = useRouter();

const postForm = reactive({
  title: '',
  content: '',
});

const isSubmitting = ref(false);
const isSaving = ref(false); // 【新增】
const postId = ref(null); // 【新增】用于存储当前编辑的帖子ID

// 【新增】一个计算属性，用于判断当前是否处于编辑模式
const isEditing = computed(() => !!postId.value);

// 【新增】生命周期钩子，在组件加载时检查路由
onMounted(async () => {
  // 如果 URL 中包含 id 参数 (例如 /posts/edit/123)，说明是编辑模式
  if (route.params.id) {
    postId.value = route.params.id;
    try {
      // 从后端加载该草稿的现有内容
      const response = await apiClient.get(`/posts/${postId.value}`);
      // 将内容填充到表单中
      postForm.title = response.data.post.title;
      postForm.content = response.data.post.content;
    } catch (error) {
      console.error("加载草稿失败:", error);
      ElMessage.error("加载草稿失败，请重试");
      router.push('/forum/create-post'); // 加载失败则跳转回新建页面
    }
  }
});


// 【重构】处理“发布”操作
const handlePublish = async () => {
  if (!postForm.title.trim() || !postForm.content.trim()) {
    ElMessage.warning('标题和内容都不能为空');
    return;
  }
  
  isSubmitting.value = true;
  try {
    if (isEditing.value) {
      // 如果是编辑模式，调用 PUT 方法更新并发布
      await apiClient.put(`/posts/${postId.value}`, { ...postForm, status: 'published' });
    } else {
      // 如果是新建模式，调用 POST 方法创建并发布
      await apiClient.post('/posts', { ...postForm, status: 'published' });
    }
    ElMessage.success('发布成功！');
    router.push({ name: 'forum' });
  } catch (error) {
    console.error("发布时发生错误:", error.response || error);
    ElMessage.error('发布失败，请稍后重试');
  } finally {
    isSubmitting.value = false;
  }
};

// 【新增】处理“存为草稿”操作
const handleSaveDraft = async () => {
  if (!postForm.title.trim() && !postForm.content.trim()) {
    ElMessage.warning('标题和内容至少需要填写一项才能保存');
    return;
  }
  isSaving.value = true;
  try {
    if (isEditing.value) {
      // 如果是编辑模式，调用 PUT 方法更新草稿
      await apiClient.put(`/posts/${postId.value}`, postForm);
    } else {
      // 如果是新建模式，调用 POST 方法创建新草稿
      await apiClient.post('/posts', { ...postForm, status: 'draft' });
    }
    ElMessage.success('草稿已保存！');
    // 保存后可以跳转到草稿箱页面
    router.push({ name: 'DraftsPage' }); // 假设您的草稿箱路由名叫 'Drafts'
  } catch (error) {
    console.error("保存草稿时发生错误:", error.response || error);
    ElMessage.error('保存失败，请稍后重试');
  } finally {
    isSaving.value = false;
  }
};


const goBack = () => {
  router.back();
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