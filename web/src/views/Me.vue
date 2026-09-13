<template>
  <div>
    <div class="page-head">
      <p class="eyebrow">个人中心 · Me</p>
      <h1>个人资料</h1>
    </div>

    <div class="workspace">
      <div class="main-col">
        <!-- 基本资料 -->
        <section class="panel">
          <p class="eyebrow">基本资料 · Profile</p>
          <h2>我的资料</h2>
          <form @submit.prevent="saveProfile">
            <div class="avatar-row">
              <div class="avatar-box">
                <img v-if="profile.avatar" :src="profile.avatar" class="avatar-preview" alt="avatar" />
                <span v-else class="avatar-fallback">{{ initial }}</span>
              </div>
              <div>
                <button class="btn btn-ghost btn-sm" type="button" @click="fileInput.click()">更换头像</button>
                <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/gif,image/webp" hidden @change="uploadAvatar" />
                <p class="hint">支持 png/jpg/gif/webp，不超过 2MB</p>
              </div>
            </div>
            <label class="field">
              <span>用户名</span>
              <input :value="profile.username" type="text" disabled />
              <em class="hint">用户名不可修改</em>
            </label>
            <label class="field">
              <span>昵称</span>
              <input v-model="profile.nickname" type="text" placeholder="你的昵称" />
            </label>
            <label class="field">
              <span>邮箱 <em class="opt">用于找回密码</em></span>
              <input v-model="profile.email" type="email" placeholder="you@example.com" />
            </label>
            <label class="field">
              <span>手机号</span>
              <input v-model="profile.phone" type="text" placeholder="可选" />
            </label>
            <button class="btn btn-primary" type="submit" :disabled="savingProfile">
              {{ savingProfile ? '保存中…' : '保存资料' }}
            </button>
          </form>
        </section>

        <!-- API 令牌 -->
        <section class="panel">
          <p class="eyebrow">API 令牌 · Tokens</p>
          <div class="section-head">
            <div>
              <h2>访问令牌</h2>
              <p class="desc">令牌可调用你的链接接口（增删改查），等同于你的账号权限。请妥善保管，仅在创建时展示一次。</p>
            </div>
          </div>
          <div class="token-create">
            <input v-model="tokenName" class="text-input" type="text" placeholder="令牌名称，例如：CI / 脚本" maxlength="60" @keyup.enter="createToken" />
            <button class="btn btn-primary btn-sm" type="button" :disabled="creating" @click="createToken">创建令牌</button>
          </div>

          <div v-if="newToken" class="token-new">
            <p class="eyebrow">新令牌（仅此一次）</p>
            <div class="token-new-row">
              <code class="token-code">{{ newToken }}</code>
              <button class="btn btn-ghost btn-sm" type="button" @click="copyText(newToken)">复制</button>
            </div>
          </div>

          <p v-if="loadingTokens" class="state-note">加载中…</p>
          <ul v-else-if="tokens.length" class="token-list">
            <li v-for="t in tokens" :key="t.id" class="token-row">
              <div class="token-main">
                <span class="token-name">{{ t.name }}</span>
                <span class="token-meta">创建于 {{ formatTime(t.created_at) }}<template v-if="t.last_used_at"> · 最近使用 {{ formatTime(t.last_used_at) }}</template></span>
              </div>
              <button class="linklike danger" type="button" @click="deleteToken(t)">删除</button>
            </li>
          </ul>
          <div v-else class="empty-mini"><p>还没有令牌。创建后即可用 <code>Authorization: Bearer &lt;token&gt;</code> 调用接口。</p></div>

          <div class="usage">
            <p class="eyebrow">用法示例</p>
            <pre class="code-block"><code># 列出我的短链接
curl -H "Authorization: Bearer slg_xxx" http://localhost:8080/api/links

