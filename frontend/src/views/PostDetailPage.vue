<template>
  <div v-if="loading" class="loading-state">加载中...</div>
  
  <!-- 主内容区 -->
  <div class="post-detail-page" v-else-if="post">
    
    <!-- 1. 帖子正文卡片 -->
    <el-card class="post-content-card">
      <!-- 修正: post.title -->
      <h1>{{ post.title }}</h1>
      <div class="post-meta">
        作者: 
        <!-- 修正: post.user.id, post.user.username -->
        <router-link :to="{ name: 'profile', params: { userId: post.user.id } }">
          {{ post.user.username }}
        </router-link> 
        <!-- 修正: post.created_at, post.views_count -->
        | 发布于: {{ new Date(post.created_at).toLocaleString() }}
        | {{ post.views_count }} 次浏览
      </div>
      <!-- 修正: post.content -->
      <div class="content-body" v-html="post.content"></div>
    </el-card>

    <!-- 2. 回复列表区域 -->
    <div class="replies-section">
      <!-- 修正: post.replies_count -->
      <h3>{{ post.replies_count }} 条回复</h3>
      
      <!-- 修正: key="reply.id" -->
      <div v-for="reply in replies" :key="reply.id" class="reply-item">
        <div class="reply-meta">
          <!-- 修正: reply.user.id, reply.user.username -->
          <router-link :to="{ name: 'profile', params: { userId: reply.user.id } }">{{ reply.user.username }}</router-link>
          
          <!-- 修正: reply.created_at -->
          <span> 回复于 {{ new Date(reply.created_at).toLocaleString() }}</span>
        </div>
        
        <!-- 修正: reply.content -->
        <div class="reply-content">{{ reply.content }}</div>
      </div>
    </div>
    
    
  </div>
  <!-- 3. 回复编辑器 -->
  <div class="sticky-reply-container" v-if="post">
    <el-card class="reply-editor-card">
      <el-input 
        v-model="newReplyContent" 
        type="textarea" 
        :rows="3"  
        placeholder="发表你的看法..." 
        resize="none"
      />
      <el-button 
        type="primary" 
        @click="submitReply" 
        :loading="replyLoading" 
        style="margin-top: 10px;"
      >
        发表回复
      </el-button>
    </el-card>
  </div>


  <!-- 加载失败或帖子不存在时的状态 -->
  <el-empty v-else description="帖子不存在或加载失败"></el-empty>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';

const route = useRoute();
const postId = route.params.postId;

const post = ref(null);
const replies = ref([]);
const loading = ref(true);
const replyLoading = ref(false);
const newReplyContent = ref('');

onMounted(async () => {
  loading.value = true;
  try {
    const response = await apiClient.get(`/posts/${postId}`);
    // 后端返回的 post 和 replies 对象的字段名都是小写/蛇形
    // 这里赋值是正确的
    post.value = response.data.post;
    replies.value = response.data.replies;
  } catch (error) {
    ElMessage.error("帖子加载失败");
    console.error("Failed to fetch post details:", error);
  } finally {
    loading.value = false;
  }
});

const submitReply = async () => {
  if (!newReplyContent.value.trim()) {
    ElMessage.warning("回复内容不能为空");
    return;
  }
  replyLoading.value = true;
  try {
    const response = await apiClient.post(`/posts/${postId}/replies`, {
      content: newReplyContent.value,
    });
    
    // API 返回的单个 reply 对象的字段名也是小写/蛇形
    // 所以直接 push 是正确的
    replies.value.push(response.data);

    // script 中访问 post.replies_count 也是正确的
    if (post.value) post.value.replies_count++;

    newReplyContent.value = '';
    ElMessage.success('回复成功！');
  } catch (error) {
    ElMessage.error('回复失败');
  } finally {
    replyLoading.value = false;
  }
};
</script>

<style scoped>
.post-detail-page {
  padding: 20px;
  max-width: 900px;
  margin: 0 auto;
  /* 【关键】为粘性编辑器留出空间，防止遮挡最后的内容 */
  padding-bottom: 150px; /* 这个值约等于编辑器的高度 + 一些间距 */
}
/* 2. 定义粘性容器的样式 */
.sticky-reply-container {
  /* 【关键】粘性定位 */
  position: sticky;
  bottom: 0;
  
  /* 为了美观，让它也居中并匹配主内容区的宽度 */
  max-width: 900px;
  margin: 0 auto;
  padding: 0 20px; /* 匹配 .post-detail-page 的 padding */
  
  /* 层级和背景 */
  z-index: 10;
  background-color: #fff; /* 需要一个背景色来遮挡下方滚动的内容 */
}
.post-content-card { margin-bottom: 30px; }
.post-meta { color: #909399; font-size: 14px; margin: 15px 0; }
.post-meta a { color: #606266; font-weight: 500; text-decoration: none; }
.post-meta a:hover { text-decoration: underline; }
.content-body { line-height: 1.8; font-size: 16px; margin-top: 20px; }
.replies-section { margin-bottom: 30px; }
.reply-item { padding: 15px; border-bottom: 1px solid #f0f2f5; }
.reply-meta { font-size: 12px; color: #909399; margin-bottom: 8px; }
.reply-meta a { font-weight: 600; color: #606266; text-decoration: none; }
.reply-meta a:hover { text-decoration: underline; }
.reply-content { font-size: 14px; line-height: 1.6; }
/* 3. 调整编辑器卡片的样式 */
/* 3. 调整编辑器卡片的样式 */
.reply-editor-card {
  /* 去掉默认的 margin */
  margin-bottom: 0;
  
  /* 保留顶部分隔线，这有助于视觉区分 */
  border-top: 1px solid #e4e7ed;
  
  /* box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.05); <-- 移除或注释掉此行 */

  /* 圆角可以保留，也可以去掉，取决于你想要的效果 */
  border-radius: 8px 8px 0 0; 
}
</style>