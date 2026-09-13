<template>
  <div>
    <div class="page-head">
      <div>
        <p class="eyebrow">系统管理 · Admin</p>
        <h1>系统管理</h1>
        <p class="sub">{{ isSuper ? '超级管理员：可管理用户角色与封禁。' : '管理员：可查看用户与配置系统。' }}</p>
      </div>
    </div>

    <div class="tabs">
      <button type="button" :class="{ active: tab === 'users' }" @click="tab = 'users'">用户管理</button>
      <button type="button" :class="{ active: tab === 'site' }" @click="tab = 'site'">网站信息</button>
      <button type="button" :class="{ active: tab === 'smtp' }" @click="tab = 'smtp'">SMTP 邮件</button>
      <button type="button" :class="{ active: tab === 'qq' }" @click="tab = 'qq'">QQ 登录</button>
      <button type="button" :class="{ active: tab === 'geo' }" @click="openGeo">访问地图</button>
    </div>

    <!-- 用户管理 -->
    <section v-if="tab === 'users'" class="panel">
      <div class="toolbar">
        <input v-model="keyword" class="text-input" type="text" placeholder="搜索用户名 / 昵称 / 邮箱" @keyup.enter="reloadUsers(1)" />
        <button class="btn btn-ghost btn-sm" type="button" @click="reloadUsers(1)">搜索</button>
      </div>
      <p v-if="!isSuper" class="note">仅超级管理员可修改角色与封禁状态。</p>
      <p v-if="loadingUsers" class="state-note">加载中…</p>
      <table v-else class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>用户</th>
            <th>昵称</th>
            <th>邮箱</th>
            <th>角色</th>
            <th>状态</th>
            <th>注册时间</th>
            <th v-if="isSuper">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td class="num mono">#{{ u.id }}</td>
            <td class="strong">{{ u.username }}</td>
            <td>{{ u.nickname || '—' }}</td>
            <td class="dim">{{ u.email || '—' }}</td>
            <td>
              <select v-if="isSuper && u.id !== myId" class="select" :value="u.role" @change="setRole(u, $event.target.value)">
                <option value="super">超级管理员</option>
                <option value="admin">管理员</option>
                <option value="vip">VIP</option>
                <option value="user">用户</option>
              </select>
              <span v-else>{{ roleName(u.role) }}</span>
            </td>
            <td>
              <span v-if="isSuper && u.id !== myId" class="status-actions">
                <button v-if="u.status === 1" class="linklike danger" type="button" @click="setStatus(u, 0)">封禁</button>
                <button v-else class="linklike ok" type="button" @click="setStatus(u, 1)">解封</button>
              </span>
              <span v-else :class="u.status === 1 ? 'ok-text' : 'danger-text'">{{ u.status === 1 ? '正常' : '已封禁' }}</span>
            </td>
            <td class="dim">{{ formatTime(u.created_at) }}</td>
            <td v-if="isSuper"></td>
          </tr>
        </tbody>
      </table>
      <div v-if="userTotal > 0" class="pager">
        <button class="btn btn-ghost btn-sm" type="button" :disabled="userPage <= 1" @click="reloadUsers(userPage - 1)">上一页</button>
        <span class="pager-info num">{{ userPage }} / {{ Math.max(1, Math.ceil(userTotal / userPageSize)) }}</span>
        <button class="btn btn-ghost btn-sm" type="button" :disabled="userPage * userPageSize >= userTotal" @click="reloadUsers(userPage + 1)">下一页</button>
      </div>
    </section>

    <!-- 网站信息 -->
    <section v-if="tab === 'site'" class="panel narrow">
      <p class="eyebrow">网站信息 · Site</p>
      <h2>基本设置</h2>
      <form @submit.prevent="saveSite">
        <div class="logo-row">
          <div class="logo-box">
            <img v-if="siteForm.site_logo" :src="siteForm.site_logo" alt="logo" />
            <span v-else class="logo-fallback">S</span>
          </div>
          <div>
            <button class="btn btn-ghost btn-sm" type="button" @click="logoInput.click()">上传 Logo</button>
            <input ref="logoInput" type="file" accept="image/png,image/jpeg,image/gif,image/webp" hidden @change="uploadLogo" />
            <p class="hint">显示在菜单栏左侧；不设置则显示默认字母标</p>
          </div>
        </div>
        <label class="field">
          <span>站点名称</span>
          <input v-model="siteForm.site_name" type="text" placeholder="ShortLinkGo" />
        </label>
        <label class="field">
          <span>站点简介</span>
          <input v-model="siteForm.site_desc" type="text" placeholder="一句话介绍" />
        </label>
        <button class="btn btn-primary" type="submit" :disabled="savingSite">{{ savingSite ? '保存中…' : '保存设置' }}</button>
      </form>
    </section>

    <!-- SMTP -->
    <section v-if="tab === 'smtp'" class="panel narrow">
      <p class="eyebrow">邮件 · SMTP</p>
      <h2>SMTP 配置</h2>
      <p class="desc">用于发送「找回密码」等系统邮件。465 端口为隐式 TLS，25/587 自动 STARTTLS。</p>
      <form @submit.prevent="saveSMTP">
        <div class="row">
          <label class="field">
            <span>主机</span>
            <input v-model="smtpForm.smtp_host" type="text" placeholder="smtp.qq.com" />
          </label>
          <label class="field">
            <span>端口</span>
            <input v-model="smtpForm.smtp_port" type="text" placeholder="465" />
          </label>
        </div>
        <label class="field">
          <span>账号</span>
          <input v-model="smtpForm.smtp_user" type="text" autocomplete="off" placeholder="your@example.com" />
        </label>
        <label class="field">
          <span>密码 / 授权码 <em class="opt">{{ smtpPassSet ? '（已设置，留空保持不变）' : '' }}</em></span>
          <input v-model="smtpForm.smtp_pass" type="password" autocomplete="new-password" placeholder="授权码或密码" />
        </label>
        <label class="field">
          <span>发件人地址</span>
          <input v-model="smtpForm.smtp_from" type="text" placeholder="ShortLinkGo <noreply@example.com>" />
        </label>
        <div class="actions">
          <button class="btn btn-primary" type="submit" :disabled="savingSMTP">{{ savingSMTP ? '保存中…' : '保存' }}</button>
        </div>
      </form>
      <div class="test-box">
        <p class="eyebrow">测试 · Test</p>
        <div class="token-create">
          <input v-model="testTo" class="text-input" type="email" placeholder="收件邮箱" />
          <button class="btn btn-ghost btn-sm" type="button" :disabled="testing" @click="testSMTP">{{ testing ? '发送中…' : '发送测试邮件' }}</button>
        </div>
      </div>
    </section>

    <!-- QQ 登录 -->
    <section v-if="tab === 'qq'" class="panel narrow">
      <p class="eyebrow">第三方登录 · QQ</p>
      <h2>QQ 互联配置</h2>
      <p class="desc">前往 <a href="https://connect.qq.com/" target="_blank">QQ 互联</a> 创建网站应用，获取 App ID 与 App Key。授权回调地址填写：<code class="inline-code">{{ callbackHint }}</code></p>
      <form @submit.prevent="saveQQ">
        <label class="field">
          <span>App ID</span>
          <input v-model="qqForm.qq_app_id" type="text" autocomplete="off" placeholder="QQ 互联 App ID" />
        </label>
        <label class="field">
          <span>App Key <em class="opt">{{ qqKeySet ? '（已设置，留空保持不变）' : '' }}</em></span>
          <input v-model="qqForm.qq_app_key" type="password" autocomplete="new-password" placeholder="QQ 互联 App Key" />
        </label>
        <button class="btn btn-primary" type="submit" :disabled="savingQQ">{{ savingQQ ? '保存中…' : '保存配置' }}</button>
      </form>
      <p class="note qq-note">{{ qqConfigured ? 'QQ 登录已启用，登录页将显示「使用 QQ 登录」按钮。' : '配置 App ID / App Key 后，登录页将出现 QQ 登录入口。' }}</p>
    </section>

    <!-- 访问地图（GeoIP） -->
    <section v-if="tab === 'geo'" class="panel narrow">
      <p class="eyebrow">访问地图 · GeoIP</p>
      <h2>IP 地理数据库</h2>
      <p class="desc">
        启用后服务会自动下载 MaxMind GeoLite2-City 数据库并每小时检查更新，用于访问统计的国家/省份解析。
        下载源<strong>无需手动填写</strong>：系统自动在候选源之间探测，用第一个可用的源下载，某个源中断会自动切换下一个；
        候选源为「社区镜像（每日更新，无需注册）」与「MaxMind 官方（需 License Key）」。
      </p>
      <form @submit.prevent="saveGeo">
        <label class="switch-field">
          <input v-model="geoForm.geoip_enabled" type="checkbox" />
          <span>启用自动下载与每小时更新</span>
        </label>
        <label class="field">
          <span>MaxMind License Key <em class="opt">{{ geoKeySet ? '（已设置，留空保持不变）' : '' }}</em></span>
          <input v-model="geoForm.geoip_license_key" type="password" autocomplete="new-password" placeholder="使用 MaxMind 官方源时必填；社区镜像留空" />
        </label>
        <label class="field">
          <span>手动指定下载地址 <em class="opt">（高级，留空 = 自动选择可用源）</em></span>
          <input v-model="geoForm.geoip_url" type="text" autocomplete="off" placeholder="留空自动选择；也可强制指定 .mmdb 或 tar.gz 直链" />
        </label>
        <div class="actions">
          <button class="btn btn-primary" type="submit" :disabled="savingGeo">{{ savingGeo ? '保存中…' : '保存配置' }}</button>
          <button class="btn btn-ghost" type="button" :disabled="geoStatus.downloading" @click="triggerGeoUpdate">
            {{ geoStatus.downloading ? '更新中…' : '立即检查更新' }}
          </button>
        </div>
      </form>

      <div class="test-box">
        <p class="eyebrow">真实访客 IP · Client IP</p>
        <p class="desc" style="margin-top: 12px">
          决定跳转统计中如何获取访客真实 IP。直连地址指与服务器直接通信的地址（反代/容器场景为代理地址）。
        </p>
        <form @submit.prevent="saveIPMode">
          <label class="field">
            <span>识别模式</span>
            <select v-model="ipForm.client_ip_mode" class="select" style="height: 38px">
              <option value="smart">智能（推荐）— 部署在反代/容器后自动读取转发头，裸机直连时防伪造</option>
              <option value="always">始终读转发头 — Cloudflare 等回源 IP 为公网的场景（可被伪造）</option>
              <option value="direct">仅直连地址 — 完全不读转发头</option>
              <option value="cidr">可信代理段 — 直连地址命中下方代理段时才读转发头</option>
            </select>
          </label>
          <label class="field">
            <span>可信代理段 <em class="opt">（仅 cidr 模式使用，CIDR 或单 IP，逗号分隔）</em></span>
            <input v-model="ipForm.client_ip_trusted_proxies" type="text" autocomplete="off" placeholder="如 173.245.48.0/20, 10.0.0.5" />
          </label>
          <div class="actions">
            <button class="btn btn-primary" type="submit" :disabled="savingIP">{{ savingIP ? '保存中…' : '保存 IP 识别设置' }}</button>
          </div>
        </form>
      </div>

      <div class="test-box">
        <p class="eyebrow">状态 · Status</p>
        <p v-if="loadingGeoStatus" class="state-note">加载中…</p>
        <dl v-else class="geo-status">
          <div><dt>数据库</dt><dd :class="geoStatus.loaded ? 'ok-text' : ''">{{ geoStatus.loaded ? '已加载' : '未加载' }}</dd></div>
          <div><dt>下载模式</dt><dd>{{ geoStatus.auto_source ? '自动选择可用源' : '手动指定' }}</dd></div>
          <div v-if="geoStatus.source"><dt>上次使用源</dt><dd>{{ geoStatus.source }}</dd></div>
          <div v-if="geoStatus.file_exists"><dt>文件大小</dt><dd class="num">{{ (geoStatus.file_size / 1048576).toFixed(1) }} MB</dd></div>
          <div v-if="geoStatus.file_exists"><dt>文件更新时间</dt><dd>{{ formatTime(geoStatus.file_mod_time) }}</dd></div>
          <div><dt>上次检查</dt><dd>{{ geoStatus.last_check ? formatTime(geoStatus.last_check) : '—' }}</dd></div>
          <div v-if="geoStatus.last_success"><dt>上次更新成功</dt><dd>{{ formatTime(geoStatus.last_success) }}</dd></div>
          <div v-if="geoStatus.last_error" class="err"><dt>上次错误</dt><dd class="danger-text">{{ geoStatus.last_error }}</dd></div>
        </dl>
        <button class="btn btn-ghost btn-sm" type="button" @click="loadGeoStatus">刷新状态</button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import api from '../api'
