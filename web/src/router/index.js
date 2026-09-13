import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue'), meta: { title: '登录', guest: true } },
  { path: '/oauth', name: 'oauth', component: () => import('../views/OauthCallback.vue'), meta: { title: '登录中' } },
  { path: '/register', name: 'register', component: () => import('../views/Register.vue'), meta: { title: '注册', guest: true } },
  { path: '/forgot', name: 'forgot', component: () => import('../views/Forgot.vue'), meta: { title: '找回密码', guest: true } },
  { path: '/reset', name: 'reset', component: () => import('../views/Reset.vue'), meta: { title: '重置密码', guest: true } },
  {
    path: '/',
    component: () => import('../components/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '统计' } },
      { path: 'links', name: 'links', component: () => import('../views/Links.vue'), meta: { title: '链接管理' } },
      { path: 'me', name: 'me', component: () => import('../views/Me.vue'), meta: { title: '个人中心' } },
      { path: 'admin', name: 'admin', component: () => import('../views/Admin.vue'), meta: { title: '系统管理', staff: true } }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to) => {
  document.title = to.meta?.title ? `${to.meta.title} - ${useSiteTitle()}` : 'ShortLinkGo'
  const store = useUserStore()
  const authed = Boolean(store.token)

  if (to.meta?.guest && authed) return { name: 'dashboard' }
  if (to.meta?.requiresAuth && !authed) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.meta?.staff && !store.isStaff) return { name: 'dashboard' }
  return true
})

function useSiteTitle() {
  try {
    const s = JSON.parse(localStorage.getItem('site') || '{}')
    return s.name || 'ShortLinkGo'
  } catch (e) {
    return 'ShortLinkGo'
  }
}

export default router
