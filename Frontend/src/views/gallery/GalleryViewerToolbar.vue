<template>
  <PictureCardToolbar
    v-if="item"
    :src="item.src"
    :viewer-full-src="item.fullSrc"
    :file-size="item.fileSize"
    :format="item.format"
    :show-admin-actions="showAdminActions"
    :show-transfer="showTransfer"
    delete-confirm-title="确定移入回收站吗？"
    :view-href="item.fullSrc"
    layout="inline"
    :surface="toolbarSurface"
    :floating-z-index="VIEWER_Z.float"
    @delete="emit('delete')"
    @transfer="emit('transfer')"
    @edit="emit('edit')"
    @download="emit('download')"
    @copy-link="emit('copy-link')"
    @pin-change="emit('pin-change', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import PictureCardToolbar from '@/components/gallery/PictureCardToolbar.vue'
import type { GalleryDisplayItem } from '@/components/gallery/types'
import { galleryMediaKindFromUrl } from '@/utils/galleryMedia'
import { VIEWER_Z } from '@/components/viewers/shared/viewerLayers'

const props = defineProps<{
  item: GalleryDisplayItem | null
  showAdminActions?: boolean
  showTransfer?: boolean
}>()

const emit = defineEmits<{
  delete: []
  transfer: []
  edit: []
  download: []
  'copy-link': []
  'pin-change': [pinned: boolean]
}>()

const toolbarSurface = computed<'dark' | 'light'>(() => {
  const item = props.item
  if (!item) return 'dark'
  const declared = item.mediaKind?.trim().toLowerCase()
  if (declared === 'image' || declared === 'video') return 'dark'
  if (declared) return 'light'
  const kind = galleryMediaKindFromUrl(item.fullSrc || item.src)
  return kind === 'image' || kind === 'video' ? 'dark' : 'light'
})
</script>
