import { createRouter, createWebHistory } from 'vue-router'
import LoginPage from '../views/LoginPage.vue'
import RecordingPage from '../views/RecordingPage.vue'
import RegisterPage from '../views/RegisterPage.vue'
import MyContentPage from '../views/MyContentPage.vue'
import MyDomainsPage from '../views/MyDomainsPage.vue'
import DomainDetailPage from '../views/DomainDetailPage.vue'
import ProfilePage from '../views/ProfilePage.vue'
import ForumPage from '../views/ForumPage.vue';
import PostDetailPage from '../views/PostDetailPage.vue';
import CreatePostPage from '../views/CreatePostPage.vue'; 
import TasksPage from '../views/TasksPage.vue'
const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            name: 'home',
            redirect: '/my-content',
            meta: { requiresAuth: true } // 这个页面需要登录
        },
        {
            path: '/login',
            name: 'login',
            component: LoginPage
        },
        {
            path: '/record/:id', // :id 是文本的 ID
            name: 'record',
            component: RecordingPage,
            meta: { requiresAuth: true }
        },
        { 
            path: '/register',
            name: 'register',
            component: RegisterPage
        },
        {
            path: '/my-content', // 新路径
            name: 'my-content',
            component: MyContentPage,
            meta: { requiresAuth: true } // 需要登录
        },
        {   path: '/my-domains', 
            name: 'my-domains', 
            component: MyDomainsPage, 
            meta: { requiresAuth: true } 
        },
        {
            path: '/domains/:id', // 动态路由
            name: 'domain-detail',
            component: DomainDetailPage,
            meta: { requiresAuth: true }
        },
        {
            path: '/profile/:userId', // 动态路由参数
            name: 'profile',
            component: ProfilePage,
            meta: { requiresAuth: true }
        },
        // 论坛相关路由
        { 
            path: '/forum', 
            name: 'forum', 
            component: ForumPage, 
            meta: { requiresAuth: true } 
        },
        { 
            path: '/forum/create-post', 
            name: 'create-post', 
            component: CreatePostPage, 
            meta: { requiresAuth: true } 
        },
        { 
            path: '/forum/post/:postId', 
            name: 'post-detail', 
            component: PostDetailPage, 
            meta: { requiresAuth: true } 
        },
        {   path: '/tasks', 
            name: 'tasks', 
            component: TasksPage,
            meta: { requiresAuth: true } 
        }
    ]
})
import { useAuthStore } from '@/stores/auth'

router.beforeEach(async (to, from, next) => {
    const authStore = useAuthStore()

    // !!! 等待初始化完成 !!!
    // (注意：如果在 main.js 中等待 mount，这里可能不需要显式等待，
    // 但作为双重保障或不同的实现方式，可以这样做)
    if (!authStore.isInitialized) {
         await authStore.initializeAuth();
    }

    if (to.meta.requiresAuth && !authStore.isAuthenticated) {
        next({ name: 'login' })
    } else {
        next()
    }
})


export default router