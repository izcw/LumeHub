/** 将 `fromIndex` 元素移动到「插入到原列表中 `insertBefore` 下标之前」的位置（与 HTML5 拖放语义一致）。 */
export function reorderByInsertBefore<T>(items: readonly T[], fromIndex: number, insertBefore: number): T[] {
  const arr = [...items]
  const it = arr.splice(fromIndex, 1)[0]
  if (it === undefined) return arr
  let pos = insertBefore
  if (fromIndex < insertBefore) pos = insertBefore - 1
  if (pos < 0) pos = 0
  if (pos > arr.length) pos = arr.length
  arr.splice(pos, 0, it)
  return arr
}