# 创建短链接
curl -X POST http://localhost:8080/api/links \
  -H "Authorization: Bearer slg_xxx" -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","code":"my-code","remark":"demo"}'</code></pre>
            <p class="hint">详见 <a href="/docs" target="_blank">/docs 接口文档</a>。</p>
          </div>
        </section>
      </div>

      <aside class="side-col">
        <section class="panel">
          <p class="eyebrow">安全 · Security</p>
          <h2>修改密码</h2>
          <form @submit.prevent="changePassword">
            <label class="field">
              <span>原密码</span>
              <input v-model="pwd.old" type="password" autocomplete="current-password" />
            </label>
            <label class="field">
              <span>新密码</span>
              <input v-model="pwd.next" type="password" autocomplete="new-password" />
            </label>
            <label class="field">
              <span>确认新密码</span>
              <input v-model="pwd.confirm" type="password" autocomplete="new-password" />
            </label>
            <button class="btn btn-ghost btn-block" type="submit" :disabled="savingPwd">
              {{ savingPwd ? '提交中…' : '更新密码' }}
            </button>
          </form>
        </section>
        <section class="panel">
          <p class="eyebrow">身份 · Role</p>
          <h2>账号信息</h2>
          <dl class="info-list">
            <div><dt>角色</dt><dd>{{ roleLabel }}</dd></div>
            <div><dt>QQ 绑定</dt><dd>{{ profile.qq_openid ? '已绑定' : '未绑定' }}</dd></div>
            <div><dt>用户 ID</dt><dd class="num">#{{ profile.id || '—' }}</dd></div>
            <div><dt>注册时间</dt><dd>{{ formatTime(profile.created_at) }}</dd></div>
          </dl>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { useUserStore } from '../stores/user'
import { toast } from '../stores/toast'

const userStore = useUserStore()
const router = useRouter()

const profile = reactive({ id: 0, username: '', nickname: '', email: '', phone: '', avatar: '', created_at: '', qq_openid: '' })
const pwd = reactive({ old: '', next: '', confirm: '' })
const savingProfile = ref(false)
const savingPwd = ref(false)
const fileInput = ref(null)
const uploading = ref(false)

const tokens = ref([])
const tokenName = ref('')
const newToken = ref('')
const creating = ref(false)
const loadingTokens = ref(false)

const initial = computed(() => (profile.nickname || profile.username || 'U').slice(0, 1).toUpperCase())
const roleLabel = computed(() => userStore.roleLabel)

function syncProfile(u) {
  if (!u) return
  profile.id = u.id
  profile.username = u.username || ''
  profile.nickname = u.nickname || ''
  profile.email = u.email || ''
  profile.phone = u.phone || ''
  profile.avatar = u.avatar || ''
  profile.created_at = u.created_at || ''
  profile.qq_openid = u.qq_openid || ''
}

async function saveProfile() {
  savingProfile.value = true
  try {
    const res = await api.put('/api/auth/profile', {
      nickname: profile.nickname.trim(),
      email: profile.email.trim(),
      phone: profile.phone.trim()
    })
    if (res.code === 0) {
      userStore.setUser(res.data)
      syncProfile(res.data)
      toast('资料已更新', 'ok')
    }
  } finally {
    savingProfile.value = false
  }
}

async function uploadAvatar(e) {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  uploading.value = true
  try {
    const res = await api.post('/api/auth/avatar', fd)
    if (res.code === 0) {
      userStore.setUser(res.data)
      syncProfile(res.data)
      toast('头像已更新', 'ok')
    }
  } finally {
    uploading.value = false
    e.target.value = ''
  }
}

async function changePassword() {
  if (pwd.next.length < 6) {
    toast('新密码长度不能少于 6 位', 'warn')
    return
  }
  if (pwd.next !== pwd.confirm) {
    toast('两次输入的新密码不一致', 'warn')
    return
  }
  savingPwd.value = true
  try {
    const res = await api.put('/api/auth/password', { old_password: pwd.old, new_password: pwd.next })
    if (res.code === 0) {
      toast('密码已修改，请重新登录', 'ok')
      userStore.logout()
      router.push('/login')
    }
  } finally {
    savingPwd.value = false
  }
}

async function loadTokens() {
  loadingTokens.value = true
  try {
    const res = await api.get('/api/tokens')
    if (res.code === 0) tokens.value = res.data
  } finally {
    loadingTokens.value = false
  }
}

async function createToken() {
  if (!tokenName.value.trim()) {
    toast('请输入令牌名称', 'warn')
    return
  }
  creating.value = true
  try {
    const res = await api.post('/api/tokens', { name: tokenName.value.trim() })
    if (res.code === 0) {
      newToken.value = res.data.token
      tokenName.value = ''
      toast('令牌已创建，请立即复制保存', 'ok')
      await loadTokens()
    }
  } finally {
    creating.value = false
  }
}

async function deleteToken(t) {
  const res = await api.delete('/api/tokens/' + t.id)
  if (res.code === 0) {
    toast('令牌已删除', 'ok')
    await loadTokens()
  }
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text)
  } catch (e) {
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    try { document.execCommand('copy') } catch (_) {}
    document.body.removeChild(ta)
  }
  toast('已复制到剪贴板', 'ok')
}

