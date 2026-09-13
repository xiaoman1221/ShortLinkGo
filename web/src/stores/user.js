import { defineStore } from 'pinia'
import api from '../api'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    user: JSON.parse(localStorage.getItem('user') || 'null')
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token),
    isStaff: (state) => state.user?.role === 'admin' || state.user?.role === 'super',
    isSuper: (state) => state.user?.role === 'super',
    roleLabel: (state) => {
      const map = { super: '超级管理员', admin: '管理员', vip: 'VIP', user: '用户' }
      return map[state.user?.role] || '用户'
    }
  },
  actions: {
    async login(username, password) {
      const res = await api.post('/api/auth/login', { username, password })
      if (res.code !== 0) throw new Error(res.msg)
      this.token = res.data.token
      this.user = res.data.user
      localStorage.setItem('token', this.token)
      localStorage.setItem('user', JSON.stringify(this.user))
    },
    async register(payload) {
      const res = await api.post('/api/auth/register', payload)
      if (res.code !== 0) throw new Error(res.msg)
    },
    async fetchProfile() {
      const res = await api.get('/api/auth/profile')
      if (res.code === 0) {
        this.user = res.data
        localStorage.setItem('user', JSON.stringify(this.user))
        return res.data
      }
      return null
    },
    setUser(user) {
      this.user = user
      localStorage.setItem('user', JSON.stringify(this.user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    }
  }
})
