<!-- src/components/PostCard.vue (已修正 API 地址) -->
<template>
  <div class="post-card" @click="goToPostDetail">
    
    <div class="post-header">
      <el-avatar class="post-avatar" :size="24" :src="post.user.avatar_url" />
      <span class="post-author">{{ post.user.username }}</span>
      <span class="post-meta">· {{ timeAgo(post.created_at) }}</span>
    </div>

    <div class="post-body" :class="{ 'with-image': post.image_url }">
      <div v-if="post.image_url" class="image-container">
        <img :src="post.image_url" alt="帖子封面" class="post-thumbnail"/>
      </div>
      <div class="content-container">
        <h3 class="post-title">{{ post.title }}</h3>
        <p class="post-excerpt" v-html="truncatedContent"></p>
      </div>
    </div>

    <div class="post-footer">
      <div 
        class="action-item" 
        :class="{ 'liked': post.is_liked_by_me }"
        @click.stop="handleLike"
      >
        <el-icon><CaretTop /></el-icon>
        <span>赞同 {{ post.votes_count || 0 }}</span>
      </div>
      <div class="action-item" @click.stop="goToComments">
        <el-icon><ChatDotRound /></el-icon>
        <span>{{ post.replies_count || 0 }} 条评论</span>
      </div>
      <div class="action-item" @click.stop>
        <el-icon><View /></el-icon>
        <span>{{ post.views_count || 0 }} 次浏览</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { View, ChatDotRound, CaretTop } from '@element-plus/icons-vue';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';

const emit = defineEmits(['update:like']);
const props = defineProps({
  post: { type: Object, required: true }
});
const router = useRouter();

const truncatedContent = computed(() => {
  const content = props.post.content || props.post.excerpt || '';
  const maxLength = 100;
  if (content.length > maxLength) {
    return content.substring(0, maxLength) + `... <span class="read-more">阅读全文</span>`;
  }
  return content;
});

const handleLike = async () => {
  const originalLiked = props.post.is_liked_by_me;
  const originalVotes = props.post.votes_count;
  const newLikedStatus = !originalLiked;
  const newVotesCount = originalLiked ? originalVotes - 1 : originalVotes + 1;

  emit('update:like', { 
    postId: props.post.id, 
    is_liked_by_me: newLikedStatus,
    votes_count: newVotesCount,
  });

  try {
    // =========================================================
    // 【核心修正】将 API 地址改回您原来的 /like
    // =========================================================
    await apiClient.post(`/posts/${props.post.id}/like`);
    
  } catch (error) {
    console.error("点赞操作失败:", error);
    ElMessage.error("操作失败，请稍后重试");
    emit('update:like', {
      postId: props.post.id,
      is_liked_by_me: originalLiked,
      votes_count: originalVotes,
    });
  }
};

const goToPostDetail = () => router.push(`/posts/${props.post.id}`);
const goToComments = () => router.push(`/posts/${props.post.id}#comments`);

const timeAgo = (dateString) => {
  const past = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now - past) / 1000);
  let interval = seconds / 31536000;
  if (interval > 1) return Math.floor(interval) + " 年前";
  interval = seconds / 2592000;
  if (interval > 1) return Math.floor(interval) + " 个月前";
  interval = seconds / 86400;
  if (interval > 1) return Math.floor(interval) + " 天前";
  interval = seconds / 3600;
  if (interval > 1) return Math.floor(interval) + " 小时前";
  interval = seconds / 60;
  if (interval > 1) return Math.floor(interval) + " 分钟前";
  return "刚刚";
};
</script>

<style scoped>
/* 样式部分无需修改，保持原样即可 */
.post-card { background-color: #fff; padding: 16px 20px; border-bottom: 1px solid #e9e9e9; cursor: pointer; transition: background-color 0.2s; }
.post-card:hover { background-color: #fafafa; }
.post-header { display: flex; align-items: center; margin-bottom: 12px; font-size: 14px; }
.post-avatar { margin-right: 8px; }
.post-author { font-weight: 500; color: #4e5969; }
.post-meta { color: #8a919f; margin-left: 8px; }
.post-body { display: flex; margin-bottom: 12px; }
.post-body.with-image { flex-direction: row; gap: 16px; }
.content-container { flex-grow: 1; min-width: 0; }
.image-container { width: 150px; height: 100px; flex-shrink: 0; border-radius: 4px; overflow: hidden; background-color: #f0f2f5; }
.post-thumbnail { width: 100%; height: 100%; object-fit: cover; }
.post-title { font-size: 17px; font-weight: 600; color: #1d2129; margin: 0 0 4px 0; }
.post-excerpt { color: #4e5969; font-size: 14px; line-height: 1.7; margin: 0; }
:deep(.read-more) { color: #007bff; margin-left: 4px; }
.post-footer { display: flex; align-items: center; color: #8a919f; font-size: 14px; gap: 24px; }
.action-item { display: flex; align-items: center; cursor: pointer; transition: color 0.2s; }
.action-item .el-icon { margin-right: 6px; }
.action-item:hover { color: #007bff; }
.action-item.liked { color: #007bff; font-weight: 600; }
</style>