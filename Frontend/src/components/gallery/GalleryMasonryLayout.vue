<template>
  <MasonryGallery
    :items="items"
    :draggable-enabled="draggable"
    :preserve-order-columns="preserveOrderColumns"
    @layout="$emit('layout')"
    @reorder="$emit('reorder', $event)"
    @drag-start="$emit('drag-start')"
    @drag-end="$emit('drag-end')"
  >
    <template #item="{ item, index }">
      <GalleryCardItem
        :item="item"
        :draggable="draggable"
        :show-admin-actions="showAdminActions"
        :gallery-folder-key="galleryFolderKey"
        :item-style="getItemStyle?.(item, index)"
        @click="$emit('card-click', $event, index)"
        @delete="$emit('delete', $event)"
        @transfer="$emit('transfer', $event)"
        @edit="$emit('edit', $event)"
        @view="$emit('view', $event)"
        @download="$emit('download', $event)"
        @copy-link="$emit('copy-link', $event)"
        @aspect-hw="(id, hw) => $emit('aspect-hw', id, hw)"
      />
    </template>
  </MasonryGallery>
</template>

<script setup lang="ts">
import MasonryGallery from '@/components/gallery/MasonryGallery.vue'
import GalleryCardItem from '@/components/gallery/GalleryCardItem.vue'
import type { GalleryDisplayItem, GalleryItemStyleFn } from '@/components/gallery/types'
import type { MasonryPlacementMap } from '@/api/gallery'

defineProps<{
  items: GalleryDisplayItem[]
  draggable?: boolean
  preserveOrderColumns?: boolean
  showAdminActions?: boolean
  galleryFolderKey?: string
  getItemStyle?: GalleryItemStyleFn
}>()

defineEmits<{
  (e: 'layout'): void
  (e: 'reorder', payload: { orderedIds: string[]; masonryPlacement: MasonryPlacementMap }): void
  (e: 'drag-start'): void
  (e: 'drag-end'): void
  (e: 'card-click', event: MouseEvent, index: number): void
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
:deep(.masonry-gallery) {
  gap: var(--index-gallery-gap);
  margin-bottom: clamp(16px, 4vw, 30px);
}

:deep(.masonry-gallery__col) {
  gap: var(--index-gallery-gap);
}

:deep(.gallery-card-drag-ghost) {
  opacity: 0.45;
}

:deep(.gallery-card-drag-chosen) {
  cursor: grabbing;
}

:deep(.gallery-card-drag-active) {
  opacity: 0.92;
}
</style>
