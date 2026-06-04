import { isGalleryVideoUrl } from '@/utils/galleryMedia'

export function isGalleryVideoItem(item: { fullSrc: string; mediaKind?: string }) {
  if (item.mediaKind?.trim().toLowerCase() === 'video') return true
  return isGalleryVideoUrl(item.fullSrc)
}

export function formatCardDateFromIso(iso?: string): string {
  const raw = iso?.trim()
  if (!raw) return ''
  const t = Date.parse(raw)
  if (Number.isNaN(t)) return ''
  const d = new Date(t)
  return `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`
}

export function linkNameFromResourceUrl(url: string): string {
  const path = (url.split('?')[0] ?? '').trim()
  const seg = path.split('/').pop() ?? ''
  try {
    return decodeURIComponent(seg)
  } catch {
    return seg
  }
}

export function resolveGalleryItemContext(
  rowId: string,
  defaultFolderKey: string,
): { folderKey: string; itemId: string } {
  const idx = rowId.indexOf(':')
  if (idx > 0) {
    return { folderKey: rowId.slice(0, idx), itemId: rowId.slice(idx + 1) }
  }
  return { folderKey: defaultFolderKey, itemId: rowId }
}

export function toAbsoluteResourceUrl(url: string): string {
  const raw = url.trim()
  if (!raw) return ''
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw
  const path = raw.startsWith('/') ? raw : `/${raw}`
  return `${window.location.origin}${path}`
}
