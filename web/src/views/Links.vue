<template>
  <div>
    <div class="page-head">
      <div>
        <p class="eyebrow">链接 · Links</p>
        <h1>短链接管理</h1>
        <p class="sub">{{ isStaff ? '管理员视角：显示全部用户的链接，可进行审核。' : '普通用户仅管理本人链接；新建链接需管理员审核通过后生效。' }}</p>
      </div>
      <span class="count-note">共 {{ total }} 条</span>
    </div>

    <div class="workspace">
      <section class="links">
        <p v-if="loading" class="state-note">加载中…</p>

        <ul v-else-if="rows.length" class="link-list">
          <li v-for="row in rows" :key="row.id" class="link-row">
            <div class="cell cell-main">
              <button class="code" type="button" :title="copyTitle(row)" @click="copyText(shortOf(row))">
                {{ row.code }}
              </button>
              <a class="short" :href="shortOf(row)" target="_blank" rel="noreferrer">{{ shortOf(row) }}</a>
            </div>

            <div class="cell cell-target">
              <a class="url" :href="row.url" target="_blank" rel="noreferrer">{{ row.url }}</a>
              <div class="row-sub">
                <span v-if="row.owner_name" class="owner">创建者：{{ row.owner_name }}</span>
                <span v-if="row.remark" class="remark">{{ row.remark }}</span>
                <span class="created">创建于 {{ formatTime(row.created_at) }}</span>
              </div>
            </div>

            <div class="cell cell-meta">
              <div class="meta status" :class="stateOf(row).key">
                <span class="dot"></span>{{ stateOf(row).label }}
              </div>
              <div class="meta"><span class="meta-label">访问</span><span class="num">{{ row.visit_count }}</span></div>
              <div class="meta"><span class="meta-label">过期</span>{{ expireText(row) }}</div>
              <div class="row-actions">
                <button class="linklike" type="button" @click="copyText(shortOf(row))">复制</button>
                <template v-if="isStaff">
                  <button v-if="row.status === 2" class="linklike ok" type="button" @click="review(row, 1)">通过</button>
                  <button v-if="row.status === 1" class="linklike" type="button" @click="review(row, 0)">停用</button>
                  <button v-if="row.status === 1" class="linklike" type="button" @click="review(row, 2)">待审核</button>
                  <button v-if="row.status === 0" class="linklike" type="button" @click="review(row, 1)">启用</button>
                </template>
                <template v-else-if="row.status !== 2">
                  <button class="linklike" type="button" @click="toggle(row)">
                    {{ row.status === 1 ? '停用' : '启用' }}
                  </button>
                </template>
                <button class="linklike danger" type="button" @click="askDelete(row)">删除</button>
              </div>
            </div>
          </li>
        </ul>

        <div v-else class="empty">
          <span class="empty-mark">↗</span>
          <p class="empty-title">还没有短链接</p>
          <p class="empty-sub">在右侧粘贴一条长链接，生成你的第一个短码。</p>
        </div>

        <div v-if="total > 0" class="pager">
          <button class="btn btn-ghost btn-sm" type="button" :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
          <span class="pager-info num">{{ pageStart }}–{{ pageEnd }} / {{ total }}</span>
          <button class="btn btn-ghost btn-sm" type="button" :disabled="pageEnd >= total" @click="changePage(page + 1)">下一页</button>
        </div>
      </section>

      <aside class="create">
        <div class="create-inner">
          <p class="eyebrow">新建 · New</p>
          <h2>创建短链接</h2>
          <form @submit.prevent="create">
            <label class="field">
              <span>目标链接</span>
              <input v-model="form.url" type="url" placeholder="https://example.com/very/long/path" required />
            </label>
            <label class="field">
              <span>自定义短码 <em class="opt">可选，留空自动生成</em></span>
              <input v-model="form.code" type="text" maxlength="32" placeholder="例如：summer-sale" />
            </label>
            <label class="field">
              <span>备注 <em class="opt">可选</em></span>
              <input v-model="form.remark" type="text" placeholder="例如：五月活动落地页" />
            </label>
            <label class="field">
              <span>过期时间 <em class="opt">可选，留空则永久</em></span>
              <input v-model="form.expire" type="datetime-local" />
            </label>
            <button class="btn btn-primary btn-block" type="submit" :disabled="creating">
              {{ creating ? '生成中…' : '生成短链接' }}
            </button>
            <p v-if="!isStaff" class="form-note">提交后状态为「待审核」，管理员通过后方可访问。</p>
          </form>

          <Transition name="fade">
            <div v-if="createdLink" class="result">
              <p class="eyebrow">已生成 · 点击复制</p>
              <div class="result-row">
                <a class="result-url" :href="shortOf(createdLink)" target="_blank" rel="noreferrer">
                  {{ shortOf(createdLink) }}
                </a>
                <button class="btn btn-ghost btn-sm" type="button" @click="copyText(shortOf(createdLink))">复制</button>
              </div>
              <p v-if="createdLink.status === 2" class="form-note">当前为待审核状态，通过后生效。</p>
            </div>
          </Transition>
        </div>
      </aside>
    </div>

    <ConfirmDialog
      v-model="showDelete"
      title="删除这条短链接？"
      :message="deleteMessage"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import api from '../api'
