import axios from 'axios'
import { toast } from '../stores/toast'

const api = axios.create({ baseURL: '/', timeout: 10000 })

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (res) => res.data,
  (err) => {
    const msg = err.response?.data?.msg || '请求失败，请稍后重试'
    toast(msg, 'error')
    return Promise.reject(err)
  }
)

export default api
