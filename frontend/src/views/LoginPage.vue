<template>
  <div class="login-container">
    <el-card class="box-card">
      <h2>登录</h2>
      <el-form @submit.prevent="handleLogin">
        <el-form-item label="用户名">
          <el-input v-model="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input type="password" v-model="password" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit">登录</el-button>
        </el-form-item>
      </el-form>
      <div class="footer-link">
        还没有账户？ <router-link to="/register">立即注册</router-link>
      </div>
      <!-- 这里可以加一个注册的入口 -->
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const username = ref('')
const password = ref('')
const authStore = useAuthStore()

const handleLogin = () => {
  authStore.login(username.value, password.value)
}
</script>

<style scoped>
.login-container { display: flex; justify-content: center; align-items: center; height: 100vh; }
.box-card { width: 400px; }
.footer-link {
  margin-top: 15px;
  text-align: center;
  font-size: 14px;
}
</style>