import { useUserStore } from '../stores/user'
import { toast } from '../stores/toast'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const userStore = useUserStore()
const isStaff = computed(() => userStore.isStaff)

const rows = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 8
const loading = ref(false)
const creating = ref(false)
const createdLink = ref(null)
const showDelete = ref(false)
const pendingDelete = ref(null)

const form = reactive({ url: '', code: '', remark: '', expire: '' })

const pageStart = computed(() => (total.value ? (page.value - 1) * pageSize + 1 : 0))
const pageEnd = computed(() => Math.min(page.value * pageSize, total.value))
const deleteMessage = computed(() =>
  pendingDelete.value
    ? '短码 ' + pendingDelete.value.code + ' 删除后不可恢复，历史访问记录将一并清除。'
    : ''
)

async function loadList() {
  loading.value = true
  try {
    const res = await api.get('/api/links', { params: { page: page.value, page_size: pageSize } })
    if (res.code === 0) {
      rows.value = res.data.list
      total.value = res.data.total
    }
  } finally {
    loading.value = false
  }
}

function changePage(p) {
  page.value = p
  loadList()
}

function absoluteURL(u) {
  if (!u) return ''
  return u.startsWith('http') ? u : window.location.origin + u
}
function shortOf(row) {
  return absoluteURL(row.short_url)
}
function stateOf(row) {
  if (row.expire_at && new Date(row.expire_at).getTime() < Date.now()) {
    return { key: 'expired', label: '已过期' }
  }
  if (row.status === 2) return { key: 'pending', label: '待审核' }
  return row.status === 1 ? { key: 'on', label: '启用中' } : { key: 'off', label: '已停用' }
}
function formatTime(v) {
  if (!v) return '—'
  return new Date(v).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', hour12: false
  })
}
function expireText(row) {
  return row.expire_at ? formatTime(row.expire_at) : '永久'
}
function copyTitle(row) {
  return '复制 ' + shortOf(row)
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

async function create() {
  if (!form.url.trim()) {
    toast('请先粘贴目标链接', 'warn')
    return
  }
  creating.value = true
  try {
    const payload = { url: form.url.trim(), remark: form.remark.trim() }
    const code = form.code.trim()
    if (code) payload.code = code
    if (form.expire) payload.expire_at = new Date(form.expire).toISOString()
    const res = await api.post('/api/links', payload)
    if (res.code === 0) {
      createdLink.value = res.data
      toast(res.msg || '短链接已生成', 'ok')
      form.url = ''
      form.code = ''
      form.remark = ''
      form.expire = ''
      page.value = 1
      await loadList()
    }
  } finally {
    creating.value = false
  }
}

async function review(row, status) {
  const res = await api.post('/api/links/' + row.id + '/review', { status })
  if (res.code === 0) {
    row.status = res.data.status
    toast(res.msg || '已更新', 'ok')
  }
}

async function toggle(row) {
  const next = row.status === 1 ? 0 : 1
  const res = await api.put('/api/links/' + row.id, { status: next })
  if (res.code === 0) {
    row.status = res.data.status
    toast(next === 1 ? '已启用' : '已停用', 'ok')
  }
}

function askDelete(row) {
  pendingDelete.value = row
  showDelete.value = true
}

async function confirmDelete() {
  const row = pendingDelete.value
  if (!row) return
  pendingDelete.value = null
  const res = await api.delete('/api/links/' + row.id)
  if (res.code === 0) {
    toast('短链接已删除', 'ok')
    if (rows.value.length === 1 && page.value > 1) page.value -= 1
    await loadList()
  }
}

onMounted(loadList)
</script>

<style scoped>
.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  padding: clamp(40px, 5vw, 64px) 0 34px;
  border-bottom: 1px solid var(--line);
  margin-bottom: 48px;
}
.page-head h1 { margin-top: 8px; font-size: 34px; font-weight: 500; letter-spacing: -0.03em; }
.page-head .sub { margin-top: 10px; font-size: 13px; color: var(--ink-3); }
.count-note { font-size: 13px; color: var(--ink-4); font-variant-numeric: tabular-nums; }

.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  gap: clamp(40px, 6vw, 88px);
  align-items: start;
}

