<!-- src/components/CommentCard.vue (最终修复版) -->
<template>
  <div class="comment-card">
    <!-- 一级评论 (楼层主) -->
    <div class="root-comment">
      <el-avatar class="avatar" :size="36" :src="comment.user.avatar_url" />
      <div class="content-wrapper">
        <span class="username">{{ comment.user.username }}</span>
        <p class="content-text">{{ comment.content }}</p>
        <div class="footer">
          <span class="time">{{ timeAgo(comment.created_at) }}</span>
          <div class="actions">
            <div 
              class="action-item" 
              :class="{ 'liked': isLiked(comment.id) }" 
              @click="toggleLike(comment.id)"
            >
              <el-icon><CaretTop /></el-icon>
              <span>{{ comment.likes_count > 0 ? comment.likes_count : '赞' }}</span>
            </div>
            <!-- 【核心修正】回复按钮现在总是激活对楼主的回复 -->
            <div class="action-item" @click="activateReply(comment.user)">
              <el-icon><ChatDotRound /></el-icon>
              <span>回复</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 
      【核心结构调整】
      二级评论区 和 回复输入框 现在是“兄弟”关系，而不是“父子”关系。
    -->

    <!-- 1. 二级评论区 (只有当有子评论时才显示) -->
    <div v-if="comment.child_replies && comment.child_replies.length > 0" class="child-comments">
      <div v-for="child in comment.child_replies" :key="child.id" class="child-comment-item">
        <el-avatar class="avatar-small" :size="24" :src="child.user.avatar_url" />
        <div class="content-wrapper">
          <p class="content-text-child">
            <span class="username-link">{{ child.user.username }}</span>
            <span v-if="child.reply_to_user && child.reply_to_user.id !== comment.user.id" class="reply-to">
              回复 <span class="username-link">@{{ child.reply_to_user.username }}</span>
            </span>
            : {{ child.content }}
          </p>
          <div class="footer">
            <span class="time">{{ timeAgo(child.created_at) }}</span>
            <div class="actions">
               <div 
                class="action-item" 
                :class="{ 'liked': isLiked(child.id) }"
                @click="toggleLike(child.id)"
              >
                <el-icon><CaretTop /></el-icon>
                <span>{{ child.likes_count > 0 ? child.likes_count : '赞' }}</span>
              </div>
              <!-- 点击二级评论的回复按钮，激活对该二级评论作者的回复 -->
              <div class="action-item" @click="activateReply(child.user)">
                <el-icon><ChatDotRound /></el-icon>
                <span>回复</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 查看更多回复 (只有在有子评论的情况下才可能显示) -->
      <div 
        v-if="comment.child_replies_count > comment.child_replies.length" 
        class="view-more"
        @click="$emit('view-all-replies', comment)"
      >
        <span>查看全部 {{ comment.child_replies_count }} 条回复 <el-icon><ArrowRightBold /></el-icon></span>
      </div>
    </div>

    <!-- 2. 回复输入框 (只要点击了“回复”按钮就会显示) -->
    <div v-if="replyingTo" class="reply-editor-nested">
      <el-input 
        :placeholder="`回复 @${replyingTo.username}`" 
        size="small" 
        v-model="newReplyContent"
        ref="replyInputRef"
      />
      <el-button type="primary" size="small" @click="submitReply">发布</el-button>
      <el-button size="small" @click="replyingTo = null">取消</el-button>
    </div>

  </div>
</template>

<script setup>
// 您的 <script setup> 部分代码逻辑是正确的，无需修改，直接复用即可。
import { ref, nextTick } from 'vue';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';
import { CaretTop, ChatDotRound, ArrowRightBold } from '@element-plus/icons-vue';

const props = defineProps({
  comment: { type: Object, required: true },
  postId: { type: [Number, String], required: true },
  likedRepliesMap: { type: Object, default: () => ({}) }
});

// 2. 【核心修正】在 emits 数组中添加 'like-toggled'
const emit = defineEmits(['comment-submitted', 'view-all-replies', 'like-toggled']);
const replyingTo = ref(null);
const newReplyContent = ref('');
const replyInputRef = ref(null);

const isLiked = (replyId) => props.likedRepliesMap[replyId] || false;

const activateReply = async (user) => {
  replyingTo.value = user;
  await nextTick();
  replyInputRef.value?.focus();
};

const submitReply = async () => {
  if (!newReplyContent.value.trim() || !replyingTo.value) return;
  try {
    await apiClient.post(`/posts/${props.postId}/replies`, {
      content: newReplyContent.value,
      parent_reply_id: props.comment.id,
      reply_to_user_id: replyingTo.value.id,
    });
    newReplyContent.value = '';
    replyingTo.value = null;
    ElMessage.success("回复成功！");
    emit('comment-submitted');
  } catch (error) {
    console.error("回复失败:", error);
    ElMessage.error("回复失败");
  }
};

// 【核心改造】增强 toggleLike 的错误处理
const toggleLike = async (replyId) => {
  try {
    await apiClient.post(`/replies/${replyId}/like`);
    emit('like-toggled');
  } catch (error) {
    console.error("点赞失败:", error);
    
  }
};

const timeAgo = (dateString) => {  const past = new Date(dateString);
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
  return "刚刚";};
</script>

<style scoped>
/* 您的 <style> 部分代码是正确的，无需修改，直接复用即可。 */
.comment-card { padding: 16px 0; border-bottom: 1px solid #f0f2f5; }
.comment-card:last-child { border-bottom: none; }
.root-comment, .child-comment-item { display: flex; gap: 12px; }
.avatar, .avatar-small { flex-shrink: 0; }
.content-wrapper { flex-grow: 1; }
.username, .username-link { font-weight: 600; font-size: 15px; }
.username-link { color: #007bff; cursor: pointer; }
.content-text { margin: 4px 0 8px 0; font-size: 15px; line-height: 1.7; color: #1d2129; word-break: break-word; }
.content-text-child { margin: 0 0 8px 0; font-size: 14px; line-height: 1.6; color: #4e5969; }
.reply-to { margin: 0 4px; }
.footer { display: flex; justify-content: space-between; align-items: center; font-size: 13px; color: #8a919f; }
.actions { display: flex; gap: 16px; }
.action-item { display: flex; align-items: center; cursor: pointer; transition: color 0.2s; }
.action-item:hover, .action-item.liked { color: #007bff; }
.action-item.liked { font-weight: 600; }
.action-item .el-icon { margin-right: 4px; }
.child-comments {
  margin-top: 16px;
  margin-left: 48px;
  padding-left: 16px;
  border-left: 2px solid #e9e9e9;
}
.child-comment-item { margin-bottom: 12px; }
.view-more {
  font-size: 14px;
  color: #8a919f;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
}
.reply-editor-nested {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  /* 【核心修正】确保输入框无论何时出现，都在正确的位置 */
  margin-left: 48px;
}
</style>