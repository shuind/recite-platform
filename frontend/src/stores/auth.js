import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from '@/api'
import router from '@/router'
import { ElMessage } from 'element-plus'

export const useAuthStore = defineStore('auth', () => {
    // --- 统一和简化的状态 ---
    const token = ref(localStorage.getItem('token') || null);
    // user 是唯一的用户信息来源
    const user = ref(JSON.parse(localStorage.getItem('user')) || null); 
    const isInitialized = ref(false);

    // --- 派生状态 (Computed Properties) ---
    const isAuthenticated = computed(() => !!token.value && !!user.value);
    const userId = computed(() => user.value?.id || null);
    const username = computed(() => user.value?.username || '');

    // --- 私有辅助函数 ---
    function setUserAndToken(userData, newToken) {
        user.value = userData;
        token.value = newToken;
        localStorage.setItem('user', JSON.stringify(userData));
        localStorage.setItem('token', newToken);
    }

    function clearAuthData() {
        user.value = null;
        token.value = null;
        localStorage.removeItem('user');
        localStorage.removeItem('token');
    }

    // --- Actions ---
    async function initializeAuth() {
        if (isInitialized.value) return; // 防止重复初始化

        if (token.value) {
            try {
                // 使用新的 /profile 接口恢复用户信息
                const response = await apiClient.get('/profile');
                // 仅更新 user 对象，token 仍然是旧的有效的 token
                user.value = response.data;
                localStorage.setItem('user', JSON.stringify(response.data));
            } catch (error) {
                console.warn("Token invalid during initialization, clearing auth data.");
                clearAuthData(); // Token 无效，清除所有认证信息
            }
        }
        isInitialized.value = true;
    }

    async function login(username, password) {
        try {
            const response = await apiClient.post('/login', { username, password });
            const responseData = response.data;
            
            // 验证后端返回的数据是否完整
            if (responseData.token && responseData.user && responseData.user.id) {
                setUserAndToken(responseData.user, responseData.token);
                router.push('/');
            } else {
                console.error("Login response is missing token or user object.");
                ElMessage.error('登录响应异常，缺少必要信息');
            }
        } catch (error) {
            console.error("Login failed:", error);
            ElMessage.error(error.response?.data?.error || '登录失败');
        }
    }

    function logout() {
        clearAuthData();
        router.push('/login');
    }
    
    async function register(username, password) {
        try {
            // 1. 调用后端的 /register API
            // 确保你的后端注册接口路径是 /api/v1/register
            const response = await apiClient.post('/register', { 
                username: username, 
                password: password 
            });

            // 2. 处理成功响应
            // 注册成功后，后端通常会返回一个成功的消息
            ElMessage.success(response.data.message || '注册成功！现在您可以登录了。');
            
            // 3. 自动跳转到登录页面
            router.push('/login');

        } catch (error) {
            // 4. 处理各种错误情况
            // error.response.data.error 是我们后端 API 返回的错误信息
            const errorMessage = error.response?.data?.error || '注册失败，发生未知错误。';
            
            ElMessage.error(errorMessage);

            // 5. 重新抛出错误，让调用它的组件知道发生了错误
            // 这样 RegisterPage.vue 的 loading 状态才能被正确处理
            throw error;
        }
    }

    return { 
        // 状态
        token, 
        user,
        isInitialized,
        // 计算属性 (Getters)
        isAuthenticated, 
        userId,
        username,
        // 方法 (Actions)
        initializeAuth, 
        login, 
        logout,
        register 
    }
})