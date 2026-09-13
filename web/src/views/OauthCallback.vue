<template>
  <div class="oauth-page">
    <div class="oauth-card">
      <span class="spinner"></span>
      <p>{{ msg }}</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { toast } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const msg = ref('正在完成登录…')

onMounted(async () => {
  const token = route.query.token
  if (!token) {
    msg.value = '登录回调缺少凭证，请重试'
    toast('QQ 登录失败，请重试', 'error')
    setTimeout(() => router.replace('/login'), 800)
    return
  }
  try {
    userStore.token = token
    localStorage.setItem('token', token)
    await userStore.fetchProfile()
    msg.value = '登录成功，即将跳转…'
    toast('登录成功，欢迎回来', 'ok')
    router.replace(route.query.redirect || '/dashboard')
  } catch (e) {
    msg.value = '登录失败，请重试'
    userStore.logout()
    setTimeout(() => router.replace('/login'), 800)
  }
})
</script>

<style scoped>
.oauth-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: var(--bg); }
.oauth-card { display: flex; flex-direction: column; align-items: center; gap: 16px; font-size: 14px; color: var(--ink-3); }
.spinner {
  width: 26px; height: 26px; border-radius: 50%;
  border: 2px solid var(--line-strong); border-top-color: var(--ink);
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
