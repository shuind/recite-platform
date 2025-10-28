import axios from 'axios'
import { useAuthStore } from '@/stores/auth' // 稍后创建

const apiClient = axios.create({
  /**
   * 【核心修正】
   * baseURL 必须是一个相对路径，与 Nginx 配置中的 location 匹配。
   * 你的 Nginx 配置是 `location /api`，但你的后端路由组是 `/api/v1`。
   * 为了统一，我们让前端所有请求都带上 /api/v1 前缀。
   */
  baseURL: '/api/v1',
});

// 请求拦截器：自动附加 Token
apiClient.interceptors.request.use(config => {
    const authStore = useAuthStore()
    const token = authStore.token
    if (token) {
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})

export default apiClient