import { useUserStore } from '../stores/user'
import { useSiteStore } from '../stores/site'
import { toast } from '../stores/toast'

const userStore = useUserStore()
const site = useSiteStore()

const isSuper = computed(() => userStore.isSuper)
const myId = computed(() => userStore.user?.id)

const tab = ref('users')

/* users */
const users = ref([])
const userTotal = ref(0)
const userPage = ref(1)
const userPageSize = 15
const keyword = ref('')
const loadingUsers = ref(false)

/* site */
const siteForm = reactive({ site_name: '', site_desc: '', site_logo: '' })
const savingSite = ref(false)
const logoInput = ref(null)

/* smtp */
const smtpForm = reactive({ smtp_host: '', smtp_port: '', smtp_user: '', smtp_pass: '', smtp_from: '' })
const smtpPassSet = ref(false)
const savingSMTP = ref(false)
const testing = ref(false)
const testTo = ref('')

const qqForm = reactive({ qq_app_id: '', qq_app_key: '' })
const qqKeySet = ref(false)
const savingQQ = ref(false)
const qqConfigured = ref(false)
const callbackHint = ref('http://localhost:8080/api/auth/qq/callback')

/* geoip */
const geoForm = reactive({ geoip_enabled: false, geoip_url: '', geoip_license_key: '' })
const geoKeySet = ref(false)
const savingGeo = ref(false)
const geoStatus = ref({})
const loadingGeoStatus = ref(false)

