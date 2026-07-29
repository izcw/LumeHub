import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  normalizeGalleryItemSortMode,
  type GalleryItemSortMode,
} from '@/utils/galleryItemOrder'

// v2 drops legacy overrides created when drag sorting was treated as the
// default mode. New categories now default to uploaded_at.
const LS_KEY = 'lumehub-gallery-item-sort-overrides-v2'

type OverridesMap = Record<string, GalleryItemSortMode>

function readOverrides(): OverridesMap {
  try {
    const raw = localStorage.getItem(LS_KEY)
    if (!raw) return {}
    const o = JSON.parse(raw) as unknown
    if (!o || typeof o !== 'object') return {}
    const out: OverridesMap = {}
    for (const [k, v] of Object.entries(o as Record<string, string>)) {
      if (typeof k !== 'string' || !k) continue
      out[k] = normalizeGalleryItemSortMode(v)
    }
    return out
  } catch {
    return {}
  }
}

function writeOverrides(map: OverridesMap) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(map))
  } catch {
    /* ignore */
  }
}

/** 各分类资源列表排序：用户覆盖写本地；否则用接口下发的默认（再否则 uploaded_at） */
export const useGalleryItemSortStore = defineStore('galleryItemSort', () => {
  const overrides = ref<OverridesMap>(readOverrides())
  const serverDefaults = ref<OverridesMap>({})

  function setServerDefault(folderKey: string, raw: string | undefined) {
    if (!folderKey) return
    serverDefaults.value = {
      ...serverDefaults.value,
      [folderKey]: normalizeGalleryItemSortMode(raw),
    }
  }

  function effectiveMode(folderKey: string): GalleryItemSortMode {
    const o = overrides.value[folderKey]
    if (o) return o
    const s = serverDefaults.value[folderKey]
    if (s) return s
    return 'uploaded_at'
  }

  function setOverride(folderKey: string, mode: GalleryItemSortMode) {
    if (!folderKey) return
    overrides.value = {
      ...overrides.value,
      [folderKey]: normalizeGalleryItemSortMode(mode),
    }
    writeOverrides(overrides.value)
  }

  function clearOverride(folderKey: string) {
    if (!folderKey) return
    const next = { ...overrides.value }
    delete next[folderKey]
    overrides.value = next
    writeOverrides(next)
  }

  return { overrides, serverDefaults, setServerDefault, effectiveMode, setOverride, clearOverride }
})
