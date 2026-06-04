/** 与 ImageViewer / 列表缩略图共享：同一 URL 预热后再次打开可减少解码停顿 */

const warmedViewerSrc = new Map<string, Promise<void>>()
const viewerNaturalSizeCache = new Map<string, { w: number; h: number }>()

export function getViewerNaturalSize(src: string) {
  return viewerNaturalSizeCache.get(src)
}

export function warmViewerImage(src: string) {
  if (!src) return
  if (!warmedViewerSrc.has(src)) {
    const p = new Promise<void>((resolve) => {
      const img = new Image()
      const done = () => resolve()
      img.onload = () => {
        if (img.naturalWidth > 0) {
          viewerNaturalSizeCache.set(src, { w: img.naturalWidth, h: img.naturalHeight })
        }
        if (typeof img.decode === 'function') {
          img.decode().then(done).catch(done)
        } else {
          done()
        }
      }
      img.onerror = done
      img.src = src
    })
    warmedViewerSrc.set(src, p)
  }
}

export function getWarmViewerPromise(src: string): Promise<void> | undefined {
  return warmedViewerSrc.get(src)
}

export function warmAdjacentViewerImages(images: string[], centerIndex: number) {
  const n = images.length
  if (n === 0) return
  warmViewerImage(images[centerIndex]!)
  const schedule =
    typeof requestIdleCallback === 'function'
      ? (cb: () => void) => requestIdleCallback(() => cb(), { timeout: 1200 })
      : (cb: () => void) => setTimeout(cb, 0)
  schedule(() => {
    if (centerIndex > 0) warmViewerImage(images[centerIndex - 1]!)
    if (centerIndex < n - 1) warmViewerImage(images[centerIndex + 1]!)
  })
}