/* client ip */
const ipForm = reactive({ client_ip_mode: 'smart', client_ip_trusted_proxies: '' })
const savingIP = ref(false)

const ROLE_NAMES = { super: '超级管理员', admin: '管理员', vip: 'VIP', user: '用户' }
function roleName(r) {
  return ROLE_NAMES[r] || r || '用户'
}

async function loadUsers() {
  loadingUsers.value = true
  try {
    const res = await api.get('/api/admin/users', {
      params: { page: userPage.value, page_size: userPageSize, keyword: keyword.value.trim() }
    })
    if (res.code === 0) {
      users.value = res.data.list
      userTotal.value = res.data.total
    }
  } finally {
    loadingUsers.value = false
  }
}
function reloadUsers(p) {
  if (p < 1) return
  userPage.value = p
  loadUsers()
}

async function setRole(u, role) {
  const res = await api.put('/api/admin/users/' + u.id + '/role', { role })
  if (res.code === 0) {
    u.role = role
    toast('角色已更新为「' + roleName(role) + '」', 'ok')
  }
}

async function setStatus(u, status) {
  const res = await api.put('/api/admin/users/' + u.id + '/status', { status })
  if (res.code === 0) {
    u.status = status
    toast(status === 0 ? '已封禁该用户' : '已解封', 'ok')
  }
}

