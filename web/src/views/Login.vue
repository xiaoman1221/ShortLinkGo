<template>
  <div class="auth">
    <section class="panel panel-dark">
      <div class="brand"><span class="brand-mark">S</span>ShortLinkGo</div>
      <div class="panel-copy">
        <p class="eyebrow dark">短链接服务 · Self-hosted</p>
        <h1>把冗长的链接<br />收进一个短码。</h1>
        <p class="lede">
          生成、自定义、追踪你的短链接 —— 多页面后台、访问统计、账户体系，
          都收纳在一个克制而清晰的控制台里。
        </p>
      </div>
      <ul class="features">
        <li v-for="(f, i) in features" :key="f">
          <span class="idx">0{{ i + 1 }}</span><span>{{ f }}</span>
        </li>
      </ul>
      <p class="foot">© 2026 ShortLinkGo · 数据存储于你自己的服务器</p>
    </section>

    <main class="panel panel-light">
      <form class="auth-form" @submit.prevent="onSubmit">
        <p class="eyebrow">欢迎回来</p>
        <h2>登录</h2>
        <p class="form-hint">没有账号？<router-link to="/register">注册</router-link> ·
          <router-link to="/forgot">忘记密码？</router-link></p>

        <label class="field">
          <span>用户名</span>
          <input v-model="form.username" type="text" autocomplete="username" placeholder="用户名" />
        </label>
        <label class="field">
          <span>密码</span>
          <input v-model="form.password" type="password" autocomplete="current-password" placeholder="输入密码" />
        </label>

        <button class="btn btn-primary btn-block" type="submit" :disabled="loading">
          {{ loading ? '登录中…' : '登录' }}
        </button>

        <div v-if="qqEnabled" class="oauth">
          <span class="oauth-line"></span>
          <span class="oauth-text">或</span>
          <span class="oauth-line"></span>
        </div>
        <a v-if="qqEnabled" class="btn btn-qq btn-block" href="/api/auth/qq">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true"><path d="M12 2C6.5 2 2 5.9 2 10.7c0 2.6 1.3 4.9 3.4 6.4l-.8 3.1c-.1.5.5.9.9.6l3.5-2.3c.9.2 1.9.4 3 .4s2.1-.1 3-.4l3.5 2.3c.4.3 1-.1.9-.6l-.8-3.1c2.1-1.5 3.4-3.8 3.4-6.4C22 5.9 17.5 2 12 2z"/></svg>
          使用 QQ 登录
        </a>
        <router-link class="doc-link" to="/register">还没有账号？立即注册 ↗</router-link>
      </form>
    </main>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { useSiteStore } from '../stores/site'
import { toast } from '../stores/toast'

const userStore = useUserStore()
const site = useSiteStore()
const router = useRouter()
const route = useRoute()
const qqEnabled = ref(false)

const features = ['自定义短码 · 访问地图 · 趋势统计', '链接审核与用户角色体系', '注册 / 登录 / 邮箱找回密码']
const form = reactive({ username: '', password: '' })
const loading = ref(false)

onMounted(() => {
  site.load().then(() => {
    qqEnabled.value = site.qqEnabled
  })
  if (route.query.oauth_error) {
    toast(String(route.query.oauth_error), 'error')
    router.replace({ query: {} })
  }
})

async function onSubmit() {
  if (!form.username.trim() || !form.password) {
    toast('请输入用户名和密码', 'warn')
    return
  }
  loading.value = true
  try {
    await userStore.login(form.username.trim(), form.password)
    toast('登录成功，欢迎回来', 'ok')
    router.push(route.query.redirect || '/dashboard')
  } catch (e) {
    // 具体错误已由 axios 拦截器 toast
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 1.12fr) minmax(380px, 0.88fr);
}

.panel-dark {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 48px;
  padding: clamp(28px, 5vw, 72px);
  background: var(--dark);
  color: #f4f4f5;
  overflow: hidden;
}
.panel-dark::after {
  content: '↗';
  position: absolute;
  right: -0.08em;
  bottom: -0.35em;
  font-size: 340px;
  line-height: 1;
  color: rgba(255, 255, 255, 0.035);
  pointer-events: none;
  user-select: none;
}
.brand { display: inline-flex; align-items: center; gap: 10px; font-size: 15px; font-weight: 600; letter-spacing: -0.01em; }
.brand-mark {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; border: 1px solid rgba(255, 255, 255, 0.35);
  border-radius: 6px; font-size: 13px;
}
.eyebrow.dark { color: rgba(255, 255, 255, 0.42); }
.panel-copy { max-width: 520px; }
.panel-copy h1 { margin-top: 22px; font-size: clamp(40px, 5.4vw, 76px); line-height: 1.04; font-weight: 500; letter-spacing: -0.035em; }
.lede { margin-top: 26px; max-width: 42ch; font-size: 15px; line-height: 1.85; color: rgba(255, 255, 255, 0.55); }

.features { position: relative; z-index: 1; border-top: 1px solid var(--dark-line); }
.features li {
  display: grid; grid-template-columns: 44px 1fr; gap: 12px;
  padding: 15px 0; border-bottom: 1px solid var(--dark-line);
  font-size: 14px; color: rgba(255, 255, 255, 0.8);
}
.features .idx { color: rgba(255, 255, 255, 0.3); font-variant-numeric: tabular-nums; font-size: 12px; padding-top: 2px; }
.foot { position: relative; z-index: 1; font-size: 12px; color: rgba(255, 255, 255, 0.3); }

.panel-light { display: flex; align-items: center; justify-content: center; padding: clamp(28px, 5vw, 72px); background: var(--bg); }
.auth-form { width: min(360px, 100%); }
.auth-form h2 { margin-top: 12px; font-size: 30px; font-weight: 600; letter-spacing: -0.02em; }
.form-hint { margin: 12px 0 30px; font-size: 13px; color: var(--ink-3); }
.form-hint a { color: var(--ink-2); text-decoration: underline; text-underline-offset: 3px; }
.oauth { display: flex; align-items: center; gap: 12px; margin: 20px 0 14px; }
.oauth-line { flex: 1; height: 1px; background: var(--line); }
.oauth-text { font-size: 12px; color: var(--ink-4); }
.btn-qq { background: var(--surface); color: #12b7f5; border-color: var(--line-strong); margin-bottom: 2px; }
.btn-qq:hover:not(:disabled) { border-color: #12b7f5; }
.doc-link { display: inline-flex; gap: 4px; margin-top: 18px; font-size: 13px; color: var(--ink-3); transition: color var(--t-fast) var(--ease); }
.doc-link:hover { color: var(--ink); }

@media (max-width: 900px) {
  .auth { grid-template-columns: 1fr; }
  .panel-dark { min-height: auto; gap: 28px; padding: 28px 24px; }
  .panel-dark::after { font-size: 180px; }
  .panel-copy h1 { font-size: 38px; }
  .features { display: none; }
  .panel-light { padding: 40px 24px; }
}
</style>
