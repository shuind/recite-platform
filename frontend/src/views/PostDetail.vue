<!-- src/views/PostDetail.vue (最终完整增强版) -->
<template>
  <div class="post-detail-page">
    <!-- 【新增】页面头部，包含返回按钮 -->
    <div class="page-header">
      <el-button type="info" :icon="ArrowLeftBold" @click="goBack" link>
        返回
      </el-button>
    </div>

    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="10" animated />
    </div>

    <div v-else-if="post" class="post-detail-container">
      <!-- 左侧主内容区 -->
      <div class="main-content">
        <el-card class="post-card-full" shadow="never">
          <h1 class="post-title-detail">{{ post.title }}</h1>
          <div class="author-info-bar">
            <el-avatar :size="48" :src="post.user.avatar_url" />
            <div class="author-details">
              <span class="author-name">{{ post.user.username }}</span>
              <span class="author-bio">这里是用户的简介或签名</span>
            </div>
            <el-button 
                v-if="loggedInUserId && post.user.id !== loggedInUserId"
                :type="isFollowingUser ? 'info' : 'primary'"
                @click="handleFollowUser"
                :plain="isFollowingUser"
                round
              >
                <el-icon v-if="!isFollowingUser"><Plus /></el-icon>
                {{ isFollowingUser ? '已关注' : '关注' }}
            </el-button>
          </div>
          
          <div class="post-content-full" v-html="post.content"></div>
          <div v-if="post.image_url" class="post-image-full">
            <img :src="post.image_url" alt="帖子图片">
          </div>

          <div class="post-footer">
            <div class="action-item" :class="{ 'liked': post.is_liked_by_me }" @click="handleLike">
              <el-icon><CaretTop /></el-icon>
              <span>赞同 {{ post.votes_count }}</span>
            </div>
            <div v-if="post.post_type === 'question'" class="action-item" :class="{ 'followed': post.is_followed_by_me }" @click="handleFollowQuestion">
              <el-icon><Star /></el-icon>
              <span>{{ post.is_followed_by_me ? '已关注' : '关注问题' }}</span>
            </div>
            <div v-if="post.post_type === 'question'" class="action-item" @click="showAnswerEditor = !showAnswerEditor">
              <el-icon><EditPen /></el-icon>
              <span>写回答</span>
            </div>
          </div>

          <div v-if="showAnswerEditor" class="answer-editor">
            <el-input
              v-model="newAnswerContent"
              type="textarea"
              :rows="5"
              placeholder="写下你的回答..."
            />
            <div class="answer-editor-actions">
              <el-button type="primary" @click="submitAnswer" :loading="isSubmittingAnswer">提交回答</el-button>
            </div>
          </div>
        </el-card>

        <!-- 【新增】评论发布区 -->
        <el-card v-if="post.post_type !== 'question'" class="reply-editor-card" shadow="never">
          <div class="editor-header">
            <el-avatar :size="40" :src="loggedInUserAvatar" />
            <div class="editor-input-wrapper">
              <el-input
                v-model="newReplyContent"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 5 }"
                placeholder="写下你的评论..."
              />
            </div>
          </div>
          <div class="editor-footer">
            <el-button type="primary" @click="submitReply" :loading="isSubmittingReply">发布评论</el-button>
          </div>
        </el-card>

        <!-- 【核心改造】评论列表区 -->
        <el-card class="comments-section" shadow="never" id="replies">
            <h4>{{ post.replies_count }} 条评论</h4>
            <div v-if="replies && replies.length > 0">
              <!-- 【核心改造】监听 CommentCard 发出的 @view-all-replies 事件 -->
              <CommentCard 
                v-for="reply in replies" 
                :key="reply.id" 
                :comment="reply"
                :post-id="post.id"
                :liked-replies-map="likedRepliesMap"
                @comment-submitted="fetchPostDetails"
                @view-all-replies="openReplyDialog"
                @like-toggled="fetchPostDetails"
              />
            </div>
           <el-empty v-else description="还没有评论，快来抢沙发吧！"></el-empty>
        </el-card>
      </div>

      <!-- 右侧侧边栏 -->
      <div class="sidebar">
        <AuthorInfoCard 
          :author="post.user" 
          :is-followed="isFollowingUser"
          @follow-toggle="handleFollowUser"
        />
      </div>
    </div>
    <!-- 【新增】将 ReplyDialog 组件放在这里 -->
    <ReplyDialog 
      v-if="viewingParentComment" 
      :key="viewingParentComment.id"
      :parent-comment="viewingParentComment"
      v-model:visible="isReplyDialogOpen"
      @main-page-refresh-needed="fetchPostDetails"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';
