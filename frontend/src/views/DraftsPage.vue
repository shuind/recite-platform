<template>
    <div class="drafts-page">
        <h2>我的草稿箱</h2>
        <el-card>
        <div v-if="drafts.length > 0" class="draft-list">
            <div v-for="draft in drafts" :key="draft.id" class="draft-item" @click="editDraft(draft.id)">
            <span class="draft-title">{{ draft.title || '无标题' }}</span>
            <span class="draft-time">最后修改于 {{ timeAgo(draft.updated_at) }}</span>
            </div>
        </div>
            <el-empty v-else description="草稿箱是空的"></el-empty>
        </el-card>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import apiClient from '@/api';

const drafts = ref([]);
const router = useRouter();

onMounted(async () => {
  const response = await apiClient.get('/drafts');
  drafts.value = response.data;
});

const editDraft = (id) => {
  router.push(`/posts/edit/${id}`);
};

// timeAgo 函数
const timeAgo = (dateString) => {
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
  return "刚刚"; };
</script>

<style scoped>
.draft-item { display: flex; justify-content: space-between; padding: 12px; cursor: pointer; }
.draft-item:hover { background-color: #fafafa; }
</style>