/* settings load/save */
async function loadSettings() {
  const res = await api.get('/api/settings')
  if (res.code === 0) {
    siteForm.site_name = res.data.site_name || ''
    siteForm.site_desc = res.data.site_desc || ''
    siteForm.site_logo = res.data.site_logo || ''
    smtpForm.smtp_host = res.data.smtp_host || ''
    smtpForm.smtp_port = res.data.smtp_port || ''
    smtpForm.smtp_user = res.data.smtp_user || ''
    smtpForm.smtp_from = res.data.smtp_from || ''
    smtpPassSet.value = res.data.smtp_pass_set === '1'
    qqForm.qq_app_id = res.data.qq_app_id || ''
    qqForm.qq_app_key = ''
    qqKeySet.value = res.data.qq_app_key_set === '1'
    qqConfigured.value = Boolean(qqForm.qq_app_id) && qqKeySet.value
    callbackHint.value = window.location.origin + '/api/auth/qq/callback'
    geoForm.geoip_enabled = res.data.geoip_enabled === '1'
    geoForm.geoip_url = res.data.geoip_url || ''
    geoForm.geoip_license_key = ''
    geoKeySet.value = res.data.geoip_license_key_set === '1'
    ipForm.client_ip_mode = res.data.client_ip_mode || 'smart'
    ipForm.client_ip_trusted_proxies = res.data.client_ip_trusted_proxies || ''
  }
}

