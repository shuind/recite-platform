<template>
  <div class="forum-container" ref="forumContainerRef"> 
    <div class="main-content">
      <QuickPostBox @post-created="handleNewPost" />

      <div class="post-list" v-loading="loading && posts.length === 0">
        <div v-if="posts.length > 0">
          <PostCard v-for="post in posts" :key="post.id" :post="post" @update:like="handlePostUpdate" />
        </div>
        <el-empty v-if="!loading && posts.length === 0" description="这里还没有帖子，快来发布第一篇吧！" />
      </div>
      
      <div class="loading-more" v-if="isLoadingMore">
        <el-skeleton :rows="3" animated />
      </div>

      <div class="no-more" v-if="!hasMore && posts.length > 0">
        <el-divider>没有更多内容了</el-divider>
      </div>
    </div>

    <div class="sidebar">
      <CreatorCenterCard />
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import apiClient from '@/api';
import { ElMessage } from 'element-plus';
import PostCard from '@/components/PostCard.vue';
import QuickPostBox from '@/components/QuickPostBox.vue';
import CreatorCenterCard from '@/components/CreatorCenterCard.vue';

const posts = ref([]);
const loading = ref(true);
const isLoadingMore = ref(false);
const hasMore = ref(true);

const pagination = reactive({
  total: 0,
  page: 1,
  limit: 5,
});

const scrollContainer = ref(null);
const forumContainerRef = ref(null);
const route = useRoute();

const resetAndFetch = async () => {
  posts.value = [];
  pagination.page = 1;
  hasMore.value = true;
  isLoadingMore.value = false;
  loading.value = true;

  if (scrollContainer.value && scrollContainer.value.scrollTo) {
    scrollContainer.value.scrollTo({ top: 0, behavior: 'smooth' });
  }
  
  await fetchPosts();
};

const fetchPosts = async () => {
  if (isLoadingMore.value || !hasMore.value) return;

  if (pagination.page > 1) {
    isLoadingMore.value = true;
  } else {
    loading.value = true;
  }

  try {
    const response = await apiClient.get('/posts', {
      params: { 
        page: pagination.page, 
        limit: pagination.limit 
      },
      headers: {
        'Cache-Control': 'no-cache',
        'Pragma': 'no-cache',
        'Expires': '0',
      },
    });
    
    posts.value.push(...response.data.posts);
    pagination.total = response.data.total;

    if (response.data.posts.length < pagination.limit || posts.value.length >= pagination.total) {
      hasMore.value = false;
    } else {
      pagination.page++;
    }
  } catch (error) {
    console.error("Failed to fetch posts:", error);
    ElMessage.error("获取帖子列表失败");
    hasMore.value = false;
  } finally {
    loading.value = false;
    isLoadingMore.value = false;
  }
};

const handleScroll = () => {
  if (!scrollContainer.value) return;
  const { clientHeight, scrollTop, scrollHeight } = scrollContainer.value;
  if (clientHeight + scrollTop >= scrollHeight - 200) {
    fetchPosts();
  }
};

// 【新增】处理子组件 PostCard 发出的 'update:like' 事件
const handlePostUpdate = (payload) => {
  const postToUpdate = posts.value.find(p => p.id === payload.postId);

  if (postToUpdate) {
    postToUpdate.is_liked_by_me = payload.is_liked_by_me;
    postToUpdate.votes_count = payload.votes_count;
  }
};
// 【新增】处理新帖子的回调函数
const handleNewPost = (newPostData) => {
  // 将子组件传递过来的新帖子数据，添加到帖子数组的顶部
  posts.value.unshift(newPostData);
  // 更新帖子总数，以保持分页逻辑的准确性
  pagination.total++;
};
onMounted(() => {
  resetAndFetch();
  
  if (forumContainerRef.value) {
    scrollContainer.value = forumContainerRef.value.closest('.app-main-content');
    if (scrollContainer.value) {
      scrollContainer.value.addEventListener('scroll', handleScroll);
    } else {
      scrollContainer.value = window;
      window.addEventListener('scroll', handleScroll);
    }
  }
});

onUnmounted(() => {
  if (scrollContainer.value) {
    scrollContainer.value.removeEventListener('scroll', handleScroll);
  }
});

watch(
  () => route.fullPath,
  (newPath, oldPath) => {
    if (posts.value.length > 0 && newPath === '/forum' && oldPath !== '/forum') {
        console.log('Watcher: 检测到返回论坛主页，强制刷新帖子列表...');
        resetAndFetch();
    }
  },
  {
    immediate: false
  }
);
</script>

<style scoped>
/* 样式部分保持不变 */
.forum-container {
  display: flex;
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  gap: 20px;
  align-items: flex-start;
}
.main-content { flex: 1; min-width: 0; }
.sidebar { width: 300px; position: sticky; top: 20px; }
.post-list { background-color: #fff; border-radius: 4px; box-shadow: 0 1px 2px rgba(0,0,0,0.05); overflow: hidden; }
.loading-more { padding: 20px; background-color: #fff; margin-top: -1px; border-bottom-left-radius: 4px; border-bottom-right-radius: 4px; }
.no-more { padding: 20px 0; color: #909399; font-size: 14px; text-align: center; }
</style>