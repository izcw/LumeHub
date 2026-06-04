<template>
  <div
    ref="waterfallScrollAnchor"
    class="waterfall-stage"
    :class="{ 'is-visible': waterfallStageVisible, 'is-dragging': draggingCards }"
  >
    <GalleryMasonryLayout
      v-if="layoutMode === 'masonry' && displayList.length > 0"
      key="gallery-masonry"
      :items="displayList"
      :draggable="canDragSort"
      :preserve-order-columns="effectiveItemSort === 'sort'"
      :show-admin-actions="showGalleryAdminActions"
      :gallery-folder-key="galleryFolderKey"
      :get-item-style="getItemStyle"
      @layout="$emit('layout')"
      @reorder="$emit('reorder', $event)"
      @drag-start="$emit('drag-start')"
      @drag-end="$emit('drag-end')"
      @card-click="(event, index) => $emit('card-click', event, index)"
      @delete="$emit('delete', $event)"
      @transfer="$emit('transfer', $event)"
      @edit="$emit('edit', $event)"
      @view="$emit('view', $event)"
      @download="$emit('download', $event)"
      @copy-link="$emit('copy-link', $event)"
      @aspect-hw="(id, hw) => $emit('aspect-hw', id, hw)"
    />
    <GalleryGridLayout
      v-else-if="layoutMode === 'grid' && displayList.length > 0"
      ref="gridLayoutRef"
      key="gallery-grid"
      v-model:items="displayListModel"
      :draggable="canDragSort"
      :show-admin-actions="showGalleryAdminActions"
      :gallery-folder-key="galleryFolderKey"
      :get-item-style="getItemStyle"
      @layout="$emit('layout')"
      @drag-start="$emit('drag-start')"
      @drag-end="$emit('drag-end')"
      @card-click="(event, index) => $emit('card-click', event, index)"
      @delete="$emit('delete', $event)"
      @transfer="$emit('transfer', $event)"
      @edit="$emit('edit', $event)"
      @view="$emit('view', $event)"
      @download="$emit('download', $event)"
      @copy-link="$emit('copy-link', $event)"
      @aspect-hw="(id, hw) => $emit('aspect-hw', id, hw)"
    />
  </div>

  <nav v-if="showPagination" class="pagination-bar" aria-label="图片分页">
    <span v-if="orderPersisting" class="order-persist-hint">排序保存中…</span>
    <span v-else-if="orderPersistHint" class="order-persist-hint">{{ orderPersistHint }}</span>
    <ToolbarSelect
      v-if="dragSortEditEnabled"
      ref="pageSizePickerRef"
      :open="pageSizeMenuOpen"
      :model-value="pageSizeSelectValue"
      :display-label="String(pageSize)"
      trigger-label="每页数量"
      menu-aria-label="每页数量选项"
      icon-text="页"
      :options="pageSizeSelectOptions"
      @toggle="$emit('toggle')"
      @select="$emit('select', $event)"
    />
    <Pagination
      class="gallery-pagination"
      :current-page="currentPage"
      :total-pages="totalPages"
      :total-count="orderedFilteredCount"
      count-unit="张"
      aria-label="图片分页"
      @prev="$emit('prev')"
      @next="$emit('next')"
    />
  </nav>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import GalleryMasonryLayout from '@/components/gallery/GalleryMasonryLayout.vue'
import GalleryGridLayout from '@/components/gallery/GalleryGridLayout.vue'
import type { GalleryDisplayItem, GalleryItemStyleFn } from '@/components/gallery/types'
import type { MasonryPlacementMap } from '@/api/gallery'
import Pagination from '@/components/shared-ui/Pagination.vue'
import ToolbarSelect from '@/components/shared-ui/ToolbarSelect.vue'
import { useMasonryLayoutStore } from '@/stores/masonryLayout'

const { galleryLayoutMode: layoutMode } = storeToRefs(useMasonryLayoutStore())

const props = defineProps<{
  waterfallStageVisible: boolean
  draggingCards: boolean
  displayList: GalleryDisplayItem[]
  canDragSort: boolean
  effectiveItemSort: string
  showGalleryAdminActions: boolean
  galleryFolderKey: string
  getItemStyle?: GalleryItemStyleFn
  showPagination: boolean
  orderPersisting: boolean
  orderPersistHint: string
  dragSortEditEnabled: boolean
  pageSizeMenuOpen: boolean
  pageSizeSelectValue: string
  pageSize: number
  pageSizeSelectOptions: Array<{ value: string; label: string }>
  currentPage: number
  totalPages: number
  orderedFilteredCount: number
}>()

const emit = defineEmits<{
  layout: []
  reorder: [payload: { orderedIds: string[]; masonryPlacement: MasonryPlacementMap }]
  'drag-start': []
  'drag-end': []
  'card-click': [event: MouseEvent, index: number]
  delete: [item: GalleryDisplayItem]
  transfer: [item: GalleryDisplayItem]
  edit: [item: GalleryDisplayItem]
  view: [item: GalleryDisplayItem]
  download: [item: GalleryDisplayItem]
  'copy-link': [item: GalleryDisplayItem]
  'aspect-hw': [itemId: string, hw: number]
  'update:displayList': [items: GalleryDisplayItem[]]
  toggle: []
  select: [size: string]
  prev: []
  next: []
}>()

const displayListModel = computed({
  get: () => props.displayList,
  set: (items) => emit('update:displayList', items),
})

const waterfallScrollAnchor = ref<HTMLElement | null>(null)
const pageSizePickerRef = ref<{ rootEl: HTMLElement | null; panelEl: HTMLElement | null } | null>(
  null,
)
const gridLayoutRef = ref<InstanceType<typeof GalleryGridLayout> | null>(null)

defineExpose({
  get waterfallScrollAnchor() {
    return waterfallScrollAnchor.value
  },
  get pageSizePickerRef() {
    return pageSizePickerRef.value
  },
  get gridLayoutRef() {
    return gridLayoutRef.value
  },
})
</script>

<style scoped lang="scss">
.waterfall-stage {
  opacity: 0;
  transition: opacity 0.8s cubic-bezier(0.22, 1, 0.36, 1);
  padding: var(--index-gallery-gap);
  box-sizing: border-box;

  &.is-visible {
    opacity: 1;
  }

  &.is-dragging {
    :deep(.gallery-card-item),
    :deep(.card),
    :deep(.card .image),
    :deep(.card .card-video),
    :deep(.card .video-poster) {
      transition: none !important;
    }

    :deep(.card:hover .image),
    :deep(.card:hover .card-video),
    :deep(.card:hover .video-poster) {
      transform: none !important;
      filter: none !important;
    }

    :deep(.gallery-item-drag-handle) {
      cursor: grabbing;
    }
  }
}

.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: var(--index-gallery-gap);
  padding: clamp(12px, 3vw, 16px) var(--index-gallery-gap) clamp(24px, 6vw, 38px);
  color: #2f2f2f;
  font-size: 13px;
}

.order-persist-hint {
  font-size: 12px;
  color: #6f7685;
}

@media (max-width: 640px) {
  .pagination-bar {
    flex-direction: column;
    gap: 10px;
    padding-bottom: clamp(20px, 5vw, 30px);
  }

  .gallery-pagination {
    :deep(.ui-pagination__info) {
      flex: 0 1 auto;
      min-width: auto;
      padding: 0 4px;
    }
  }
}
</style>
