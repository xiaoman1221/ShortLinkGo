import { defineStore } from 'pinia'
import api from '../api'

export const useSiteStore = defineStore('site', {
  state: () => ({
    name: 'ShortLinkGo',
    logo: '',
    desc: '',
    qqEnabled: false
  }),
  actions: {
    async load() {
      try {
        const res = await api.get('/api/site')
        if (res.code === 0 && res.data) {
          this.name = res.data.site_name || this.name
          this.logo = res.data.site_logo || ''
          this.desc = res.data.site_desc || ''
          this.qqEnabled = res.data.qq_enabled === '1'
          localStorage.setItem('site', JSON.stringify({ name: this.name }))
        }
      } catch (e) {
        // 站点信息加载失败时使用默认值
      }
    }
  }
})
