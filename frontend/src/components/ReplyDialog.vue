<!-- src/components/ReplyDialog.vue (最终优化版) -->
<template>
  <el-dialog
    v-model="isDialogVisible"
    :title="dialogTitle"
    width="600px"
    @closed="reset"
    class="reply-dialog"
  >
    <!-- 【修改】给弹窗内容区加上 v-loading 指令 -->
    <div v-loading="loading">
      <!-- 1. 楼层主评论 -->
      <div v-if="parentComment" class="parent-comment-wrapper">
        <div class="root-comment">
          <el-avatar class="avatar" :size="36" :src="parentComment.user.avatar_url" />
          <div class="content-wrapper">
            <span class="username">{{ parentComment.user.username }}</span>
            <p class="content-text">{{ parentComment.content }}</p>
          </div>
        </div>
      </div>
      <el-divider />
      
      <!-- 2. 所有子评论列表 -->
      <div class="child-replies-list">
        <!-- 【修改】用 template 包裹，以正确显示加载和空状态 -->
        <template v-if="!loading">
          <div v-if="childReplies.length > 0">
            <div v-for="child in childReplies" :key="child.id" class="child-comment-item">
              <el-avatar class="avatar-small" :size="24" :src="child.user.avatar_url" />
              <div class="content-wrapper">
                <p class="content-text-child">
                  <span class="username-link">{{ child.user.username }}</span>
                  <span v-if="child.reply_to_user && child.reply_to_user.id !== parentComment.user.id" class="reply-to">
                    回复 <span class="username-link">@{{ child.reply_to_user.username }}</span>
                  </span>
                  : {{ child.content }}
                </p>
                <div class="footer">
                  <span class="time">{{ timeAgo(child.created_at) }}</span>
                  <!-- 【新增】互动按钮 -->
                  <div class="actions">
                    <div class="action-item" :class="{ 'liked': isLiked(child.id) }" @click="toggleLike(child.id)">
                      <el-icon><CaretTop /></el-icon>
                      <span>{{ child.likes_count > 0 ? child.likes_count : '赞' }}</span>
                    </div>
                    <div class="action-item" @click="activateReply(child.user)">
                      <el-icon><ChatDotRound /></el-icon>
                      <span>回复</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无回复"></el-empty>
        </template>
      </div>
       <!-- 【新增】弹窗底部的回复输入框 -->
      <div class="dialog-reply-editor" v-if="replyingTo">
        <el-input 
          :placeholder="`回复 @${replyingTo.username}`" 
          size="small" 
          v-model="newReplyContent"
        />
        <el-button type="primary" size="small" @click="submitReply">发布</el-button>
        <el-button size="small" @click="replyingTo = null">取消</el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, onMounted, computed, defineProps, defineEmits } from 'vue'; // 【核心】引入 onMounted
import apiClient from '@/api';
import { ElMessage } from 'element-plus';
import { CaretTop, ChatDotRound } from '@element-plus/icons-vue';

// 【核心改造】移除 visible prop
const props = defineProps({
  parentComment: { type: Object, required: true },
  visible: { type: Boolean, default: false } // 接收 visible prop
});

// 【核心修改】创建一个计算属性来代理 prop，从而实现 v-model
const isDialogVisible = computed({
  get() {
    return props.visible;
  },
  set(value) {
    emit('update:visible', value); // 当弹窗关闭时，通知父组件
    if (!value) {
      emit('close'); // 保留原有的 close 事件逻辑
    }
  }
});
// 【核心修正】将 reset 函数加回来
const reset = () => {
  childReplies.value = [];
  likedMap.value = {};
};
// 【核心改造】移除 update:visible，新增 close
const emit = defineEmits(['close', 'main-page-refresh-needed', 'update:visible']); // 
// --- 状态定义 (保持不变) ---
const loading = ref(true); // 初始为 true，因为一创建就要加载
const childReplies = ref([]);
const likedMap = ref({});
const replyingTo = ref(null);
const newReplyContent = ref('');

// --- 计算属性 (保持不变) ---
const dialogTitle = computed(() => {
  if (props.parentComment && props.parentComment.child_replies_count > 0) {
    return `${props.parentComment.child_replies_count} 条回复`;
  }
  return '回复列表';
});

// --- 数据获取 ---
const fetchFullReplies = async () => {
  if (!props.parentComment) return;
  loading.value = true;
  try {
    const response = await apiClient.get(`/replies/${props.parentComment.id}/children`);
    childReplies.value = response.data;
    // (理想) 后端应该也返回点赞状态图
  } catch (error) {
    console.error("加载完整回复失败:", error);
  } finally {
    loading.value = false;
  }
};

// 【核心改造】使用 onMounted 生命周期钩子来加载数据
onMounted(() => {
  fetchFullReplies();
});

// --- 方法定义 (保持不变) ---
const isLiked = (replyId) => likedMap.value[replyId] || false;
const activateReply = (user) => { replyingTo.value = user; };

const toggleLike = async (replyId) => {
  likedMap.value[replyId] = !likedMap.value[replyId];
  try {
    await apiClient.post(`/replies/${replyId}/like`);
    await fetchFullReplies();
    emit('main-page-refresh-needed');
  } catch (error) {
    likedMap.value[replyId] = !likedMap.value[replyId];
    ElMessage.error("操作失败", error);
  }
};

const submitReply = async () => {
  if (!newReplyContent.value.trim() || !replyingTo.value) return;
  try {
    await apiClient.post(`/posts/${props.parentComment.post_id}/replies`, {
      content: newReplyContent.value,
      parent_reply_id: props.parentComment.id,
      reply_to_user_id: replyingTo.value.id,
    });
    newReplyContent.value = '';
    replyingTo.value = null;
    ElMessage.success("回复成功！");
    await fetchFullReplies(); // 自己刷新
    emit('main-page-refresh-needed'); // 通知父组件
  } catch (error) {
    ElMessage.error("回复失败", error);
  }
};

const timeAgo = (dateString) => {
  if (!dateString) return '';
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
.parent-comment-wrapper { padding: 20px 20px 0 20px; } /* 调整padding让主评论更好看 */
.reply-dialog :deep(.el-dialog__body) { padding: 0; min-height: 200px; }
.el-divider { margin: 16px 0 0 0; }
.child-replies-list { padding: 20px; max-height: 60vh; overflow-y: auto; }
.root-comment, .child-comment-item { display: flex; gap: 12px; }
.child-comment-item { margin-bottom: 16px; }
.avatar { flex-shrink: 0; }
.avatar-small { flex-shrink: 0; }
.content-wrapper { flex-grow: 1; }
.username { font-weight: 600; font-size: 15px; }
.username-link { font-weight: 600; color: #007bff; }
.content-text { margin: 4px 0 0 0; font-size: 15px; word-break: break-word; } /* 移除底部margin */
.content-text-child { margin: 0 0 8px 0; font-size: 14px; word-break: break-word; }
.reply-to { margin: 0 4px; }
.footer { font-size: 13px; color: #8a919f; }
.dialog-reply-editor {
  display: flex; gap: 8px; align-items: center;
  padding: 10px 20px; border-top: 1px solid #e9e9e9;
}
</style>