<template>
  <div class="auth-page">
    <div class="auth-card">
      <router-link class="brand" to="/login"><span class="brand-mark">S</span>ShortLinkGo</router-link>
      <p class="eyebrow">创建账号</p>
      <h1>注册</h1>
      <p class="note">首个注册用户将自动成为超级管理员（UID=1）。</p>
      <form @submit.prevent="onSubmit">
        <label class="field">
          <span>用户名</span>
          <input v-model="form.username" type="text" autocomplete="username" placeholder="2-64 位" />
        </label>
        <label class="field">
          <span>邮箱 <em class="opt">用于找回密码</em></span>
          <input v-model="form.email" type="email" autocomplete="email" placeholder="you@example.com" />
        </label>
        <label class="field">
          <span>昵称 <em class="opt">可选</em></span>
          <input v-model="form.nickname" type="text" placeholder="默认与用户名相同" />
        </label>
        <label class="field">
          <span>密码</span>
          <input v-model="form.password" type="password" autocomplete="new-password" placeholder="至少 6 位" />
        </label>
        <label class="field">
          <span>确认密码</span>
          <input v-model="form.confirm" type="password" autocomplete="new-password" placeholder="再次输入密码" />
        </label>
        <button class="btn btn-primary btn-block" type="submit" :disabled="loading">
          {{ loading ? '注册中…' : '注册' }}
        </button>
      </form>
      <p class="switch">已有账号？<router-link to="/login">去登录</router-link></p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { toast } from '../stores/toast'

const userStore = useUserStore()
const router = useRouter()
const form = reactive({ username: '', email: '', nickname: '', password: '', confirm: '' })
const loading = ref(false)

async function onSubmit() {
  if (!form.username.trim() || !form.email.trim()) {
    toast('请填写用户名和邮箱', 'warn')
    return
  }
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
    await userStore.register({
      username: form.username.trim(),
      email: form.email.trim(),
      nickname: form.nickname.trim(),
      password: form.password
    })
    toast('注册成功，请登录', 'ok')
    router.push('/login')
  } catch (e) {
    // 错误由拦截器 toast
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
.auth-card h1 { margin: 12px 0 30px; font-size: 32px; font-weight: 600; letter-spacing: -0.025em; }
.opt { font-style: normal; font-weight: 400; color: var(--ink-4); }
.note { margin: 6px 0 18px; font-size: 12.5px; color: var(--warn); }
.switch { margin-top: 22px; font-size: 13px; color: var(--ink-3); }
.switch a { color: var(--ink-2); text-decoration: underline; text-underline-offset: 3px; }
</style>
