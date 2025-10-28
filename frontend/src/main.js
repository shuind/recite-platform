// src/main.js

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth' // 导入 auth store

// --- 1. 创建应用和 Pinia 实例 ---
const app = createApp(App)
const pinia = createPinia()

// --- 2. 使用插件 ---
app.use(pinia) // 先使用 Pinia，这样 store 才能被创建
app.use(router)
app.use(ElementPlus)

// --- 3. 定义一个异步的启动函数 ---
async function startApp() {
  // 在 use(pinia) 之后，我们才能安全地使用 store
  const authStore = useAuthStore()

  try {
    // 等待异步的认证初始化完成
    await authStore.initializeAuth()
  } catch (error) {
    // 即使初始化失败，也要让应用继续加载，路由守卫会处理重定向
    console.error("Auth initialization failed, proceeding with app mount.", error)
  }

  // --- 4. 在所有准备工作完成后，只挂载一次！---
  app.mount('#app')
}

// --- 5. 调用启动函数 ---
startApp()