async function saveSite() {
  savingSite.value = true
  try {
    const res = await api.put('/api/settings', {
      site_name: siteForm.site_name.trim(),
      site_desc: siteForm.site_desc.trim()
    })
    if (res.code === 0) {
      site.load()
      toast('网站信息已保存', 'ok')
    }
  } finally {
    savingSite.value = false
  }
}

async function uploadLogo(e) {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res = await api.post('/api/settings/logo', fd)
    if (res.code === 0) {
      siteForm.site_logo = res.data.site_logo
      site.load()
      toast('Logo 已更新', 'ok')
    }
  } finally {
    e.target.value = ''
  }
}

async function saveSMTP() {
  savingSMTP.value = true
  try {
    const res = await api.put('/api/settings', {
      smtp_host: smtpForm.smtp_host.trim(),
      smtp_port: smtpForm.smtp_port.trim(),
      smtp_user: smtpForm.smtp_user.trim(),
      smtp_pass: smtpForm.smtp_pass,
      smtp_from: smtpForm.smtp_from.trim()
    })
    if (res.code === 0) {
      smtpForm.smtp_pass = ''
      smtpPassSet.value = true
      toast('SMTP 设置已保存', 'ok')
    }
  } finally {
    savingSMTP.value = false
  }
}

async function testSMTP() {
  if (!testTo.value.trim()) {
    toast('请输入收件邮箱', 'warn')
    return
  }
  testing.value = true
  try {
    const res = await api.post('/api/settings/smtp/test', { to: testTo.value.trim() })
    if (res.code === 0) toast('测试邮件已发送', 'ok')
  } finally {
    testing.value = false
  }
}

async function saveQQ() {
  savingQQ.value = true
  try {
    const res = await api.put('/api/settings', {
      qq_app_id: qqForm.qq_app_id.trim(),
      qq_app_key: qqForm.qq_app_key
    })
    if (res.code === 0) {
      qqForm.qq_app_key = ''
      qqKeySet.value = Boolean(qqForm.qq_app_id.trim())
      qqConfigured.value = Boolean(qqForm.qq_app_id.trim()) && qqKeySet.value
      site.load()
      toast('QQ 配置已保存', 'ok')
    }
  } finally {
    savingQQ.value = false
  }
}

/* geoip save/status */
async function saveGeo() {
  savingGeo.value = true
  try {
    const res = await api.put('/api/settings', {
      geoip_enabled: geoForm.geoip_enabled ? '1' : '0',
      geoip_url: geoForm.geoip_url.trim(),
      geoip_license_key: geoForm.geoip_license_key
    })
    if (res.code === 0) {
      geoForm.geoip_license_key = ''
      geoKeySet.value = Boolean(geoForm.geoip_license_key) || geoKeySet.value
      toast('GeoIP 配置已保存', 'ok')
      loadGeoStatus()
    }
  } finally {
    savingGeo.value = false
  }
}

async function loadGeoStatus() {
  loadingGeoStatus.value = true
  try {
    const res = await api.get('/api/settings/geoip/status')
    if (res.code === 0) geoStatus.value = res.data
  } finally {
    loadingGeoStatus.value = false
  }
}

async function triggerGeoUpdate() {
  const res = await api.post('/api/settings/geoip/update')
  if (res.code === 0) {
    toast(res.msg || '已开始后台更新', 'ok')
    setTimeout(loadGeoStatus, 1500)
  }
}

async function saveIPMode() {
  savingIP.value = true
  try {
    const res = await api.put('/api/settings', {
      client_ip_mode: ipForm.client_ip_mode,
      client_ip_trusted_proxies: ipForm.client_ip_trusted_proxies.trim()
    })
    if (res.code === 0) toast('IP 识别设置已保存', 'ok')
  } finally {
    savingIP.value = false
  }
}

function openGeo() {
  tab.value = 'geo'
  loadGeoStatus()
}

function formatTime(v) {
  if (!v) return '—'
  return new Date(v).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}

onMounted(() => {
  loadSettings()
  loadUsers()
})
</script>

<style scoped>
.page-head { padding: clamp(40px, 5vw, 64px) 0 34px; border-bottom: 1px solid var(--line); margin-bottom: 34px; }
.page-head h1 { margin-top: 8px; font-size: 34px; font-weight: 500; letter-spacing: -0.03em; }
.page-head .sub { margin-top: 10px; font-size: 13px; color: var(--ink-3); }

