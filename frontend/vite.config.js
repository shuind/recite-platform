// vite.config.js

import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },

  // ---  在这里添加 server 和 proxy 配置 ---
  server: {
    proxy: {
      // 当请求路径以 /api/v1 开头时，触发此代理规则
      '/api': {
        // 将请求转发到的目标服务器地址
        // !!! 请确保这里的端口号与你 Go 后端服务运行的端口号一致 !!!
        target: 'http://localhost:8080', 
        
        // 必需：修改请求头中的 Origin 字段，通常是为了解决跨域问题
        changeOrigin: true,
      }
    }
  }
})