import { CaretTop, Star, EditPen, Plus, ArrowLeftBold } from '@element-plus/icons-vue';
import AuthorInfoCard from '@/components/AuthorInfoCard.vue';
import CommentCard from '@/components/CommentCard.vue'; // 确保引入的是新的 CommentCard
import ReplyDialog from '@/components/ReplyDialog.vue'; // 【核心】引入弹窗组件

// 1. 初始化路由
const route = useRoute();
const router = useRouter();

// 2. 定义所有需要的响应式状态
const post = ref(null);
const replies = ref([]); // 这个数组现在直接接收后端返回的、带嵌套子评论的数组
const answers = ref([]);
const likedRepliesMap = ref({}); // 存储所有评论的点赞状态
const loading = ref(true);
// --- 【新增】用于控制弹窗的状态 ---
const isReplyDialogOpen = ref(false);
const viewingParentComment = ref(null); // 存储要查看的楼层主评论对象

// 【核心改造】openReplyDialog
const openReplyDialog = (parentComment) => {
  viewingParentComment.value = parentComment;
  isReplyDialogOpen.value = true;
};  

import { watch } from 'vue';

watch(isReplyDialogOpen, (newValue) => {
  if (!newValue) {
    // 当弹窗关闭时，重置正在查看的评论，以确保组件下次能被正确销毁和重建
    viewingParentComment.value = null;
  }
});
// 交互状态
const isFollowingUser = ref(false);
const showAnswerEditor = ref(false);
const newAnswerContent = ref('');
const isSubmittingAnswer = ref(false);

const newReplyContent = ref('');
const isSubmittingReply = ref(false);

// 3. 获取当前登录用户信息
const loggedInUser = JSON.parse(localStorage.getItem('user'));
const loggedInUserId = computed(() => (loggedInUser ? loggedInUser.id : null));
const loggedInUserAvatar = computed(() => (loggedInUser ? loggedInUser.avatar_url : ''));

// 4. 定义所有函数方法

// 返回上一页
const goBack = () => {
  router.back();
};

// 获取帖子所有数据 (核心)
const fetchPostDetails = async () => {
  loading.value = true;
  try {
    const response = await apiClient.get(`/posts/${route.params.id}`);
    post.value = response.data.post;
    
    // 直接使用后端返回的结构化数据
    replies.value = response.data.replies || []; 
    answers.value = response.data.answers || [];
    
    // 从后端获取点赞状态图
    likedRepliesMap.value = response.data.liked_replies_map || {};

    // 假设后端能在 post.user 中返回 is_followed_by_me
    isFollowingUser.value = response.data.post.user.is_followed_by_me || false;

  } catch (error) {
    console.error("获取帖子详情失败:", error);
    ElMessage.error("帖子加载失败，请刷新重试");
  } finally {
    loading.value = false;
  }
};

// 帖子点赞
const handleLike = async () => {
  if (!post.value) return;
  const originalState = post.value.is_liked_by_me;
  post.value.is_liked_by_me = !originalState;
  post.value.votes_count += originalState ? -1 : 1;
  try {
    await apiClient.post(`/posts/${post.value.id}/like`);
  } catch (error) {
    post.value.is_liked_by_me = originalState;
    post.value.votes_count += originalState ? 1 : -1;
    console.error("帖子点赞失败:", error);
    ElMessage.error("操作失败");
  }
};

// 关注/取关用户
const handleFollowUser = async () => {
  if (!post.value) return;
  const originalState = isFollowingUser.value;
  isFollowingUser.value = !originalState;
  try {
    if (originalState) {
      await apiClient.delete(`/users/${post.value.user.id}/follow`);
      if (post.value.user.followers_count > 0) post.value.user.followers_count--;
    } else {
      await apiClient.post(`/users/${post.value.user.id}/follow`);
      post.value.user.followers_count++;
    }
  } catch (error) {
    isFollowingUser.value = originalState;
    console.error("关注/取关用户失败:", error);
    ElMessage.error("操作失败");
  }
};

// 关注/取关问题
const handleFollowQuestion = async () => {
  if (!post.value || post.value.post_type !== 'question') return;
  const originalState = post.value.is_followed_by_me;
  post.value.is_followed_by_me = !originalState;
  post.value.followers_count += originalState ? -1 : 1;
  try {
    await apiClient.post(`/posts/${post.value.id}/follow`);
  } catch (error) {
    post.value.is_followed_by_me = originalState;
    post.value.followers_count += originalState ? 1 : -1;
    console.error("关注/取关问题失败:", error);
    ElMessage.error("操作失败");
  }
};

