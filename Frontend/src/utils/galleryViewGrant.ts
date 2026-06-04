const STORAGE_KEY = 'lumehub_gallery_vg'
const LEGACY_STORAGE_KEY = 'lumehub_gallery_vp'

type ViewGrantEntry = {
  grant: string
  expiresAt: number
}

type ViewGrantStore = Record<string, ViewGrantEntry>

function purgeLegacyStorage() {
  if (typeof sessionStorage === 'undefined') return
  try {
    sessionStorage.removeItem(LEGACY_STORAGE_KEY)
  } catch {
    /* ignore */
  }
}

function readStore(): ViewGrantStore {
  if (typeof sessionStorage === 'undefined') return {}
  purgeLegacyStorage()
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as ViewGrantStore
    if (!parsed || typeof parsed !== 'object') return {}
    const now = Date.now()
    let changed = false
    for (const [key, entry] of Object.entries(parsed)) {
      if (!entry?.grant || now >= entry.expiresAt) {
        delete parsed[key]
        changed = true
      }
    }
    if (changed) writeStore(parsed)
    return parsed
  } catch {
    return {}
  }
}

function writeStore(store: ViewGrantStore) {
  if (typeof sessionStorage === 'undefined') return
  try {
    if (Object.keys(store).length === 0) {
      sessionStorage.removeItem(STORAGE_KEY)
      return
    }
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(store))
  } catch {
    /* 存储配额等异常时忽略 */
  }
}

/** 读取当前标签页内有效的相册查看令牌（关闭网页后 sessionStorage 自动清空）。 */
export function getGalleryViewGrant(folderKey: string): string {
  const key = folderKey.trim()
  if (!key) return ''
  const entry = readStore()[key]
  return entry?.grant?.trim() ?? ''
}

/** 保存服务端签发的查看令牌（不含明文密码）。 */
export function setGalleryViewGrant(folderKey: string, grant: string, expiresAtMs: number) {
  const key = folderKey.trim()
  const token = grant.trim()
  if (!key || !token || !Number.isFinite(expiresAtMs) || expiresAtMs <= Date.now()) return
  const store = readStore()
  store[key] = { grant: token, expiresAt: expiresAtMs }
  writeStore(store)
}

export function clearGalleryViewGrant(folderKey: string) {
  const key = folderKey.trim()
  if (!key) return
  const store = readStore()
  if (!store[key]) return
  delete store[key]
  writeStore(store)
}

purgeLegacyStorage()