.state-note { padding: 48px 0; color: var(--ink-4); font-size: 14px; }

.link-list { border-top: 1px solid var(--line); }
.link-row {
  display: grid;
  grid-template-columns: 148px minmax(0, 1fr) 172px;
  gap: 28px;
  align-items: center;
  padding: 20px 6px;
  border-bottom: 1px solid var(--line);
  transition: background-color var(--t-fast) var(--ease);
}
.link-row:hover { background: rgba(24, 24, 27, 0.02); }

.cell-main { min-width: 0; }
.code {
  display: block;
  max-width: 100%;
  background: none;
  border: none;
  padding: 0;
  font-family: var(--font-mono);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.01em;
  color: var(--ink);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
}
.code:hover { text-decoration: underline; text-underline-offset: 3px; }
.short {
  display: block;
  margin-top: 3px;
  font-size: 12px;
  color: var(--ink-4);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--t-fast) var(--ease);
}
.short:hover { color: var(--ink-3); }

.cell-target { min-width: 0; }
.url {
  display: block;
  font-size: 14px;
  color: var(--ink-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.url:hover { color: var(--ink); text-decoration: underline; text-underline-offset: 3px; }
.row-sub { display: flex; align-items: center; gap: 12px; margin-top: 5px; font-size: 12px; color: var(--ink-4); }
.row-sub .owner { color: var(--ink-2); }
.remark { color: var(--ink-3); }

.cell-meta { display: flex; flex-direction: column; align-items: flex-end; gap: 3px; font-size: 12.5px; color: var(--ink-3); }
.meta { display: inline-flex; align-items: center; gap: 7px; font-variant-numeric: tabular-nums; }
.meta-label { color: var(--ink-4); }
.status { font-size: 12.5px; }
.dot { width: 7px; height: 7px; border-radius: 50%; background: var(--ink-4); }
.status.on { color: var(--ok); }
.status.on .dot { background: var(--ok); }
.status.off .dot { background: var(--ink-4); }
.status.pending { color: var(--warn); }
.status.pending .dot { background: var(--warn); }
.status.expired { color: var(--ink-3); }
.status.expired .dot { background: var(--ink-4); }

.row-actions { display: flex; gap: 12px; margin-top: 7px; flex-wrap: wrap; }
.linklike {
  background: none;
  border: none;
  padding: 2px 0;
  font-size: 13px;
  color: var(--ink-3);
  cursor: pointer;
  transition: color var(--t-fast) var(--ease);
}
.linklike:hover { color: var(--ink); }
.linklike.ok { color: var(--ok); }
.linklike.ok:hover { color: var(--ok); text-decoration: underline; text-underline-offset: 3px; }
.linklike.danger:hover { color: var(--danger); }

.empty { padding: 72px 0 64px; text-align: center; }
.empty-mark { display: block; font-size: 44px; color: var(--ink-4); line-height: 1; margin-bottom: 18px; }
.empty-title { font-size: 16px; font-weight: 500; }
.empty-sub { margin-top: 6px; font-size: 13px; color: var(--ink-4); }

.pager { display: flex; align-items: center; justify-content: flex-end; gap: 16px; padding-top: 24px; }
.pager-info { font-size: 13px; color: var(--ink-4); }

.create { position: sticky; top: 84px; }
.create-inner {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  padding: 30px 28px 26px;
}
.create-inner h2 { margin: 8px 0 26px; font-size: 24px; font-weight: 600; letter-spacing: -0.02em; }
.opt { font-style: normal; font-weight: 400; color: var(--ink-4); }
.form-note { margin-top: 12px; font-size: 12px; line-height: 1.7; color: var(--warn); }

.result { margin-top: 24px; padding-top: 20px; border-top: 1px solid var(--line); }
.result-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 10px; }
.result-url {
  font-family: var(--font-mono);
  font-size: 13.5px;
  font-weight: 500;
  color: var(--ink);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.result-url:hover { text-decoration: underline; text-underline-offset: 3px; }

.fade-enter-active, .fade-leave-active { transition: opacity var(--t-base) var(--ease), transform var(--t-base) var(--ease); }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(4px); }

@media (max-width: 1080px) {
  .workspace { grid-template-columns: minmax(0, 1fr) 320px; gap: 36px; }
  .link-row { grid-template-columns: 140px minmax(0, 1fr); gap: 20px; }
  .cell-meta { grid-column: 2; flex-direction: row; flex-wrap: wrap; align-items: baseline; gap: 4px 16px; }
  .row-actions { margin-left: auto; }
}
@media (max-width: 860px) {
  .workspace { grid-template-columns: 1fr; }
  .create { position: static; order: -1; }
}
</style>
