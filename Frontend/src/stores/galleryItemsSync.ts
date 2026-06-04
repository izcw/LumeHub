import { defineStore } from 'pinia'
import { ref } from 'vue'

/** 画廊列表与回收站等外部操作的同步信号 */
export const useGalleryItemsSyncStore = defineStore('galleryItemsSync', () => {
  const reloadNonce = ref(0)
  const dirtyFolderKeys = ref<ReadonlySet<string>>(new Set())
  const globalPoolDirty = ref(false)

  function markCategoryItemsChanged(folderKey: string) {
    const fk = folderKey.trim()
    if (!fk) return
    const next = new Set(dirtyFolderKeys.value)
    next.add(fk)
    dirtyFolderKeys.value = next
    globalPoolDirty.value = true
    reloadNonce.value++
  }

  function takeFolderDirty(folderKey: string): boolean {
    const fk = folderKey.trim()
    if (!dirtyFolderKeys.value.has(fk)) return false
    const next = new Set(dirtyFolderKeys.value)
    next.delete(fk)
    dirtyFolderKeys.value = next
    return true
  }

  function takeGlobalPoolDirty(): boolean {
    if (!globalPoolDirty.value) return false
    globalPoolDirty.value = false
    return true
  }

  return {
    reloadNonce,
    markCategoryItemsChanged,
    takeFolderDirty,
    takeGlobalPoolDirty,
  }
})
