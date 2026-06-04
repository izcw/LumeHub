import { GALLERY_CATEGORY_OPTIONS } from '@/stores/gallerySearch'

export type GalleryMediaKind = 'image' | 'video' | 'audio' | 'archive' | 'document' | 'other'

const IMAGE_EXT_LIST = GALLERY_CATEGORY_OPTIONS.find((c) => c.id === 'image')!.extensions
const IMAGE_EXTS = new Set<string>()
for (const e of IMAGE_EXT_LIST) {
  IMAGE_EXTS.add(e === 'jpeg' ? 'jpg' : e)
}

const EXT_TO_KIND = new Map<string, Exclude<GalleryMediaKind, 'image'>>()
for (const cat of GALLERY_CATEGORY_OPTIONS) {
  if (cat.id === 'image') continue
  const id = cat.id as Exclude<GalleryMediaKind, 'image'>
  for (const e of cat.extensions) {
    EXT_TO_KIND.set(e, id)
  }
}

/** 与筛选逻辑一致：从 URL 路径取扩展名，jpeg→jpg */
export function galleryExtFromUrl(url: string): string {
  const pathOnly = (url.split('?')[0] || '').split('#')[0] || ''
  const seg = pathOnly.split(/[/\\]/).pop() || ''
  const m = seg.match(/\.([a-z0-9]+)$/i)
  let ext = (m?.[1] || '').toLowerCase()
  if (ext === 'jpeg') ext = 'jpg'
  return ext
}

export function galleryMediaKindFromUrl(url: string): GalleryMediaKind {
  if (!url) return 'image'
  const ext = galleryExtFromUrl(url)
  if (!ext) return 'other'
  if (IMAGE_EXTS.has(ext)) return 'image'
  return EXT_TO_KIND.get(ext) ?? 'other'
}

export function isGalleryImageUrl(url: string): boolean {
  return galleryMediaKindFromUrl(url) === 'image'
}

export function isGalleryImageItem(item: { fullSrc: string; mediaKind?: string }): boolean {
  const kind = item.mediaKind?.trim().toLowerCase()
  if (kind === 'video' || isGalleryVideoUrl(item.fullSrc)) return false
  if (kind === 'image') return true
  return isGalleryImageUrl(item.fullSrc)
}

export function isGalleryVideoUrl(url: string): boolean {
  return galleryMediaKindFromUrl(url) === 'video'
}

const MEDIA_KIND_LABEL = new Map<string, string>(
  GALLERY_CATEGORY_OPTIONS.map((c) => [c.id, c.label]),
)
MEDIA_KIND_LABEL.set('other', '其他')

export function galleryMediaKindLabel(kind: GalleryMediaKind | string | undefined): string {
  if (!kind) return '其他'
  return MEDIA_KIND_LABEL.get(kind) ?? '其他'
}
