<template>
  <div
    ref="rootRef"
    class="gallery-grid-wrap"
    :style="{ gridTemplateColumns: `repeat(${gridColumnCount}, minmax(0, 1fr))` }"
  >
    <Draggable
      v-model="itemsModel"
      item-key="id"
      tag="div"
      class="gallery-grid"
      :disabled="!sortEnabled"
      handle=".gallery-item-drag-handle"
      :animation="180"
      ghost-class="gallery-card-drag-ghost"
      chosen-class="gallery-card-drag-chosen"
      drag-class="gallery-card-drag-active"
      @start="$emit('drag-start')"
      @end="$emit('drag-end')"
    >
    <template #item="{ element, index }">
      <div class="gallery-grid__cell">
        <GalleryCardItem
          :item="element"
          :draggable="sortEnabled"
      :show-admin-actions="showAdminActions"
      :gallery-folder-key="galleryFolderKey"
      fixed-landscape43
      :item-style="getItemStyle?.(element, index)"
          @click="$emit('card-click', $event, index)"
          @delete="$emit('delete', $event)"
          @transfer="$emit('transfer', $event)"
          @edit="$emit('edit', $event)"
          @view="$emit('view', $event)"
          @download="$emit('download', $event)"
          @copy-link="$emit('copy-link', $event)"
          @aspect-hw="(id, hw) => $emit('aspect-hw', id, hw)"
        />
      </div>
    </template>
  </Draggable>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import Draggable from 'vuedraggable'
import GalleryCardItem from '@/components/gallery/GalleryCardItem.vue'
import type { GalleryDisplayItem, GalleryItemStyleFn } from '@/components/gallery/types'
import { useMasonryLayoutStore } from '@/stores/masonryLayout'
import {
  resolveGridColumnCount,
  estimateGridRowsPerScreen,
  readGalleryGapPx,
  measureAvailableGridHeight,
} from '@/utils/gridPagination'

const props = defineProps<{
  items: GalleryDisplayItem[]
  draggable?: boolean
  showAdminActions?: boolean
  galleryFolderKey?: string
  getItemStyle?: GalleryItemStyleFn
}>()

const sortEnabled = computed(() => Boolean(props.draggable))

const emit = defineEmits<{
  (e: 'update:items', items: GalleryDisplayItem[]): void
  (e: 'layout'): void
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

const itemsModel = computed({
  get: () => props.items,
  set: (items) => emit('update:items', items),
})

const masonryLayoutStore = useMasonryLayoutStore()
const { columnChoice } = storeToRefs(masonryLayoutStore)
const gridColumnCount = ref(2)
const gridRowsPerScreen = ref(1)
const rootRef = ref<HTMLElement | null>(null)

function syncGridMetrics() {
  const el = rootRef.value
  if (!el) return
  const w = el.getBoundingClientRect().width
  const nextCols = resolveGridColumnCount(w, columnChoice.value)
  if (nextCols !== gridColumnCount.value) gridColumnCount.value = nextCols
  const gapPx = readGalleryGapPx(el)
  const availableHeight = measureAvailableGridHeight(el)
  const nextRows = estimateGridRowsPerScreen({
    containerWidth: w,
    columns: nextCols,
    gapPx,
    availableHeight,
  })
  if (nextRows !== gridRowsPerScreen.value) gridRowsPerScreen.value = nextRows
}

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  syncGridMetrics()
  const el = rootRef.value
  if (!el || typeof ResizeObserver === 'undefined') return
  resizeObserver = new ResizeObserver(() => syncGridMetrics())
  resizeObserver.observe(el)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})

watch(columnChoice, () => syncGridMetrics())

watch(
  () => [props.items.length, gridColumnCount.value, gridRowsPerScreen.value] as const,
  async () => {
    if (props.items.length === 0) return
    await nextTick()
    requestAnimationFrame(() => emit('layout'))
  },
  { immediate: true },
)

defineExpose({
  rootRef,
  syncGridMetrics,
  get gridColumnCount() {
    return gridColumnCount.value
  },
  get gridRowsPerScreen() {
    return gridRowsPerScreen.value
  },
})
</script>

<style scoped lang="scss">
.gallery-grid-wrap {
  display: grid;
  gap: var(--index-gallery-gap);
  width: 100%;
  margin-bottom: clamp(16px, 4vw, 30px);
  align-items: start;
}

/* Sortable 会破坏直接设 grid 的容器；用 display:contents 让卡片成为外层 grid 的子项 */
.gallery-grid {
  display: contents;
}

:deep(.gallery-card-item) {
  min-width: 0;
}

.gallery-grid__cell {
  width: 100%;
  min-width: 0;
  position: relative;

  &:has(.card.is-tilting) {
    z-index: 2;
  }
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
