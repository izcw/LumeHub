import axios from 'axios'

/** 从 Axios / 接口错误中提取可读文案（兼容纯文本与 JSON error 字段）。 */
export function parseApiErrorMessage(e: unknown, fallback = '请求失败'): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data
    if (typeof data === 'string') {
      const text = data.trim()
      if (text) return text
    }
    if (data && typeof data === 'object') {
      const rec = data as Record<string, unknown>
      if (typeof rec.error === 'string' && rec.error.trim()) return rec.error.trim()
      if (typeof rec.message === 'string' && rec.message.trim()) return rec.message.trim()
    }
  }
  if (e instanceof Error) {
    const msg = e.message.trim()
    if (msg && !/^Request failed with status code \d+$/i.test(msg)) {
      return msg
    }
  }
  return fallback
}
