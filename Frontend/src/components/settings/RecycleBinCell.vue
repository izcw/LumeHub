<template>
  <div ref="target" class="recycle-bin__cell">
    <PictureCard
      v-if="inView"
      :src="item.src"
      :viewer-full-src="item.fullSrc"
      :live-video-url="item.liveVideoUrl"
      :is-live-photo="item.isLivePhoto"
      :date="item.cardDate"
      :file-size="item.fileSize"
      :format="item.format"
      :media-kind="item.mediaKind"
      lite
      fixed-landscape43
      show-admin-actions
      delete-only
      show-restore
      delete-confirm-title="确定永久删除该文件吗？此操作不可恢复。"
      :view-href="item.fullSrc"
      @delete="emit('delete')"
      @restore="emit('restore')"
    />
    <div v-else class="recycle-bin__cell-placeholder" aria-hidden="true" />
  </div>
</template>

<script setup lang="ts">
import { inject, ref, type Ref } from 'vue'
import PictureCard from '@/components/gallery/PictureCard.vue'
import { useInView } from '@/composables/useInView'

export type RecycleBinCellItem = {
  src: string
  fullSrc: string
  liveVideoUrl?: string
  isLivePhoto?: boolean
  cardDate: string
  fileSize?: number
  format?: string
  mediaKind?: string
}

defineProps<{
  item: RecycleBinCellItem
}>()

const emit = defineEmits<{
  delete: []
  restore: []
}>()

const scrollRoot = inject<Ref<Element | null | undefined>>(
  'recycleBinScrollRoot',
  ref(null),
)

const { target, inView } = useInView({ root: scrollRoot })
</script>

<style scoped lang="scss">
.recycle-bin__cell-placeholder {
  width: 100%;
  aspect-ratio: 4 / 3;
  border-radius: 8px;
  background: #f0f0f2;
}
</style>
