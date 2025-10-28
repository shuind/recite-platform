<template>
  <div class="page-container">
    <!-- 1. 页面头部 -->
    <div class="header">
      <h1>我的圈子</h1>
      <div>
        <el-button @click="openJoinDialog">加入圈子</el-button>
        <el-button type="primary" @click="openCreateDialog">创建新圈子</el-button>
      </div>
    </div>

    <!-- 【已修复】:key 使用小写 id -->
    <el-col :xs="24" :sm="12" :md="8" v-for="domain in myDomains" :key="domain.id">
      <el-card class="domain-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <span>{{ domain.name }}</span>
            <!-- 【已修复】params.id 使用小写 id -->
            <router-link :to="{ name: 'domain-detail', params: { id: domain.id } }" style="text-decoration: none;">
              <el-button class="button" text>进入圈子</el-button>
            </router-link>
          </div>
        </template>
        
        <div class="card-body">
          <p class="description">{{ domain.description || '圈主很懒，什么都没留下...' }}</p>
          
          <div v-if="Number(domain.owner_id) === Number(authStore.userId)" class="join-info">
            <!-- 【已修复】显示小写 id -->
            <p>圈子ID: <strong>{{ domain.id }}</strong></p>
            <p>邀请码: <strong>{{ domain.join_code }}</strong></p>
            <el-button size="small" @click="copyInviteInfo(domain)">复制邀请信息</el-button>
          </div>
        </div>
      </el-card>
    </el-col>

    <!-- 3. 创建圈子 Dialog (无问题) -->
    <el-dialog v-model="createDialogVisible" title="创建新圈子" width="400px">
      <!-- 主体内容 -->
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="圈子名称">
          <el-input v-model="createForm.name" placeholder="给你的圈子起个名字"></el-input>
        </el-form-item>
        <el-form-item label="圈子简介">
          <el-input v-model="createForm.description" type="textarea" placeholder="简单介绍一下你的圈子"></el-input>
        </el-form-item>
      </el-form>
      
      <!-- 底部按钮 -->
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleCreateDomain" :loading="formLoading">
            确定创建
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 4. 加入圈子 Dialog (无问题) -->
    <el-dialog v-model="joinDialogVisible" title="加入圈子" width="400px">
      <!-- 主体内容 -->
      <!-- 【重要】为 el-form 绑定 ref 和 rules -->
      <el-form :model="joinForm" ref="joinFormRef" :rules="joinFormRules" label-width="80px">
        <el-form-item label="圈子ID" prop="domain_id">
          <el-input v-model.number="joinForm.domain_id" placeholder="请输入圈子ID"></el-input>
        </el-form-item>
        <el-form-item label="邀请码" prop="join_code">
          <el-input v-model="joinForm.join_code" placeholder="请输入邀请码"></el-input>
        </el-form-item>
      </el-form>

      <!-- 底部按钮 -->
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="joinDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleJoinDomain" :loading="formLoading">
            立即加入
          </el-button>
        </div>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import apiClient from '@/api'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth' // 假设你的 auth store 存了 userID

// --- 状态变量 ---
const authStore = useAuthStore()
const myDomains = ref([])
const loading = ref(true)
const formLoading = ref(false)

const createDialogVisible = ref(false)
const joinDialogVisible = ref(false)
const joinFormRef = ref(null) // 新增，用于获取表单实例
const createForm = reactive({ name: '', description: '' })
const joinForm = reactive({
  domain_id: null,
  join_code: ''
})
const joinFormRules = reactive({
  domain_id: [
    { required: true, message: '圈子ID是必填项', trigger: 'blur' },
    { type: 'number', message: '圈子ID必须是数字类型', trigger: 'blur' },
  ],
  join_code: [
    { required: true, message: '邀请码是必填项', trigger: 'blur' },
  ]
});
// --- API 调用函数 ---
const fetchMyDomains = async () => {
  loading.value = true
  try {
    const response = await apiClient.get('/domains/my')
    myDomains.value = response.data
    if (myDomains.value.length > 0) {
      const firstDomain = myDomains.value[0];
    }
  } catch (error) {
    ElMessage.error('获取圈子列表失败')
  } finally {
    loading.value = false
  }
}

// --- 事件处理函数 ---
onMounted(fetchMyDomains)

const openCreateDialog = () => {
  createForm.name = ''
  createForm.description = ''
  createDialogVisible.value = true
}

const openJoinDialog = () => {
  joinForm.domain_id = null;
  joinForm.join_code = '';
  // 如果上次有校验失败的提示，也一并清除
  joinFormRef.value?.resetFields();
  joinDialogVisible.value = true
}

const handleCreateDomain = async () => {
  if (!createForm.name) {
    ElMessage.warning('圈子名称不能为空')
    return
  }
  formLoading.value = true
  try {
    await apiClient.post('/domains', createForm)
    ElMessage.success('圈子创建成功！')
    createDialogVisible.value = false
    fetchMyDomains() // 重新获取列表
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '创建失败')
  } finally {
    formLoading.value = false
  }
}
// 【修改】处理加入圈子的逻辑
const handleJoinDomain = async () => {
  if (!joinFormRef.value) return;

  // 在提交前，先进行表单校验
  await joinFormRef.value.validate(async (valid) => {
    if (valid) {
      formLoading.value = true;
      try {
        // 直接使用 joinForm 对象作为请求体
        await apiClient.post('/domains/join', joinForm);
        ElMessage.success('成功加入圈子！');
        joinDialogVisible.value = false;
        fetchMyDomains(); // 刷新列表
      } catch (error) {
        ElMessage.error(error.response?.data?.error || '加入失败，请检查圈子ID和邀请码');
      } finally {
        formLoading.value = false;
      }
    } else {
      // 表单校验失败
      console.log('表单校验失败');
      return false;
    }
  });
}

const copyInviteInfo = (domain) => {
    // 【已修复】使用小写 id
    const textToCopy = `邀请你加入我的圈子 "${domain.name}"！\n圈子ID: ${domain.id}\n邀请码: ${domain.join_code}`;
    
    navigator.clipboard.writeText(textToCopy).then(() => {
        ElMessage.success('邀请信息已复制到剪贴板！');
    }).catch(err => {
        ElMessage.error('复制失败');
    });
}

</script>

<style scoped>
.page-container {
  padding: 20px;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.domain-card {
  margin-bottom: 20px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-body .description {
  font-size: 14px;
  color: #606266;
  min-height: 40px;
}
.card-body .join-code {
  margin-top: 15px;
  font-size: 14px;
}
</style>