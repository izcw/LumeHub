const VIDEO_UPLOAD_RE = /\.(mp4|m4v|mov|webm|mkv|avi)$/i

export function isVideoUploadFile(file: File): boolean {
  const name = file.name.trim().toLowerCase()
  if (VIDEO_UPLOAD_RE.test(name)) return true
  const t = file.type.trim().toLowerCase()
  return t.startsWith('video/')
}

function pickSeekSeconds(duration: number): number[] {
  const d = Number.isFinite(duration) && duration > 0 ? duration : 0
  const picks = [0.5, 1, 2, 0.1, 0]
  if (d > 3) {
    picks.unshift(Math.min(3, d * 0.03))
  }
  const out: number[] = []
  for (const t of picks) {
    const sec = d > 0 ? Math.min(t, Math.max(0, d - 0.05)) : t
    if (!out.some((x) => Math.abs(x - sec) < 0.02)) out.push(sec)
  }
  return out
}

async function seekVideo(video: HTMLVideoElement, seconds: number): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    const timer = window.setTimeout(() => resolve(), 8000)
    const done = () => {
      window.clearTimeout(timer)
      resolve()
    }
    const fail = () => {
      window.clearTimeout(timer)
      reject(new Error('video seek failed'))
    }
    video.addEventListener('seeked', done, { once: true })
    video.addEventListener('error', fail, { once: true })
    try {
      video.currentTime = seconds
    } catch {
      fail()
    }
  })
}

function canvasMostlyBlack(ctx: CanvasRenderingContext2D, w: number, h: number): boolean {
  const step = Math.max(4, Math.floor(Math.min(w, h) / 48))
  let dark = 0
  let total = 0
  for (let y = 0; y < h; y += step) {
    for (let x = 0; x < w; x += step) {
      const p = ctx.getImageData(x, y, 1, 1).data
      const lum = 0.299 * (p[0] ?? 0) + 0.587 * (p[1] ?? 0) + 0.114 * (p[2] ?? 0)
      if (lum < 24) dark++
      total++
    }
  }
  return total > 0 && dark / total > 0.92
}

async function frameToJpegBlob(
  video: HTMLVideoElement,
  maxEdge: number,
): Promise<{ blob: Blob | null; mostlyBlack: boolean }> {
  const vw = video.videoWidth
  const vh = video.videoHeight
  if (!Number.isFinite(vw) || !Number.isFinite(vh) || vw <= 0 || vh <= 0) {
    return { blob: null, mostlyBlack: true }
  }
  const scale = Math.min(1, maxEdge / Math.max(vw, vh))
  const cw = Math.max(1, Math.round(vw * scale))
  const ch = Math.max(1, Math.round(vh * scale))
  const canvas = document.createElement('canvas')
  canvas.width = cw
  canvas.height = ch
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) return { blob: null, mostlyBlack: true }
  ctx.drawImage(video, 0, 0, cw, ch)
  const mostlyBlack = canvasMostlyBlack(ctx, cw, ch)
  const blob = await new Promise<Blob | null>((resolve) => {
    canvas.toBlob((b) => resolve(b), 'image/jpeg', 0.88)
  })
  return { blob, mostlyBlack }
}

/** 在浏览器截取视频封面（避开 0 秒黑场），供上传后作为缩略图（无需服务端 ffmpeg）。 */
export async function captureVideoPosterBlob(
  file: File,
  maxEdge = 1280,
): Promise<Blob | null> {
  if (!isVideoUploadFile(file)) return null
  if (typeof document === 'undefined' || typeof URL === 'undefined') return null

  const url = URL.createObjectURL(file)
  try {
    const video = document.createElement('video')
    video.muted = true
    video.playsInline = true
    video.preload = 'auto'
    video.setAttribute('webkit-playsinline', 'true')
    await new Promise<void>((resolve, reject) => {
      const onErr = () => reject(new Error('video load failed'))
      video.addEventListener('error', onErr, { once: true })
      video.addEventListener(
        'loadedmetadata',
        () => {
          video.removeEventListener('error', onErr)
          resolve()
        },
        { once: true },
      )
      video.src = url
    })

    const seekList = pickSeekSeconds(video.duration)
    let fallback: Blob | null = null
    for (const sec of seekList) {
      try {
        await seekVideo(video, sec)
      } catch {
        continue
      }
      const { blob, mostlyBlack } = await frameToJpegBlob(video, maxEdge)
      if (!blob) continue
      if (!mostlyBlack) return blob
      if (!fallback) fallback = blob
    }
    return fallback
  } catch {
    return null
  } finally {
    URL.revokeObjectURL(url)
  }
}
