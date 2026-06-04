const LS_DEVICE_ID = 'lumehub_device_id'

function fingerprintFromCanvas(): string {
  try {
    const canvas = document.createElement('canvas')
    canvas.width = 220
    canvas.height = 56
    const ctx = canvas.getContext('2d')
    if (!ctx) return ''
    ctx.textBaseline = 'top'
    ctx.font = '16px sans-serif'
    ctx.fillStyle = '#f60'
    ctx.fillRect(0, 0, 110, 56)
    ctx.fillStyle = '#069'
    ctx.fillText('LumeHub', 2, 16)
    ctx.strokeStyle = 'rgba(102,204,0,0.7)'
    ctx.arc(80, 24, 18, 0, Math.PI * 2)
    ctx.stroke()
    return canvas.toDataURL()
  } catch {
    return ''
  }
}

function hashStringFNV1a(input: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  const part1 = (h >>> 0).toString(16).padStart(8, '0')
  let h2 = 0x811c9dc5
  for (let i = input.length - 1; i >= 0; i--) {
    h2 ^= input.charCodeAt(i)
    h2 = Math.imul(h2, 0x01000193)
  }
  const part2 = (h2 >>> 0).toString(16).padStart(8, '0')
  return (part1 + part2).padEnd(32, '0').slice(0, 32)
}

/** 基于 canvas 指纹生成并缓存设备 ID（未登录时用于查看密码限流）。 */
export function getDeviceId(): string {
  if (typeof localStorage !== 'undefined') {
    const cached = localStorage.getItem(LS_DEVICE_ID)?.trim()
    if (cached && /^[a-f0-9]{16,128}$/i.test(cached)) {
      return cached.toLowerCase()
    }
  }
  const canvasData = fingerprintFromCanvas()
  const seed =
    canvasData ||
    [navigator.userAgent, navigator.language, screen.width, screen.height, screen.colorDepth].join('|')
  const id = hashStringFNV1a(seed)
  try {
    localStorage.setItem(LS_DEVICE_ID, id)
  } catch {
    /* ignore */
  }
  return id
}
