<template>
  <div class="forum-page">
    <div class="page-header">
      <h1>社区论坛</h1>
      <!-- 使用 router-link 跳转到专门的发帖页更符合 SPA 规范 -->
      <router-link to="/forum/create-post">
        <el-button type="primary">发布新帖</el-button>
      </router-link>
    </div>
    
    <el-table :data="posts" @row-click="goToPostDetail" v-loading="loading" style="cursor: pointer;">
      <el-table-column label="主题" min-width="400">
        <template #default="scope">
            <div class="post-title">{{ scope.row.title }}</div>
            <div class="post-meta">
            <!-- 修正: scope.row.User -> scope.row.user -->
            <!-- 修正: scope.row.user.id, scope.row.user.username -->
            作者: <router-link :to="{ name: 'profile', params: { userId: scope.row.user.id } }" @click.stop>{{ scope.row.user.username }}</router-link>
            </div>
        </template>
        </el-table-column>
        <el-table-column label="回复/查看" width="150" align="center">
        <template #default="scope">
            <!-- 这两个字段已经是 snake_case，是正确的 -->
            {{ scope.row.replies_count }} / {{ scope.row.views_count }}
        </template>
        </el-table-column>
        <el-table-column label="最后回复" width="250">
        <template #default="scope">
            <!-- 修正: scope.row.LastRepliedByUser -> scope.row.last_replied_by_user -->
            <div v-if="scope.row.last_replied_by_user">
            <!-- last_replied_at 已经是 snake_case，是正确的 -->
            <div>{{ new Date(scope.row.last_replied_at).toLocaleString() }}</div>
            <div class="post-meta">
                <!-- 修正: scope.row.last_replied_by_user.id, scope.row.last_replied_by_user.username -->
                by <router-link :to="{ name: 'profile', params: { userId: scope.row.last_replied_by_user.id } }" @click.stop>{{ scope.row.last_replied_by_user.username }}</router-link>
            </div>
            </div>
            <span v-else>暂无回复</span>
        </template>
        </el-table-column>
    </el-table>
    
    <!-- 分页控件 -->
    <div class="pagination-container">
      <el-pagination
        background
        layout="prev, pager, next"
        :total="pagination.total"
        :page-size="pagination.limit"
        :current-page="pagination.page"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import apiClient from '@/api';
import { ElMessage } from 'element-plus'; // 引入 ElMessage

const posts = ref([]);
const loading = ref(true);
const router = useRouter();
const pagination = reactive({
  total: 0,
  page: 1,
  limit: 20,
});

const fetchPosts = async (page = 1) => {
  loading.value = true;
  try {
    const response = await apiClient.get('/posts', {
      params: { page: page, limit: pagination.limit }
    });
    // 后端返回的 JSON 已经是 { posts: [...], total: ..., page: ... }
    // 所以这里的访问是正确的
    posts.value = response.data.posts;
    pagination.total = response.data.total;
    pagination.page = response.data.page;
  } catch (error) {
    console.error("Failed to fetch posts:", error);
    ElMessage.error("获取帖子列表失败");
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchPosts(1);
});

const handlePageChange = (newPage) => {
  fetchPosts(newPage);
};

const goToPostDetail = (row) => {
  // 尝试将 row.id 转换为数字
  const postId = Number(row.id);

  // 检查转换后的结果是否是一个有效的数字 (不是 NaN)
  // isNaN(null) 是 false，所以要额外检查 row.id 是否存在
  if (row.id != null && !isNaN(postId)) { 
    router.push({ name: 'post-detail', params: { postId: postId } });
  } else {
    console.error('Failed to get a valid post ID from row.id. Row data:', row);
    ElMessage.error('无法跳转，帖子ID无效');
  }
};
</script>

<style scoped>
.forum-page { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.post-title { font-weight: 600; font-size: 16px; margin-bottom: 5px; }
.post-meta { font-size: 12px; color: #909399; }
.post-meta a { color: #909399; text-decoration: none; }
.post-meta a:hover { text-decoration: underline; }
.pagination-container { margin-top: 20px; display: flex; justify-content: center; }
</style>