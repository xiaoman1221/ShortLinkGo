import { reactive } from 'vue'

// 轻量全局 toast（不依赖组件库）
export const toasts = reactive([])
let seed = 0

export function toast(message, type = 'info', duration = 2400) {
  const id = ++seed
  toasts.push({ id, message, type })
  setTimeout(() => dismiss(id), duration)
}

export function dismiss(id) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}
