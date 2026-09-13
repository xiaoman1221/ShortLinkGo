<template>
  <div class="app">
    <header class="topbar">
      <div class="wrap topbar-in">
        <router-link class="brand" to="/dashboard">
          <img v-if="site.logo" class="brand-logo" :src="site.logo" alt="logo" />
          <span v-else class="brand-mark">S</span>
          <span>{{ site.name }}</span>
        </router-link>

        <nav class="nav">
          <router-link to="/dashboard">统计</router-link>
          <router-link to="/links">链接</router-link>
          <router-link v-if="isStaff" to="/admin">系统管理</router-link>
          <a href="/docs" target="_blank">文档</a>
        </nav>

        <div class="account" ref="accountRef">
          <button class="avatar-btn" type="button" @click.stop="menuOpen = !menuOpen" aria-label="个人菜单">
            <img v-if="avatarUrl" class="avatar-img" :src="avatarUrl" alt="avatar" />
            <span v-else class="avatar-fallback">{{ initial }}</span>
          </button>
          <Transition name="pop">
            <div v-if="menuOpen" class="menu" @click.stop>
              <div class="menu-head">
                <span class="menu-name">{{ displayName }}</span>
                <span class="menu-role">{{ roleLabel }}</span>
              </div>
              <router-link to="/me" class="menu-item" @click="menuOpen = false">个人中心</router-link>
              <a class="menu-item" href="/docs" target="_blank">接口文档</a>
              <button class="menu-item danger" type="button" @click="logout">退出登录</button>
            </div>
          </Transition>
        </div>
      </div>
    </header>

    <main class="wrap main">
      <router-view />
    </main>

    <footer class="footer">
      <div class="wrap">{{ site.name }} · Go + Vue 自托管短链接服务</div>
    </footer>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { useSiteStore } from '../stores/site'

const userStore = useUserStore()
const site = useSiteStore()
const router = useRouter()

const menuOpen = ref(false)
const accountRef = ref(null)

const displayName = computed(() => userStore.user?.nickname || userStore.user?.username || '—')
const roleLabel = computed(() => userStore.roleLabel)
const isStaff = computed(() => userStore.isStaff)
const avatarUrl = computed(() => userStore.user?.avatar || '')
const initial = computed(() => (displayName.value || 'U').slice(0, 1).toUpperCase())

function onDocClick(e) {
  if (accountRef.value && !accountRef.value.contains(e.target)) menuOpen.value = false
}

function logout() {
  userStore.logout()
  router.push('/login')
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  site.load()
  userStore.fetchProfile().catch(() => {})
})
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
</script>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 40;
  background: rgba(250, 250, 250, 0.88);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--line);
}
.topbar-in {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}
.brand { display: inline-flex; align-items: center; gap: 10px; font-size: 15px; font-weight: 600; letter-spacing: -0.01em; }
.brand-logo { width: 24px; height: 24px; object-fit: contain; border-radius: 6px; }
.brand-mark {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; border: 1px solid var(--line-strong); border-radius: 6px; font-size: 13px;
}
.nav { display: flex; gap: 26px; }
.nav a {
  padding: 4px 0;
  font-size: 14px;
  color: var(--ink-3);
  transition: color var(--t-fast) var(--ease);
  border-bottom: 1px solid transparent;
  margin-bottom: -1px;
}
.nav a:hover { color: var(--ink); }
.nav a.router-link-active { color: var(--ink); border-bottom-color: var(--ink); }

.account { position: relative; display: flex; align-items: center; }
.avatar-btn {
  width: 34px; height: 34px; padding: 0; border: 1px solid var(--line-strong);
  border-radius: 50%; background: var(--surface); cursor: pointer; overflow: hidden;
  display: inline-flex; align-items: center; justify-content: center;
  transition: border-color var(--t-fast) var(--ease);
}
.avatar-btn:hover { border-color: var(--ink-3); }
.avatar-img { width: 100%; height: 100%; object-fit: cover; }
.avatar-fallback { font-size: 14px; font-weight: 600; color: var(--ink-2); }

.menu {
  position: absolute; right: 0; top: calc(100% + 10px); width: 200px;
  background: var(--surface); border: 1px solid var(--line); border-radius: var(--r-md);
  padding: 6px; z-index: 60;
}
.menu-head { padding: 10px 12px; border-bottom: 1px solid var(--line); margin-bottom: 4px; }
.menu-name { display: block; font-size: 14px; font-weight: 600; }
.menu-role { display: block; font-size: 12px; color: var(--ink-4); margin-top: 2px; }
.menu-item {
  display: block; width: 100%; text-align: left;
  padding: 9px 12px; font-size: 13.5px; color: var(--ink-2);
  background: none; border: none; border-radius: 4px; cursor: pointer;
  transition: background-color var(--t-fast) var(--ease);
}
.menu-item:hover { background: var(--surface-2); }
.menu-item.danger { color: var(--danger-ink); }

.pop-enter-active, .pop-leave-active { transition: opacity var(--t-base) var(--ease), transform var(--t-base) var(--ease); }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: translateY(-4px); }

.wrap { max-width: 1180px; margin: 0 auto; padding: 0 24px; }
.main { padding-bottom: 96px; min-height: calc(100vh - 200px); }
.footer { border-top: 1px solid var(--line); padding: 26px 0; }
.footer .wrap { font-size: 12px; color: var(--ink-4); }

@media (max-width: 860px) { .nav { gap: 16px; } }
@media (max-width: 720px) { .nav { display: none; } }
</style>
