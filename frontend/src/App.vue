<!-- src/App.vue -->
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
          <el-menu-item index="/my-content">我的内容</el-menu-item>
          <el-menu-item index="/forum">社区论坛</el-menu-item>

          <div class="flex-grow" />

          <!-- ============================================= -->
          <!--      【核心改造】用户下拉菜单                  -->
          <!-- ============================================= -->
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
  </div>
</template>

<script setup>
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

// --- 1. 导入所有在 <template> 中用到的图标组件 ---
import { 
  ArrowDown, 
  User, 
  SwitchButton 
} from '@element-plus/icons-vue';

// --- 2. 获取实例 ---
const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

// --- 3. 定义所有在 <template> 中用到的变量 ---
const defaultAvatar = 'https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png';

// --- 4. 定义所有在 <template> 中用到的函数 ---
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
  router.push('/my-domains'); // 或者你希望的默认主页
};


authStore.initializeAuth();

</script>

<style>
/* 全局基础样式 */
html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  width: 100%;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
}

/* 【核心】统一的全局背景 */
.global-layout {
  height: 100%;
  background-color: #f7f9fd; /* Fallback color */
  background-image:
    radial-gradient(ellipse at center, #F4F6FC, transparent 70%),
    linear-gradient(to bottom, #F7F9FD, #F5F8FD);
  overflow: hidden;
}

.app-container {
  height: 100vh;
  width: 100%;
}

/* Header 样式 */
.app-header {
  --el-header-padding: 0 20px;
  background-color: transparent !important;
  border-bottom: 1px solid transparent;
  padding: 0;
  height: 78px; 
}

/* el-menu 样式 */
.app-header .el-menu {
  border-bottom: none !important;
  height: 100%;
  background-color: transparent !important;
  padding: 0 20px;
}

.app-header .logo {
  display: flex;
  align-items: center;
  font-size: 28px;
  font-weight: bold;
  color: #303133;
  cursor: pointer;
  /* 【修改】给Logo右侧一点边距，和菜单项分开 */
  margin-right: 30px; 
  margin-left: -5px; 
}

/* 核心间隔样式 */
.flex-grow {
  flex-grow: 1;
}

/* 主内容区 */
.app-main-content {
  flex: 1;
  background-color: transparent !important;
  padding: 0;
  overflow: hidden;
}
/* 【新增】用户下拉菜单的样式 */
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