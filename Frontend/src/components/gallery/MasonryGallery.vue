<template>
  <div ref="rootRef" class="masonry-gallery">
    <draggable
      v-for="(column, ci) in dragColumns"
      :key="ci"
      :list="column"
      item-key="id"
      tag="div"
      class="masonry-gallery__col"
      group="gallery-masonry-cells"
      :disabled="!draggableEnabled"
      handle=".gallery-item-drag-handle"
      :animation="140"
      :force-fallback="false"
      :fallback-on-body="false"
      :fallback-tolerance="2"
      :swap-threshold="0.65"
      :invert-swap="false"
      :empty-insert-threshold="64"
      ghost-class="gallery-card-drag-ghost"
      chosen-class="gallery-card-drag-chosen"
      drag-class="gallery-card-drag-active"
      @start="onDragStart"
      @end="onDragEnd"
    >
      <template #item="{ element }">
        <div class="masonry-gallery__cell">
          <slot name="item" :item="element.item" :index="element.index" />
        </div>
      </template>
    </draggable>
  </div>
</template>

<script setup lang="ts" generic="T extends { id: number | string }">
import { nextTick, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import { storeToRefs } from 'pinia'
import draggable from 'vuedraggable'
import { useMasonryLayoutStore } from '@/stores/masonryLayout'

const props = defineProps<{
  /** 每项可带 masonryAspectHW（高/宽），用于分列占位；缺省为 4/3，与图片未解码前竖向占位一致 */
  items: readonly (T & { masonryAspectHW?: number })[]
  draggableEnabled?: boolean
  /** 自定义排序模式下启用：按顺序保留分列，不再做最矮列自动补齐 */
  preserveOrderColumns?: boolean
}>()

const masonryStore = useMasonryLayoutStore()
const { columnChoice } = storeToRefs(masonryStore)
const draggableEnabled = ref(Boolean(props.draggableEnabled))

defineSlots<{
  item(props: { item: T; index: number }): unknown
}>()

const emit = defineEmits<{
  layout: []
  reorder: [payload: { orderedIds: string[]; masonryPlacement: Record<string, { col: number; row: number }> }]
  dragStart: []
  dragEnd: []
}>()

const rootRef = ref<HTMLElement | null>(null)
const columnCount = ref(2)
const measuredWidth = ref(0)
const draggingNow = ref(false)
const pendingSyncAfterDrag = ref(false)

/** 与 PictureCard 未解码前的 3:4 占位一致，用于「下一张塞进当前最矮列」 */
const PLACEHOLDER_ASPECT_HW = 4 / 3

const breakpoints: readonly { minWidth: number; cols: number }[] = [
  { minWidth: 1200, cols: 4 },
  { minWidth: 768, cols: 3 },
  { minWidth: 481, cols: 2 },
  { minWidth: 0, cols: 2 },
]

function colsForWidth(w: number): number {
  for (const bp of breakpoints) {
    if (w >= bp.minWidth) return bp.cols
  }
  return 2
}

function applyColumnChoice(width: number) {
  const choice = columnChoice.value
  if (choice === 'auto') {
    const next = colsForWidth(width)
    if (next !== columnCount.value) columnCount.value = next
  } else {
    const clamped = Math.min(6, Math.max(1, choice))
    if (clamped !== columnCount.value) columnCount.value = clamped
  }
}

interface Cell {
  id: string
  item: T
  index: number
}

const dragColumns = shallowRef<Cell[][]>([])

function buildColumnsFromItems(items: readonly (T & { masonryAspectHW?: number })[]): Cell[][] {
  const list = items
  const k = Math.max(1, columnCount.value)
  if (list.length === 0) return []
  const cols: Cell[][] = Array.from({ length: k }, () => [])
  if (props.preserveOrderColumns) {
    const positioned: Array<{ item: T; index: number; col: number; row: number }> = []
    const fallback: Array<{ item: T; index: number }> = []
    let maxSavedCol = -1
    for (let index = 0; index < list.length; index++) {
      const item = list[index]!
      const rawCol = (item as { masonryCol?: number }).masonryCol
      const rawRow = (item as { masonryRow?: number }).masonryRow
      if (
        typeof rawCol === 'number' &&
        Number.isFinite(rawCol) &&
        rawCol >= 0 &&
        typeof rawRow === 'number' &&
        Number.isFinite(rawRow) &&
        rawRow >= 0
      ) {
        const col = Math.floor(rawCol)
        const row = Math.floor(rawRow)
        if (col > maxSavedCol) maxSavedCol = col
        positioned.push({ item, index, col, row })
      } else {
        fallback.push({ item, index })
      }
    }
    const savedColCount = maxSavedCol + 1
    if (positioned.length > 0 && savedColCount === k) {
      positioned.sort((a, b) => (a.col - b.col) || (a.row - b.row) || (a.index - b.index))
      for (const p of positioned) {
        if (p.col < 0 || p.col >= k) continue
        cols[p.col]!.push({ id: String(p.item.id), item: p.item, index: p.index })
      }
      let fillCol = 0
      for (const p of fallback) {
        cols[fillCol]!.push({ id: String(p.item.id), item: p.item, index: p.index })
        fillCol = (fillCol + 1) % k
      }
      return cols
    }
    for (let index = 0; index < list.length; index++) {
      const item = list[index]!
      const colIndex = index % k
      cols[colIndex]!.push({ id: String(item.id), item, index })
    }
    return cols
  }
  const heights = Array(k).fill(0)
  list.forEach((item, index) => {
    let best = 0
    let min = heights[0]!
    for (let i = 1; i < k; i++) {
      if (heights[i]! < min) {
        min = heights[i]!
        best = i
      }
    }
    cols[best]!.push({ id: String(item.id), item, index })
    const hw =
      typeof (item as { masonryAspectHW?: number }).masonryAspectHW === 'number'
        ? (item as { masonryAspectHW: number }).masonryAspectHW
        : PLACEHOLDER_ASPECT_HW
    heights[best] += hw
  })
  return cols
}

function rebuildColumns() {
  dragColumns.value = buildColumnsFromItems(props.items)
}

function flattenCellsByRows(cols: readonly Cell[][]): Cell[] {
  let maxLen = 0
  for (const col of cols) {
    if (col.length > maxLen) maxLen = col.length
  }
  const out: Cell[] = []
  for (let row = 0; row < maxLen; row++) {
    for (let ci = 0; ci < cols.length; ci++) {
      const cell = cols[ci]?.[row]
      if (cell) out.push(cell)
    }
  }
  return out
}

function itemIds(items: readonly (T & { masonryAspectHW?: number })[]): string[] {
  return items.map((item) => String(item.id))
}

function sameIdOrder(a: readonly string[], b: readonly string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

function syncCellsFromItemsIfSameOrder(): boolean {
  const nextIds = itemIds(props.items)
  const currentIds = flattenCellsByRows(dragColumns.value).map((cell) => cell.id)
  if (!sameIdOrder(nextIds, currentIds)) return false
  const byId = new Map<string, { item: T; index: number }>()
  props.items.forEach((item, index) => {
    byId.set(String(item.id), { item, index })
  })
  dragColumns.value = dragColumns.value.map((col) =>
    col
      .map((cell) => {
        const hit = byId.get(cell.id)
        if (!hit) return null
        return { ...cell, item: hit.item, index: hit.index }
      })
      .filter((cell): cell is Cell => cell !== null),
  )
  return true
}

function normalizeCellIndexes() {
  const flat = flattenCellsByRows(dragColumns.value)
  for (let i = 0; i < flat.length; i++) {
    flat[i]!.index = i
  }
}

function onDragStart() {
  draggingNow.value = true
  emit('dragStart')
}

async function onDragEnd() {
  draggingNow.value = false
  emit('dragEnd')
  await nextTick()
  normalizeCellIndexes()
  const orderedIds = flattenCellsByRows(dragColumns.value).map((cell) => cell.id)
  const masonryPlacement: Record<string, { col: number; row: number }> = {}
  for (let col = 0; col < dragColumns.value.length; col++) {
    const cells = dragColumns.value[col]!
    for (let row = 0; row < cells.length; row++) {
      const id = cells[row]!.id
      masonryPlacement[id] = { col, row }
    }
  }
  emit('reorder', { orderedIds, masonryPlacement })
  if (pendingSyncAfterDrag.value) {
    pendingSyncAfterDrag.value = false
    rebuildColumns()
    await nextTick()
    requestAnimationFrame(() => notifyLayout())
  }
}

let resizeObserver: ResizeObserver | null = null

function notifyLayout() {
  emit('layout')
}

watch(
  () => props.items,
  async () => {
    if (draggingNow.value) {
      pendingSyncAfterDrag.value = true
      return
    }
    if (!syncCellsFromItemsIfSameOrder()) {
      rebuildColumns()
    }
    await nextTick()
    requestAnimationFrame(() => notifyLayout())
  },
  { immediate: true },
)

watch(columnCount, async () => {
  rebuildColumns()
  await nextTick()
  requestAnimationFrame(() => notifyLayout())
})

watch(columnChoice, () => {
  applyColumnChoice(measuredWidth.value || rootRef.value?.getBoundingClientRect().width || 0)
})

watch(
  () => props.draggableEnabled,
  (enabled) => {
    draggableEnabled.value = Boolean(enabled)
  },
  { immediate: true },
)

onMounted(() => {
  const el = rootRef.value
  if (!el) return
  const w = el.getBoundingClientRect().width
  measuredWidth.value = w
  applyColumnChoice(w)
  resizeObserver = new ResizeObserver((entries) => {
    const rw = entries[0]?.contentRect.width ?? 0
    if (rw > 0) {
      measuredWidth.value = rw
      applyColumnChoice(rw)
    }
  })
  resizeObserver.observe(el)
  void nextTick().then(() => requestAnimationFrame(() => notifyLayout()))
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})
</script>

<style scoped lang="scss">
.masonry-gallery {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  width: 100%;
  margin-bottom: 30px;
}

.masonry-gallery__col {
  flex: 1;
  min-width: 0;
  min-height: 220px;
  padding-bottom: 72px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  box-sizing: border-box;
}

.masonry-gallery__cell {
  width: 100%;
  position: relative;

  &:has(.card.is-tilting) {
    z-index: 2;
  }
}
</style>
