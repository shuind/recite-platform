<!-- src/App.vue (私信功能集成版) -->
<template>
  <div v-if="!authStore.isInitialized" class="app-loading">
    正在加载应用...
  </div>
  <div class="global-layout">
    <el-container class="app-container">
      <el-header v-if="authStore.isAuthenticated" class="app-header">
        <el-menu mode="horizontal" :router="true" :default-active="route.path" :ellipsis="false" background-color="transparent">
          
          <div class="logo" @click="goHome">朗读练习</div>

          <el-menu-item index="/my-domains">我的圈子</el-menu-item>
          <el-menu-item index="/tasks">任务规划</el-menu-item>
          <el-menu-item index="/my-content">我的内容</el-menu-item>
          <el-menu-item index="/forum">社区论坛</el-menu-item>

          <div class="flex-grow" />

          <el-dropdown v-if="authStore.isAuthenticated" @command="handleCommand" class="user-dropdown">
            <span class="el-dropdown-link">
              <el-avatar :size="32" :src="defaultAvatar" style="margin-right: 8px;"></el-avatar>
              {{ authStore.username || '用户' }}
              <el-icon class="el-icon--right"><arrow-down /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>
                  个人主页
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          
        </el-menu>
      </el-header>

      <el-main class="app-main-content">
        <router-view />
      </el-main>
    </el-container>

    <!-- 
      【核心新增】
      将私信弹窗组件放置在全局布局的根节点
    -->
    <MessageDialog 
      :visible="isMessageDialogOpen"
      :recipient="messageRecipient"
      @close="isMessageDialogOpen = false"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'; // 【新增】引入 ref, onMounted, onUnmounted
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { ArrowDown, User, SwitchButton } from '@element-plus/icons-vue';

// 【新增】引入事件总线和私信弹窗组件
import eventBus from '@/utils/eventBus';
import MessageDialog from '@/components/MessageDialog.vue';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const defaultAvatar = 'https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png';

// --- 【新增】私信弹窗的状态管理 ---
const isMessageDialogOpen = ref(false);
const messageRecipient = ref(null);

const handleOpenMessageDialog = (recipient) => {
  messageRecipient.value = recipient;
  isMessageDialogOpen.value = true;
};
// ------------------------------------

const handleCommand = (command) => {
  if (command === 'logout') {
    authStore.logout();
  } else if (command === 'profile') {
    if (authStore.userId) {
      router.push({ name: 'profile', params: { userId: authStore.userId } });
    }
  }
};

const goHome = () => {
  router.push('/my-domains');
};


authStore.initializeAuth();


// --- 【新增】在组件生命周期中监听和销毁全局事件 ---
onMounted(() => {
  eventBus.on('open-message-dialog', handleOpenMessageDialog);
});

onUnmounted(() => {
  eventBus.off('open-message-dialog', handleOpenMessageDialog);
});
// ------------------------------------------------

</script>
<style>
:root {
  --header-h: 78px;
  /* ... 其他 CSS 变量 */
}

/* 全局基础样式 */
html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  width: 100%;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
}

/* 
  【核心修改】
  - 移除 overflow: hidden
  - 将背景色设置在这里，作为真正的全局背景
*/
.global-layout {
  height: 100%;
  background-color: #f4f5f5; /* 论坛页的背景色移到这里 */
}

.app-container {
  height: 100vh;
  width: 100%;
  display: flex; /* 使用 Flex 布局 */
  flex-direction: column; /* 垂直排列 Header 和 Main */
}

/* Header 样式 (保持不变) */
.app-header {
  --el-header-padding: 0 20px;
  background-color: #ffffff; /* 建议给 Header 一个明确的白色背景 */
  border-bottom: 1px solid #e9e9e9; /* 增加一个细边框更有质感 */
  padding: 0;
  height: var(--header-h);
  flex-shrink: 0; /* 防止 Header 在内容过多时被压缩 */
}

.app-header .el-menu {
  border-bottom: none !important;
  height: 100%;
  padding: 0 20px;
}

.app-header .logo {
  display: flex;
  align-items: center;
  font-size: 28px;
  font-weight: bold;
  color: #303133;
  cursor: pointer;
  margin-right: 30px; 
  margin-left: -5px; 
}

.flex-grow {
  flex-grow: 1;
}

/* 
  【核心修改】
  - 移除固定的 height 计算
  - 将 overflow: hidden 改为 overflow-y: auto
*/
.app-main-content {
  flex-grow: 1; /* 让主内容区自动填充剩余空间 */
  background-color: transparent !important;
  padding: 0;
  overflow-y: auto; /* 【关键】只在垂直方向上，当内容超出时，自动显示滚动条 */
}

/* 用户下拉菜单的样式 (保持不变) */
.user-dropdown {
  display: flex;
  align-items: center;
  height: 100%;
}

.el-dropdown-link {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: var(--el-menu-text-color);
  font-size: 14px;
} 
</style>