function formatTime(v) {
  if (!v) return '—'
  return new Date(v).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', hour12: false
  })
}

onMounted(() => {
  syncProfile(userStore.user)
  userStore.fetchProfile().then((u) => {
    if (u) syncProfile(u)
  }).catch(() => {})
  loadTokens()
})
</script>

<style scoped>
.page-head { padding: clamp(40px, 5vw, 64px) 0 34px; border-bottom: 1px solid var(--line); margin-bottom: 48px; }
.page-head h1 { margin-top: 8px; font-size: 34px; font-weight: 500; letter-spacing: -0.03em; }

.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: clamp(36px, 5vw, 72px);
  align-items: start;
}
.main-col, .side-col { display: flex; flex-direction: column; gap: 56px; min-width: 0; }
.panel h2 { margin: 8px 0 24px; font-size: 24px; font-weight: 600; letter-spacing: -0.02em; }
.panel .field { max-width: 480px; }
.section-head { display: flex; align-items: flex-end; justify-content: space-between; }
.section-head .desc { margin-top: 10px; font-size: 13px; color: var(--ink-3); max-width: 56ch; }

.avatar-row { display: flex; align-items: center; gap: 18px; margin-bottom: 22px; }
.avatar-box { width: 72px; height: 72px; border-radius: 50%; border: 1px solid var(--line-strong); overflow: hidden; display: inline-flex; align-items: center; justify-content: center; background: var(--surface); }
.avatar-preview { width: 100%; height: 100%; object-fit: cover; }
.avatar-fallback { font-size: 26px; font-weight: 600; color: var(--ink-2); }
.hint { display: block; margin-top: 6px; font-size: 12px; color: var(--ink-4); }
.opt { font-style: normal; font-weight: 400; color: var(--ink-4); }
.panel input[disabled] { background: var(--surface-2); color: var(--ink-4); border-color: var(--line); }

/* tokens */
.text-input {
  height: 36px; padding: 0 12px; border: 1px solid var(--line-strong); border-radius: var(--r-sm);
  background: var(--surface); color: var(--ink); font-size: 14px; min-width: 0; flex: 1;
}
.text-input:focus { outline: none; border-color: var(--ink); }
.token-create { display: flex; gap: 10px; margin-bottom: 16px; }
.token-new { margin-bottom: 18px; padding: 14px; border: 1px solid var(--line); border-radius: var(--r-md); background: var(--surface); }
.token-new-row { display: flex; align-items: center; gap: 10px; margin-top: 8px; }
.token-code { font-family: var(--font-mono); font-size: 13px; word-break: break-all; flex: 1; }

.token-list { border-top: 1px solid var(--line); }
.token-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 13px 2px; border-bottom: 1px solid var(--line); }
.token-name { display: block; font-size: 14px; font-weight: 500; }
.token-meta { display: block; font-size: 12px; color: var(--ink-4); margin-top: 2px; }
.linklike { background: none; border: none; padding: 2px 0; font-size: 13px; color: var(--ink-3); cursor: pointer; }
.linklike.danger:hover { color: var(--danger); }
.empty-mini { padding: 20px 0; color: var(--ink-4); font-size: 13px; line-height: 1.8; }
.empty-mini code { font-family: var(--font-mono); font-size: 12px; background: var(--surface-2); padding: 1px 5px; border-radius: 4px; }
.state-note { padding: 24px 0; color: var(--ink-4); font-size: 14px; }

.usage { margin-top: 26px; padding-top: 22px; border-top: 1px solid var(--line); }
.code-block {
  margin: 12px 0; padding: 16px; background: var(--dark); color: #e4e4e7; border-radius: var(--r-md);
  font-size: 12.5px; line-height: 1.8; overflow-x: auto;
}
.code-block code { font-family: var(--font-mono); }
.usage .hint a { color: var(--ink-2); text-decoration: underline; text-underline-offset: 3px; }

.info-list { border-top: 1px solid var(--line); }
.info-list div { display: flex; justify-content: space-between; padding: 12px 2px; border-bottom: 1px solid var(--line); font-size: 14px; }
.info-list dt { color: var(--ink-4); }
.info-list dd { color: var(--ink-2); }

@media (max-width: 1000px) {
  .workspace { grid-template-columns: 1fr; }
}
</style>
