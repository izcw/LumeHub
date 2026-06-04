/** 系统保留的搜索结果画廊目录标识（非磁盘相册，仅虚拟二级导航） */
export const GALLERY_SEARCH_FOLDER_KEY = 'search'

export function normalizeGalleryFolderKey(folderKey: string): string {
  return folderKey.trim().toLowerCase()
}

export function isGallerySearchFolderKey(folderKey: string): boolean {
  return normalizeGalleryFolderKey(folderKey) === GALLERY_SEARCH_FOLDER_KEY
}

/** 用户创建/重命名画廊时不可使用的目录标识 */
export function isReservedGalleryFolderKey(folderKey: string): boolean {
  return isGallerySearchFolderKey(folderKey)
}

export function isOnSearchGalleryRoute(
  routeName: string | symbol | null | undefined,
  folderKey: string | string[] | undefined,
): boolean {
  if (routeName === 'category') {
    const raw = folderKey
    const fk = typeof raw === 'string' ? raw : Array.isArray(raw) ? (raw[0] ?? '') : ''
    return isGallerySearchFolderKey(fk)
  }
  return false
}
