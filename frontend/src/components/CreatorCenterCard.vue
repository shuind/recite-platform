<template>
  <el-card class="creator-center-card" shadow="never">
    <div class="card-header">
      <span class="title">
        <el-icon><User /></el-icon> 创作中心
      </span>
      <!-- 【修改】点击时调用 goToDrafts 函数 -->
      <a @click="goToDrafts" class="draft-link">草稿箱 ({{ draftCount }})</a>
    </div>
    <div class="card-body">
      <img src="https://p3-juejin.byteimg.com/tos-cn-i-k3u1fbpfcp/47206154e2a6444a98a4a50433735165~tplv-k3u1fbpfcp-jj-mark:0:0:0:0:q75.image#?w=200&h=200&s=1349&e=png&f=0-0" alt="creator" class="illustration"/>
      <div class="welcome-text">开启你的创作之旅</div>
      <p class="slogan">快来成为社区创作者吧~</p>
      <!-- 【修改】点击时调用 goToCreatePost 函数 -->
      <el-button type="primary" size="large" class="create-button" @click="goToCreatePost">
        <el-icon><Plus /></el-icon>
        开始创作
      </el-button>
    </div>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'; // 【新增】引入 ref, onMounted
import { useRouter } from 'vue-router'; // 【新增】引入 useRouter
import { User, Plus } from '@element-plus/icons-vue';
import apiClient from '@/api'; // 【新增】引入您的 apiClient

const router = useRouter(); // 【新增】
const draftCount = ref(0); // 【新增】用于存储草稿数量

// 【新增】在组件挂载后，调用API获取草稿数量
onMounted(async () => {
  try {
    const response = await apiClient.get('/drafts');
    // response.data 是一个草稿数组，我们只需要它的长度
    draftCount.value = response.data.length;
  } catch (error) {
    console.error("获取草稿数量失败:", error);
    // 即使失败，也让 draftCount 保持为 0，不影响页面显示
  }
});

// 【新增】编程式导航函数
const goToDrafts = () => {
  router.push('/drafts');
};
const goToCreatePost = () => {
  router.push('/forum/create-post'); // 确保您的路由路径是这个
};

</script>

<style scoped>
.creator-center-card {
  border: none;
  background-color: #fff;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 10px;
  border-bottom: 1px solid #f0f2f7;
  margin-bottom: 20px;
}
.title {
  font-weight: 600;
  font-size: 16px;
  display: inline-flex;
  align-items: center;
}
.title .el-icon {
  margin-right: 6px;
}
/* 【修改】将 .drafts-link 改为 .draft-link 并添加 cursor */
.draft-link {
  font-size: 14px;
  color: #909399;
  text-decoration: none;
  cursor: pointer;
}
.draft-link:hover {
  color: #409eff;
}
.card-body {
  text-align: center;
}
.illustration {
  width: 80px;
  height: auto;
  margin-bottom: 10px;
}
.welcome-text {
  font-size: 18px;
  font-weight: bold;
  color: #333;
}
.slogan {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
  margin-bottom: 20px;
}
.create-button {
  width: 100%;
}
</style>
