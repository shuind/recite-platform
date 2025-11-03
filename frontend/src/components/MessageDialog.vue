<!-- src/components/MessageDialog.vue -->
<template>
  <el-dialog
    v-model="dialogVisible"
    :title="`与 ${recipient?.username} 的私信`"
    width="650px"
    @closed="$emit('close')"
    class="message-dialog"
    append-to-body
  >
    <div class="message-container" v-loading="loading" ref="messageContainerRef">
      <div v-if="messages.length > 0" class="message-list">
        <div v-for="msg in messages" :key="msg.id" class="message-item" :class="{ 'is-me': msg.sender_id === loggedInUserId }">
          <el-avatar class="avatar" :src="msg.sender_id === loggedInUserId ? loggedInUserAvatar : recipient.avatar_url" />
          <div class="message-bubble">
            <p class="message-content">{{ msg.content }}</p>
          </div>
        </div>
      </div>
      <el-empty v-else description="还没有消息，开始对话吧"></el-empty>
    </div>
    <div class="message-input-area">
      <el-input
        v-model="newMessage"
        type="textarea"
        :rows="3"
        placeholder="输入消息..."
        @keydown.enter.prevent="sendMessage"
      />
      <el-button type="primary" @click="sendMessage" :loading="isSending">发送</el-button>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue';
import apiClient from '@/api';

const props = defineProps({
  visible: { type: Boolean, default: false },
  recipient: { type: Object, default: null } // 接收完整的聊天对象用户信息
});

defineEmits(['close']);

const loading = ref(false);
const isSending = ref(false);
const messages = ref([]);
const newMessage = ref('');
const messageContainerRef = ref(null);

const dialogVisible = ref(props.visible);

const loggedInUser = JSON.parse(localStorage.getItem('user'));
const loggedInUserId = loggedInUser ? loggedInUser.id : null;
const loggedInUserAvatar = loggedInUser ? loggedInUser.avatar_url : '';

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick();
  const container = messageContainerRef.value;
  if (container) {
    container.scrollTop = container.scrollHeight;
  }
};

const fetchMessages = async () => {
  if (!props.recipient) return;
  loading.value = true;
  try {
    const response = await apiClient.get(`/messages/with/${props.recipient.id}`);
    messages.value = response.data;
    await scrollToBottom();
  } catch (error) { console.error("加载私信失败:", error); } 
  finally { loading.value = false; }
};

const sendMessage = async () => {
  if (!newMessage.value.trim() || !props.recipient) return;
  isSending.value = true;
  try {
    const response = await apiClient.post('/messages', {
      recipient_id: props.recipient.id,
      content: newMessage.value
    });
    messages.value.push(response.data);
    newMessage.value = '';
    await scrollToBottom();
  } catch (error) { console.error("发送私信失败:", error); } 
  finally { isSending.value = false; }
};

watch(() => props.visible, (val) => {
  dialogVisible.value = val;
  if (val) {
    fetchMessages();
  }
});
</script>

<style scoped>
.message-dialog :deep(.el-dialog__body) { padding: 0; }
.message-container { height: 50vh; overflow-y: auto; padding: 20px; }
.message-list { display: flex; flex-direction: column; gap: 16px; }
.message-item { display: flex; gap: 10px; max-width: 70%; }
.message-item.is-me { align-self: flex-end; flex-direction: row-reverse; }
.message-bubble { background-color: #f2f3f5; padding: 10px 14px; border-radius: 18px; }
.message-item.is-me .message-bubble { background-color: #409eff; color: #fff; }
.message-content { margin: 0; line-height: 1.5; }
.message-input-area { display: flex; padding: 10px; border-top: 1px solid #e9e9e9; }
.message-input-area .el-textarea { margin-right: 10px; }
</style>