/** 无二级画廊时，用大分类 id 生成占位 folderKey，用于路由与导航（不对应磁盘目录） */
const MAJOR_PLACEHOLDER_PREFIX = '@major-'

export function majorPlaceholderFolderKey(majorId: number): string {
  return `${MAJOR_PLACEHOLDER_PREFIX}${majorId}`
}

export function parseMajorPlaceholderFolderKey(folderKey: string): number | null {
  const fk = folderKey.trim()
  if (!fk.startsWith(MAJOR_PLACEHOLDER_PREFIX)) return null
  const id = Number.parseInt(fk.slice(MAJOR_PLACEHOLDER_PREFIX.length), 10)
  return Number.isFinite(id) ? id : null
}

export function isMajorPlaceholderFolderKey(folderKey: string): boolean {
  return parseMajorPlaceholderFolderKey(folderKey) != null
}
