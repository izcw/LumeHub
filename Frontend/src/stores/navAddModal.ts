import { defineStore } from 'pinia'
import { ref } from 'vue'

export type NavAddModalMode = 'primary' | 'gallery'

export const useNavAddModalStore = defineStore('navAddModal', () => {
  const open = ref(false)
  const mode = ref<NavAddModalMode>('primary')
  /** 画廊「+」时当前大分类名称（展示） */
  const galleryMajorName = ref('')
  /** 画廊「+」时当前大分类 id（提交子分类用） */
  const galleryMajorId = ref<number | null>(null)

  function openPrimary() {
    mode.value = 'primary'
    galleryMajorName.value = ''
    galleryMajorId.value = null
    open.value = true
  }

  function openGallery(majorId: number | null | undefined, majorName?: string) {
    mode.value = 'gallery'
    galleryMajorName.value = majorName?.trim() ?? ''
    galleryMajorId.value = typeof majorId === 'number' && Number.isFinite(majorId) ? majorId : null
    open.value = true
  }

  function close() {
    open.value = false
  }

  return {
    open,
    mode,
    galleryMajorName,
    galleryMajorId,
    openPrimary,
    openGallery,
    close,
  }
})
