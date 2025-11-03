<!-- src/components/AuthorInfoCard.vue (私信功能增强版) -->
<template>
  <el-card class="author-card" shadow="never">
    <template #header>
      <span>关于作者</span>
    </template>
    <div class="author-summary">
      <el-avatar :size="50" :src="author.avatar_url" />
      <div class="author-card-details">
        <span class="author-name">{{ author.username }}</span>
        <span class="author-bio">用户的个人简介</span>
      </div>
    </div>
    <div class="author-stats">
      <div class="stat-item">
        <span>回答</span>
        <strong>--</strong>
      </div>
      <div class="stat-item">
        <span>关注者</span>
        <strong>{{ author.followers_count || 0 }}</strong>
      </div>
    </div>
    <div class="author-card-actions">
      <!-- 关注按钮 (您的原有代码，保持不变) -->
      <el-button 
        v-if="author && loggedInUserId && author.id !== loggedInUserId"
        :type="isFollowed ? 'info' : 'primary'"
        style="flex: 1;"
        @click="$emit('follow-toggle')"
        :plain="isFollowed"
        round
      >
        {{ isFollowed ? '已关注' : '+ 关注他' }}
      </el-button>
      
      <!-- 【核心改造】私信按钮现在绑定了 openMessage 函数 -->
      <el-button style="flex: 1;" round @click="openMessage">发私信</el-button>
    </div>
  </el-card>
</template>

<script setup>
import eventBus from '@/utils/eventBus'; // 【新增】引入事件总线

const props = defineProps({
  author: { type: Object, required: true },
  isFollowed: { type: Boolean, default: false },
  loggedInUserId: { type: [Number, null], default: null }
});

// 声明组件可以发出的 'follow-toggle' 事件 (您的原有代码，保持不变)
defineEmits(['follow-toggle']);

// 【新增】广播事件的函数
const openMessage = () => {
  // 通过全局事件总线广播一个 'open-message-dialog' 事件
  // 并将完整的作者(聊天对象)信息作为参数传递出去
  eventBus.emit('open-message-dialog', props.author);
};

</script>

<style scoped>
/* 您所有的样式代码保持完全不变 */
.author-card { 
  border: 1px solid #e9e9e9;
  border-radius: 4px;
}
.author-summary { 
  display: flex; 
  gap: 12px; 
  align-items: center; 
}
.author-name { 
  font-weight: 600; 
  font-size: 16px;
}
.author-bio { 
  font-size: 13px; 
  color: #8a919f; 
  margin-top: 4px;
}
.author-stats { 
  display: flex; 
  justify-content: space-around; 
  text-align: center; 
  margin: 20px 0; 
}
.stat-item span { 
  font-size: 14px; 
  color: #8a919f; 
}
.stat-item strong { 
  display: block; 
  font-size: 18px; 
  font-weight: 600; 
  margin-top: 4px;
  color: #1d2129;
}
.author-card-actions { 
  display: flex; 
  gap: 10px; 
}
</style>