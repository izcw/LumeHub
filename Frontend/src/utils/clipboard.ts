/** 复制文本到剪贴板，HTTP 等非安全上下文下回退到 execCommand */
export async function copyTextToClipboard(text: string): Promise<boolean> {
  const value = text.trim()
  if (!value) return false

  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      /* 非 HTTPS / 无权限时回退 */
    }
  }

  try {
    const ta = document.createElement('textarea')
    ta.value = value
    ta.setAttribute('readonly', 'true')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    ta.style.top = '0'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
