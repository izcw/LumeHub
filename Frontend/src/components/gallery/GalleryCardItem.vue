<template>
  <div
    class="gallery-card-item"
    :class="{ 'is-new': item.isNew, 'is-draggable': draggable }"
    :style="itemStyle"
    @click="$emit('click', $event)"
  >
    <button
      v-if="draggable"
      type="button"
      class="gallery-item-drag-handle"
      aria-label="拖拽排序手柄"
      @click.stop
    >
      <img src="@/assets/icon/drag.svg" alt="" width="14" height="14" aria-hidden="true" />
    </button>
    <PictureCard
      :key="item.id"
      :src="item.src"
      :viewer-full-src="item.fullSrc"
      :live-video-url="item.liveVideoUrl"
      :is-live-photo="item.isLivePhoto"
      :date="item.cardDate"
      :file-size="item.fileSize"
      :format="item.format"
      :media-kind="item.mediaKind"
      :fixed-landscape43="fixedLandscape43"
      :show-admin-actions="showAdminActions"
      :show-transfer="showTransfer"
      delete-confirm-title="确定移入回收站吗？"
      :view-href="item.fullSrc"
      @delete="$emit('delete', item)"
      @transfer="$emit('transfer', item)"
      @edit="$emit('edit', item)"
      @view="$emit('view', item)"
      @download="$emit('download', item)"
      @copy-link="$emit('copy-link', item)"
      @aspect-hw="(hw) => $emit('aspect-hw', item.id, hw)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import PictureCard from '@/components/gallery/PictureCard.vue'
import type { GalleryDisplayItem } from '@/components/gallery/types'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { resolveGalleryItemContext } from '@/views/gallery/utils'

const props = defineProps<{
  item: GalleryDisplayItem
  draggable?: boolean
  showAdminActions?: boolean
  galleryFolderKey?: string
  fixedLandscape43?: boolean
  itemStyle?: Readonly<Record<string, string>>
}>()

const categoryNavStore = useCategoryNavStore()

const showTransfer = computed(() => {
  if (!props.showAdminActions) return false
  const defaultFk = props.galleryFolderKey?.trim() ?? ''
  const { folderKey: fk } = resolveGalleryItemContext(props.item.id, defaultFk)
  const major = categoryNavStore.galleryMajorForFolderKey(fk)
  if (!major?.subcategories?.length) return false
  return major.subcategories.some((s) => s.folderKey.trim() && s.folderKey !== fk)
})

defineEmits<{
  (e: 'click', event: MouseEvent): void
  (e: 'delete', item: GalleryDisplayItem): void
  (e: 'transfer', item: GalleryDisplayItem): void
  (e: 'edit', item: GalleryDisplayItem): void
  (e: 'view', item: GalleryDisplayItem): void
  (e: 'download', item: GalleryDisplayItem): void
  (e: 'copy-link', item: GalleryDisplayItem): void
  (e: 'aspect-hw', itemId: string, hw: number): void
}>()
</script>

<style scoped lang="scss">
.gallery-card-item {
  position: relative;
  width: 100%;
  opacity: 1;
  transition: opacity 0.8s cubic-bezier(0.22, 1, 0.36, 1);

  &:has(.card.is-tilting) {
    z-index: 2;
  }

  &.is-draggable {
    cursor: grab;
  }

  &.is-new {
    opacity: 0;
    animation: gallery-card-fade-in 0.8s cubic-bezier(0.22, 1, 0.36, 1) forwards;
  }
}

.gallery-item-drag-handle {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 4;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.56);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  cursor: grab;

  img {
    width: 14px;
    height: 14px;
    display: block;
    object-fit: contain;
    filter: invert(1);
    opacity: 0.95;
  }
}

:deep(.gallery-card-drag-chosen) .gallery-item-drag-handle {
  cursor: grabbing;
}

:deep(.card) {
  height: auto !important;
  display: block !important;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  transition: box-shadow 0.3s ease;

  &:hover {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  }
}

:deep(.image-container) {
  height: auto !important;
}

:deep(.card .image),
:deep(.card .card-video),
:deep(.card .video-poster) {
  width: 100%;
  display: block;
  transition: transform 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

:deep(.card .file-type-panel) {
  transition: filter 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

:deep(.card:hover .image),
:deep(.card:hover .card-video),
:deep(.card:hover .video-poster) {
  transform: scale(1.03) !important;
  filter: brightness(1.03) contrast(1.02);
}

:deep(.date-tag) {
  bottom: 2px;
  right: 8px;
}

@keyframes gallery-card-fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
</style>
