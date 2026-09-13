<template>
  <div class="auth-page">
    <div class="auth-card">
      <router-link class="brand" to="/login"><span class="brand-mark">S</span>ShortLinkGo</router-link>
      <p class="eyebrow">找回密码</p>
      <h1>重置密码</h1>
      <p class="sub">输入注册邮箱，我们会发送一封包含重置链接的邮件（30 分钟内有效）。</p>
      <form @submit.prevent="onSubmit">
        <label class="field">
          <span>注册邮箱</span>
          <input v-model="email" type="email" autocomplete="email" placeholder="you@example.com" />
        </label>
        <button class="btn btn-primary btn-block" type="submit" :disabled="loading">
          {{ loading ? '发送中…' : '发送重置邮件' }}
        </button>
      </form>

      <div v-if="sent" class="notice">
        <p class="notice-title">{{ sentMsg }}</p>
        <p v-if="devUrl" class="notice-dev">
          开发模式重置链接：<a :href="devUrl" target="_blank">{{ devUrl }}</a>
        </p>
      </div>

      <p class="switch">想起来了？<router-link to="/login">返回登录</router-link></p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import api from '../api'

const email = ref('')
const loading = ref(false)
const sent = ref(false)
const sentMsg = ref('')
const devUrl = ref('')

async function onSubmit() {
  if (!email.value.trim()) {
    sent.value = false
    return
  }
  loading.value = true
  try {
    const res = await api.post('/api/auth/forgot', { email: email.value.trim() })
    if (res.code === 0) {
      sent.value = true
      sentMsg.value = res.msg || '如果该邮箱已注册，重置邮件已发送'
      devUrl.value = res.data?.reset_url || ''
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 40px 20px; background: var(--bg); }
.auth-card { width: min(420px, 100%); }
.brand { display: inline-flex; align-items: center; gap: 10px; font-size: 15px; font-weight: 600; letter-spacing: -0.01em; margin-bottom: 44px; }
.brand-mark {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; border: 1px solid var(--line-strong); border-radius: 6px; font-size: 13px;
}
.auth-card h1 { margin: 12px 0 14px; font-size: 32px; font-weight: 600; letter-spacing: -0.025em; }
.sub { margin-bottom: 26px; font-size: 13px; line-height: 1.8; color: var(--ink-3); }

.notice { margin-top: 22px; padding: 16px; border: 1px solid var(--line); border-radius: var(--r-md); background: var(--surface); }
.notice-title { font-size: 14px; color: var(--ink-2); }
.notice-dev { margin-top: 8px; font-size: 12px; line-height: 1.7; color: var(--ink-4); word-break: break-all; }
.notice-dev a { color: var(--ink-3); text-decoration: underline; text-underline-offset: 2px; }

.switch { margin-top: 22px; font-size: 13px; color: var(--ink-3); }
.switch a { color: var(--ink-2); text-decoration: underline; text-underline-offset: 3px; }
</style>