// 提交回答 (针对问题)
const submitAnswer = async () => {
  if (!newAnswerContent.value.trim()) { ElMessage.warning("回答内容不能为空"); return; }
  isSubmittingAnswer.value = true;
  try {
    await apiClient.post(`/posts/${post.value.id}/answers`, { content: newAnswerContent.value });
    ElMessage.success("回答成功！");
    showAnswerEditor.value = false;
    newAnswerContent.value = '';
    await fetchPostDetails(); // 刷新数据
  } catch (error) {
    console.error("提交回答失败:", error);
    ElMessage.error("回答失败");
  } finally {
    isSubmittingAnswer.value = false;
  }
};

// 提交一级评论 (针对文章/想法)
const submitReply = async () => {
  if (!newReplyContent.value.trim()) { ElMessage.warning("评论内容不能为空"); return; }
  isSubmittingReply.value = true;
  try {
    await apiClient.post(`/posts/${post.value.id}/replies`, {
      content: newReplyContent.value,
      parent_reply_id: null // 明确这是一级评论
    });
    ElMessage.success("评论发布成功！");
    newReplyContent.value = '';
    await fetchPostDetails(); // 刷新数据
  } catch (error) {
    console.error("提交评论失败:", error);
    ElMessage.error("评论失败，请稍后重试");
  } finally {
    isSubmittingReply.value = false;
  }
};

// 5. 调用 onMounted 生命周期钩子
onMounted(fetchPostDetails);
</script>

<style scoped>
.post-detail-page {
  background-color: #f7f8fa;
  padding: 20px 0;
}

/* 【新增】页面头部样式 */
.page-header {
  max-width: 1000px;
  margin: 0 auto 16px auto;
  padding: 0 10px;
  display: flex;
  align-items: center;
}
.page-header .el-button {
  font-size: 15px;
}

.post-detail-container {
  display: flex;
  max-width: 1000px;
  margin: 0 auto;
  gap: 20px;
  align-items: flex-start;
}
.main-content {
  flex: 1;
  min-width: 0;
}
.sidebar {
  width: 300px;
  position: sticky;
  top: 80px;
}
.loading-state {
  max-width: 700px;
  margin: 20px auto;
}
.post-card-full, .comments-section, .reply-editor-card {
  border: none;
  border-radius: 4px;
  background-color: #fff;
  padding: 24px;
}
.post-title-detail {
  font-size: 28px;
  font-weight: 700;
  color: #1d2129;
  margin: 0 0 24px 0;
  line-height: 1.4;
}
.author-info-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.author-details {
  display: flex;
  flex-direction: column;
  flex-grow: 1;
}
.author-name {
  font-weight: 600;
  color: #1d2129;
}
.author-bio {
  font-size: 14px;
  color: #8a919f;
  margin-top: 2px;
}
.post-content-full {
  font-size: 16px;
  line-height: 1.8;
  color: #4e5969;
  word-break: break-word;
}
.post-content-full :deep(p) {
  margin-bottom: 1em;
}
.post-image-full {
  margin: 12px 0;
}
.post-image-full img {
  max-width: 25%;
  border-radius: 4px;
}
.post-footer {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #e9e9e9;
}
.action-item {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  background-color: #f2f3f5;
  padding: 8px 16px;
  border-radius: 20px;
  font-size: 14px;
  color: #4e5969;
  transition: all 0.2s;
}
.action-item:hover {
  background-color: #e5e6eb;
}
.action-item .el-icon {
  margin-right: 6px;
}
.action-item.liked {
  color: #fff;
  background-color: #007bff;
}
.action-item.liked:hover {
  background-color: #0056b3;
}
.action-item.followed {
  color: #fff;
  background-color: #E6A23C;
}
.action-item.followed:hover {
  background-color: #cf9236;
}
.answer-editor {
  margin-top: 24px;
  border-top: 1px solid #e9e9e9;
  padding-top: 20px;
}
.answer-editor-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

/* 【新增】评论编辑器样式 */
.reply-editor-card {
  margin-top: 20px;
}
.editor-header {
  display: flex;
  gap: 16px;
}
.editor-input-wrapper {
  flex: 1;
}
.editor-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.comments-section {
  margin-top: 20px;
}
.comments-section h4 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 20px 0;
  border-bottom: 1px solid #e9e9e9;
  padding-bottom: 16px;
}
.item-card {
  padding: 16px 0;
  border-bottom: 1px solid #f0f2f5;
}
.item-card:last-child {
  border-bottom: none;
}
.item-author {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.item-author-name {
  font-size: 15px;
  font-weight: 500;
}
.item-content {
  font-size: 15px;
  line-height: 1.7;
  color: #4e5969;
}
</style>