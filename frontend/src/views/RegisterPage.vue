<template>
  <div class="register-container">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>创建新账户</span>
        </div>
      </template>
      <el-form @submit.prevent="handleRegister">
        <el-form-item label="用户名">
          <el-input v-model="username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input type="password" v-model="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input type="password" v-model="confirmPassword" placeholder="请再次输入密码" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading">立即注册</el-button>
        </el-form-item>
      </el-form>
      <div class="footer-link">
        已有账户？ <router-link to="/login">点此登录</router-link>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)

const authStore = useAuthStore()

const handleRegister = async () => {
  if (password.value !== confirmPassword.value) {
    ElMessage.error('两次输入的密码不一致！')
    return
  }
  if (!username.value || !password.value) {
    ElMessage.error('用户名和密码不能为空！')
    return
  }

  loading.value = true
  try {
    // 调用 store 中的注册 action
    await authStore.register(username.value, password.value)
    // 注册成功后，authStore 内部会处理跳转或提示
  } catch (error) {
    // 错误已在 store 中处理，这里不需要额外操作
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f5f7fa;
}
.box-card {
  width: 450px;
}
.footer-link {
  margin-top: 15px;
  text-align: center;
  font-size: 14px;
}
</style>