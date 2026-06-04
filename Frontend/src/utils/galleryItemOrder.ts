import type { GalleryRemoteRow } from '@/stores/gallerySearch'
import { rankSortField } from '@/utils/sortRank'

export type GalleryItemSortMode = 'uploaded_at' | 'updated_at' | 'sort'

export function normalizeGalleryItemSortMode(raw: string | undefined): GalleryItemSortMode {
  const s = (raw || '').trim().toLowerCase()
  if (s === 'updated_at') return 'updated_at'
  if (s === 'sort' || s === 'custom') return 'sort'
  return 'uploaded_at'
}

function parseMs(iso: string | undefined): number | null {
  if (!iso?.trim()) return null
  const t = Date.parse(iso)
  return Number.isNaN(t) ? null : t
}

/** 与后端 ordering 一致：新时间靠前；无时间排后。 */
export function sortGalleryRows(
  rows: readonly GalleryRemoteRow[],
  mode: GalleryItemSortMode,
): GalleryRemoteRow[] {
  const list = [...rows]
  if (mode === 'sort') {
    list.sort((a, b) => {
      const d = rankSortField(a.sort) - rankSortField(b.sort)
      if (d !== 0) return d
      return a.url.localeCompare(b.url)
    })
    return list
  }
  if (mode === 'updated_at') {
    list.sort((a, b) => {
      const ua = parseMs(a.updatedAt)
      const ub = parseMs(b.updatedAt)
      const c = compareDescMs(ua, ub)
      if (c !== 0) return c
      const ia = parseMs(a.uploadedAt)
      const ib = parseMs(b.uploadedAt)
      return compareDescMs(ia, ib) || a.url.localeCompare(b.url)
    })
    return list
  }
  // uploaded_at
  list.sort((a, b) => {
    const ta = parseMs(a.uploadedAt)
    const tb = parseMs(b.uploadedAt)
    const c = compareDescMs(ta, tb)
    if (c !== 0) return c
    return a.url.localeCompare(b.url)
  })
  return list
}

function compareDescMs(a: number | null, b: number | null): number {
  if (a == null && b == null) return 0
  if (a == null) return 1
  if (b == null) return -1
  return b - a
}
