<template>
  <div class="auth-page">
    <div class="auth-card">
      <router-link class="brand" to="/login"><span class="brand-mark">S</span>ShortLinkGo</router-link>
      <p class="eyebrow">设置新密码</p>
      <h1>重置密码</h1>
      <p class="sub">为你的账号设置一个新密码。</p>
      <form @submit.prevent="onSubmit">
        <label class="field">
          <span>新密码</span>
          <input v-model="form.password" type="password" autocomplete="new-password" placeholder="至少 6 位" />
        </label>
        <label class="field">
          <span>确认新密码</span>
          <input v-model="form.confirm" type="password" autocomplete="new-password" placeholder="再次输入密码" />
        </label>
        <button class="btn btn-primary btn-block" type="submit" :disabled="loading || !token">
          {{ loading ? '提交中…' : '重置密码' }}
        </button>
      </form>
      <p v-if="!token" class="warn">重置链接缺少令牌，请从邮件中的链接进入。</p>
      <p class="switch"><router-link to="/login">返回登录</router-link></p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api'
import { toast } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const token = ref(route.query.token || '')
const form = reactive({ password: '', confirm: '' })
const loading = ref(false)

async function onSubmit() {
  if (form.password.length < 6) {
    toast('密码长度不能少于 6 位', 'warn')
    return
  }
  if (form.password !== form.confirm) {
    toast('两次输入的密码不一致', 'warn')
    return
  }
  loading.value = true
  try {
    const res = await api.post('/api/auth/reset', { token: token.value, password: form.password })
    if (res.code === 0) {
      toast('密码已重置，请使用新密码登录', 'ok')
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 40px 20px; background: var(--bg); }
.auth-card { width: min(400px, 100%); }
.brand { display: inline-flex; align-items: center; gap: 10px; font-size: 15px; font-weight: 600; letter-spacing: -0.01em; margin-bottom: 44px; }
.brand-mark {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; border: 1px solid var(--line-strong); border-radius: 6px; font-size: 13px;
}
.auth-card h1 { margin: 12px 0 14px; font-size: 32px; font-weight: 600; letter-spacing: -0.025em; }
.sub { margin-bottom: 26px; font-size: 13px; color: var(--ink-3); }
.warn { margin-top: 16px; font-size: 13px; color: var(--warn); }
.switch { margin-top: 22px; font-size: 13px; color: var(--ink-3); }
.switch a { color: var(--ink-2); text-decoration: underline; text-underline-offset: 3px; }
</style>