.tabs { display: flex; gap: 6px; border-bottom: 1px solid var(--line); margin-bottom: 34px; }
.tabs button {
  padding: 10px 16px; font-size: 14px; color: var(--ink-3); background: none; border: none;
  cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px;
  transition: color var(--t-fast) var(--ease), border-color var(--t-fast) var(--ease);
}
.tabs button:hover { color: var(--ink); }
.tabs button.active { color: var(--ink); border-bottom-color: var(--ink); }

.panel { min-width: 0; }
.panel.narrow { max-width: 720px; }
.panel h2 { margin: 8px 0 22px; font-size: 24px; font-weight: 600; letter-spacing: -0.02em; }
.panel .desc { margin-bottom: 22px; font-size: 13px; color: var(--ink-3); max-width: 60ch; line-height: 1.8; }
.note { margin-bottom: 14px; font-size: 13px; color: var(--warn); }
.state-note { padding: 40px 0; color: var(--ink-4); font-size: 14px; }

.toolbar { display: flex; gap: 10px; margin-bottom: 18px; }
.text-input {
  height: 36px; padding: 0 12px; border: 1px solid var(--line-strong); border-radius: var(--r-sm);
  background: var(--surface); color: var(--ink); font-size: 14px; min-width: 0; flex: 1; max-width: 320px;
}
.text-input:focus { outline: none; border-color: var(--ink); }

.table { width: 100%; border-collapse: collapse; font-size: 13.5px; }
.table th {
  text-align: left; font-weight: 500; color: var(--ink-4); font-size: 12px; letter-spacing: 0.04em;
  padding: 10px 8px; border-bottom: 1px solid var(--line-strong);
}
.table td { padding: 12px 8px; border-bottom: 1px solid var(--line); vertical-align: middle; }
.table tr:hover td { background: rgba(24, 24, 27, 0.015); }
.mono { font-family: var(--font-mono); }
.strong { font-weight: 500; }
.dim { color: var(--ink-3); }
.select {
  height: 30px; padding: 0 8px; font-size: 13px; color: var(--ink); background: var(--surface);
  border: 1px solid var(--line-strong); border-radius: var(--r-sm); cursor: pointer;
}
.select:focus { outline: none; border-color: var(--ink); }
.ok-text { color: var(--ok); }
.danger-text { color: var(--danger-ink); }
.linklike { background: none; border: none; padding: 2px 0; font-size: 13px; color: var(--ink-3); cursor: pointer; }
.linklike.ok { color: var(--ok); }
.linklike.danger { color: var(--danger-ink); }
.pager { display: flex; align-items: center; justify-content: flex-end; gap: 14px; padding-top: 18px; }
.pager-info { font-size: 13px; color: var(--ink-4); }

.logo-row { display: flex; align-items: center; gap: 18px; margin-bottom: 24px; }
.logo-box {
  width: 56px; height: 56px; border: 1px solid var(--line-strong); border-radius: 12px; overflow: hidden;
  display: inline-flex; align-items: center; justify-content: center; background: var(--surface);
}
.logo-box img { width: 100%; height: 100%; object-fit: contain; }
.logo-fallback { font-size: 22px; font-weight: 600; }
.hint { display: block; margin-top: 6px; font-size: 12px; color: var(--ink-4); }
.opt { font-style: normal; font-weight: 400; color: var(--ink-4); }

.inline-code { font-family: var(--font-mono); font-size: 12.5px; background: var(--surface-2); border: 1px solid var(--line); border-radius: 4px; padding: 1px 6px; word-break: break-all; }
.qq-note { margin-top: 18px; }
.row { display: grid; grid-template-columns: minmax(0, 1fr) 140px; gap: 16px; }
.actions { display: flex; gap: 12px; }
.test-box { margin-top: 34px; padding-top: 26px; border-top: 1px solid var(--line); }
.test-box .token-create { display: flex; gap: 10px; margin-top: 14px; }

.switch-field { display: flex; align-items: center; gap: 10px; margin-bottom: 22px; font-size: 14px; cursor: pointer; }
.switch-field input { width: 16px; height: 16px; accent-color: var(--ink); cursor: pointer; }
.geo-status { display: grid; gap: 10px; margin: 16px 0 18px; }
.geo-status > div { display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 14px; font-size: 13.5px; }
.geo-status dt { color: var(--ink-4); }
.geo-status dd { min-width: 0; word-break: break-all; }
.geo-status .err dd { color: var(--danger-ink); }
</style>
