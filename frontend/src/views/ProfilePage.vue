<template>
  <div v-if="loading" class="loading-state">正在加载用户资料...</div>
  <div v-else-if="userProfile" class="profile-page">
    <!-- 1. 用户信息头 -->
    <div class="profile-header">
      <el-avatar :size="80" :src="userProfile.avatar_url || defaultAvatar" />
      <div class="user-info">
        <h2>{{ userProfile.username }}</h2>
        <div class="stats">
          <span @click="showFollowers" class="stat-item"><strong>{{ userProfile.followers_count }}</strong> 粉丝</span>
          <span @click="showFollowing" class="stat-item"><strong>{{ userProfile.following_count }}</strong> 关注</span>
        </div>
      </div>
      <!-- 2. 操作按钮 -->
      <div class="action-button">
        <el-button v-if="isMe" size="large" @click="editProfile">编辑资料</el-button>
        <el-button v-else-if="isFollowing" size="large" @click="toggleFollow" type="info" :loading="followLoading">已关注</el-button>
        <el-button v-else size="large" @click="toggleFollow" type="primary" :loading="followLoading">关注</el-button>
      </div>
    </div>
    
    <!-- 3. Tab 切换区域 (先做个占位) -->
    <el-tabs v-model="activeTab" class="profile-tabs">
      <el-tab-pane label="他的作品" name="recordings">
        <el-empty description="作品列表功能开发中..."></el-empty>
      </el-tab-pane>
      <el-tab-pane label="粉丝" name="followers">
        <el-empty description="粉丝列表功能开发中..."></el-empty>
      </el-tab-pane>
      <el-tab-pane label="关注" name="following">
        <el-empty description="关注列表功能开发中..."></el-empty>
      </el-tab-pane>
    </el-tabs>
  </div>
  <el-empty v-else description="用户不存在或无法加载" />
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';

const route = useRoute();
const authStore = useAuthStore();

const userProfile = ref(null);
const loading = ref(true);
const followLoading = ref(false);
const isFollowing = ref(false);
const activeTab = ref('recordings');
const defaultAvatar = 'https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png'; // 默认头像

// 计算属性，判断是否是自己的主页
const isMe = computed(() => 
  userProfile.value && Number(userProfile.value.id) === Number(authStore.userId)
);

// API 调用：获取用户资料
const fetchUserProfile = async (userId) => {
  loading.value = true;
  try {
    const response = await apiClient.get(`/users/${userId}`);
    userProfile.value = response.data;
    isFollowing.value = response.data.is_followed_by_me;
  } catch (error) {
    ElMessage.error('加载用户资料失败');
    userProfile.value = null;
  } finally {
    loading.value = false;
  }
};

// 关注/取关逻辑 (包含乐观更新)
const toggleFollow = async () => {
  if (!userProfile.value) return;
  
  followLoading.value = true;
  const targetState = !isFollowing.value;
  const originalFollowersCount = userProfile.value.followers_count;
  
  // 乐观更新
  isFollowing.value = targetState;
  userProfile.value.followers_count += targetState ? 1 : -1;

  try {
    const userId = userProfile.value.id;
    if (targetState) {
      await apiClient.post(`/users/${userId}/follow`);
    } else {
      await apiClient.delete(`/users/${userId}/follow`);
    }
  } catch (error) {
    ElMessage.error('操作失败');
    // 回滚
    isFollowing.value = !targetState;
    userProfile.value.followers_count = originalFollowersCount;
  } finally {
    followLoading.value = false;
  }
};

// 监听路由参数变化，以便在同一个页面组件内跳转到不同用户主页时能刷新数据
watch(() => route.params.userId, (newId) => {
    if (newId && route.name === 'profile') {
        fetchUserProfile(newId);
    }
}, { immediate: true }); // immediate: true 保证组件首次加载时也会执行

// 占位函数
const editProfile = () => ElMessage.info('编辑资料功能开发中...');
const showFollowers = () => ElMessage.info('查看粉丝列表功能开发中...');
const showFollowing = () => ElMessage.info('查看关注列表功能开发中...');
</script>

<style scoped>
.profile-page { padding: 20px 40px; }
.profile-header { display: flex; align-items: center; gap: 20px; padding-bottom: 20px; border-bottom: 1px solid #e4e7ed; }
.user-info { flex-grow: 1; }
.user-info h2 { margin: 0 0 10px 0; font-size: 24px; }
.stats { display: flex; gap: 20px; color: #606266; }
.stat-item { cursor: pointer; }
.profile-tabs { margin-top: 20px; }
</style>