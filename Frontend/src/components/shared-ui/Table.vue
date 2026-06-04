<template>
  <div class="ui-table" :class="{ 'has-border-top': borderTop, 'has-border-bottom': borderBottom }">
    <div class="ui-table__scroll" :style="scrollStyle">
      <table class="ui-table__inner" :style="tableStyle">
        <thead class="ui-table__head" :class="{ 'is-sticky': stickyHeader }">
          <tr>
            <th
              v-for="column in normalizedColumns"
              :key="column.key"
              class="ui-table__th"
              :class="headerClass(column)"
              :style="headerStyle(column)"
            >
              <slot :name="`header-${column.key}`" :column="column">
                {{ column.title }}
              </slot>
            </th>
          </tr>
        </thead>
        <draggable
          :model-value="pageRows"
          item-key="rowKey"
          tag="tbody"
          handle=".ui-table__drag"
          :animation="180"
          ghost-class="ui-table__row-ghost"
          chosen-class="ui-table__row-chosen"
          drag-class="ui-table__row-drag"
          :move="onDragMoveByDraggable"
          @start="onDragStartByDraggable"
          @end="onDragEndByDraggable"
        >
          <template #item="{ element: entry }">
            <tr
              class="ui-table__tr"
              :class="{
                'is-level-0': entry.level === 0,
                'is-level-1': entry.level === 1,
              }"
            >
              <td
                v-for="column in normalizedColumns"
                :key="`${entry.rowKey}-${column.key}`"
                class="ui-table__td"
                :class="cellClass(column)"
                :style="cellStyle(column)"
              >
                <template v-if="column.type === 'control'">
                  <div class="ui-table__control" :style="controlIndentStyle(entry.level)">
                    <button type="button" class="ui-table__drag" aria-label="拖动排序手柄" @click.stop>
                      <img :src="dragIconSrc" alt="" aria-hidden="true" />
                    </button>
                    <button
                      v-if="tree && entry.hasChildren"
                      type="button"
                      class="ui-table__tree-toggle"
                      :aria-label="entry.expanded ? '收起子节点' : '展开子节点'"
                      @click.stop="toggleExpand(entry.rowKey)"
                    >
                      <img
                        class="ui-table__tree-icon"
                        :class="{ expanded: entry.expanded }"
                        :src="triangleIconSrc"
                        alt=""
                        aria-hidden="true"
                      />
                    </button>
                    <span v-else-if="tree" class="ui-table__tree-spacer" />
                  </div>
                </template>
                <template v-else-if="column.type === 'index'">
                  <span class="ui-table__index">{{ entry.serial }}</span>
                </template>
                <template v-else>
                  <div class="ui-table__cell-content" :class="{ 'is-ellipsis': !!column.ellipsis }">
                    <slot :name="`cell-${column.key}`" :row="entry.row" :column="column" :value="cellValue(entry.row, column)">
                      {{ cellValue(entry.row, column) }}
                    </slot>
                  </div>
                </template>
              </td>
            </tr>
          </template>
        </draggable>
      </table>
    </div>

    <div v-if="paginated" class="ui-table__pagination">
      <Pagination
        :current-page="safePage"
        :total-pages="pageCount"
        :total-count="flattenedRows.length"
        :show-count="true"
        count-unit="项"
        compact
        aria-label="表格分页"
        @update:current-page="onPageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import draggable from 'vuedraggable'
import dragIconSrc from '@/assets/icon/drag.svg'
import triangleIconSrc from '@/assets/icon/triangle.svg'
import Pagination from '@/components/shared-ui/Pagination.vue'

export type TableFixed = 'left' | 'right'

export interface TableColumn {
  key: string
  title: string
  width?: number | string
  align?: 'left' | 'center' | 'right'
  fixed?: TableFixed
  ellipsis?: boolean
  type?: 'text' | 'index' | 'control'
}

type RowLike = Record<string, unknown> & {
  children?: RowLike[]
}

const props = withDefaults(
  defineProps<{
    columns: TableColumn[]
    rows: RowLike[]
    rowKey?: string
    tree?: boolean
    childrenKey?: string
    borderTop?: boolean
    borderBottom?: boolean
    stickyHeader?: boolean
    fluidHeight?: boolean
    maxHeight?: number
    paginated?: boolean
    page?: number
    pageSize?: number
    sortable?: boolean
  }>(),
  {
    rowKey: 'id',
    tree: false,
    childrenKey: 'children',
    borderTop: true,
    borderBottom: true,
    stickyHeader: true,
    fluidHeight: true,
    maxHeight: 400,
    paginated: false,
    page: 1,
    pageSize: 10,
    sortable: true,
  },
)

const emit = defineEmits<{
  'update:page': [value: number]
}>()

type FlatEntry = {
  row: RowLike
  rowKey: string
  serial: string
  level: number
  hasChildren: boolean
  expanded: boolean
}

const localRows = ref<RowLike[]>([])
const expandedKeys = ref<Set<string>>(new Set())
const dragSnapshot = ref<FlatEntry[]>([])

const normalizedColumns = computed<TableColumn[]>(() =>
  props.columns.map((column, index) => ({
    ...column,
    fixed: column.fixed ?? (index <= 1 ? 'left' : undefined),
  })),
)

const safePageSize = computed(() => Math.max(1, props.pageSize))
const flattenedRows = computed<FlatEntry[]>(() => flattenRows(localRows.value, 0))
const pageCount = computed(() =>
  props.paginated ? Math.max(1, Math.ceil(flattenedRows.value.length / safePageSize.value)) : 1,
)
const safePage = computed(() => (props.paginated ? Math.min(Math.max(1, props.page), pageCount.value) : 1))
const pageRows = computed<FlatEntry[]>(() => {
  if (!props.paginated) return flattenedRows.value
  const start = (safePage.value - 1) * safePageSize.value
  return flattenedRows.value.slice(start, start + safePageSize.value)
})

const tableStyle = computed(() => {
  const minWidth = normalizedColumns.value.reduce((sum, column) => sum + resolveColumnWidth(column), 0)
  return { minWidth: `${Math.max(minWidth, 620)}px` }
})

const scrollStyle = computed(() => {
  const maxHeight = props.fluidHeight ? Math.max(140, props.maxHeight) : props.maxHeight
  return { maxHeight: `${maxHeight}px` }
})

const fixedLeftOffsets = computed(() => {
  const map = new Map<string, number>()
  let offset = 0
  for (const column of normalizedColumns.value) {
    if (column.fixed === 'left') {
      map.set(column.key, offset)
      offset += resolveColumnWidth(column)
    }
  }
  return map
})

const fixedRightOffsets = computed(() => {
  const map = new Map<string, number>()
  let offset = 0
  for (let i = normalizedColumns.value.length - 1; i >= 0; i -= 1) {
    const column = normalizedColumns.value[i]
    if (!column) continue
    if (column.fixed === 'right') {
      map.set(column.key, offset)
      offset += resolveColumnWidth(column)
    }
  }
  return map
})

watch(
  () => props.rows,
  (rows) => {
    localRows.value = cloneRows(rows)
    const allExpandable = collectExpandableKeys(localRows.value, new Set())
    if (expandedKeys.value.size === 0) expandedKeys.value = allExpandable
    else expandedKeys.value = new Set([...expandedKeys.value].filter((key) => allExpandable.has(key)))
  },
  { immediate: true, deep: true },
)

function cloneRows(rows: RowLike[]): RowLike[] {
  return rows.map((row) => {
    const out = { ...row } as RowLike
    const children = row[props.childrenKey]
    if (Array.isArray(children)) out[props.childrenKey] = cloneRows(children as RowLike[])
    return out
  })
}

function flattenRows(rows: RowLike[], level: number, parentSerial = ''): FlatEntry[] {
  const out: FlatEntry[] = []
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i]
    if (!row) continue
    const serial = parentSerial ? `${parentSerial}.${i + 1}` : `${i + 1}`
    const rowKey = String(row[props.rowKey] ?? `${level}-${i}`)
    const children = asChildren(row)
    const hasChildren = props.tree && children.length > 0
    const expanded = hasChildren ? expandedKeys.value.has(rowKey) : false
    out.push({ row, rowKey, serial, level, hasChildren, expanded })
    if (hasChildren && expanded) out.push(...flattenRows(children, level + 1, serial))
  }
  return out
}

function collectExpandableKeys(rows: RowLike[], bucket: Set<string>): Set<string> {
  for (const row of rows) {
    const key = String(row[props.rowKey] ?? '')
    const children = asChildren(row)
    if (key && children.length > 0) {
      bucket.add(key)
      collectExpandableKeys(children, bucket)
    }
  }
  return bucket
}

function asChildren(row: RowLike): RowLike[] {
  const list = row[props.childrenKey]
  if (Array.isArray(list)) return list as RowLike[]
  return []
}

function toggleExpand(rowKey: string) {
  if (expandedKeys.value.has(rowKey)) expandedKeys.value.delete(rowKey)
  else expandedKeys.value.add(rowKey)
}

function onDragStartByDraggable() {
  dragSnapshot.value = pageRows.value.slice()
}

function onDragEndByDraggable(evt: { oldIndex?: number; newIndex?: number }) {
  if (!props.sortable) return
  const oldIndex = evt.oldIndex
  const newIndex = evt.newIndex
  if (oldIndex == null || newIndex == null || oldIndex === newIndex) return
  const source = dragSnapshot.value[oldIndex]
  const target = dragSnapshot.value[newIndex]
  if (!source || !target) return
  moveRowWithinSameSiblings(source.rowKey, target.rowKey, oldIndex < newIndex ? 'after' : 'before')
}

function onDragMoveByDraggable(evt: {
  draggedContext?: { element?: FlatEntry }
  relatedContext?: { element?: FlatEntry }
}) {
  if (!props.sortable) return false
  const dragged = evt.draggedContext?.element
  const related = evt.relatedContext?.element
  if (!dragged || !related) return false
  if (dragged.level !== related.level) return false
  return isSameSiblingList(dragged.rowKey, related.rowKey)
}

function isSameSiblingList(leftKey: string, rightKey: string): boolean {
  const left = findRowContext(localRows.value, leftKey)
  const right = findRowContext(localRows.value, rightKey)
  if (!left || !right) return false
  return left.list === right.list
}

function moveRowWithinSameSiblings(dragKey: string, targetKey: string, place: 'before' | 'after') {
  const dragCtx = findRowContext(localRows.value, dragKey)
  const targetCtx = findRowContext(localRows.value, targetKey)
  if (!dragCtx || !targetCtx || dragCtx.list !== targetCtx.list) return
  const list = dragCtx.list
  const [moved] = list.splice(dragCtx.index, 1)
  if (!moved) return
  const targetIndex = list.findIndex((item) => String(item[props.rowKey] ?? '') === targetKey)
  if (targetIndex < 0) {
    list.push(moved)
    return
  }
  const insertAt = place === 'after' ? targetIndex + 1 : targetIndex
  list.splice(insertAt, 0, moved)
}

function findRowContext(rows: RowLike[], key: string): { list: RowLike[]; index: number } | null {
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i]
    if (!row) continue
    const rowKey = String(row[props.rowKey] ?? '')
    if (rowKey === key) return { list: rows, index: i }
    const children = asChildren(row)
    if (children.length) {
      const childCtx = findRowContext(children, key)
      if (childCtx) return childCtx
    }
  }
  return null
}

function onPageChange(nextPage: number) {
  emit('update:page', nextPage)
}

function cellValue(row: RowLike, column: TableColumn): string {
  const value = row[column.key]
  if (value == null) return ''
  return String(value)
}

function resolveColumnWidth(column: TableColumn): number {
  if (typeof column.width === 'number') return column.width
  if (typeof column.width === 'string') {
    const n = Number.parseInt(column.width, 10)
    if (Number.isFinite(n)) return n
  }
  if (column.type === 'control') return 104
  if (column.type === 'index') return 56
  if (column.fixed) return 160
  return 170
}

function alignClass(column: TableColumn): string {
  if (column.align === 'center') return 'align-center'
  if (column.align === 'right') return 'align-right'
  return 'align-left'
}

function headerClass(column: TableColumn): string[] {
  const classes = [alignClass(column)]
  if (column.fixed === 'left') classes.push('is-fixed-left')
  if (column.fixed === 'right') classes.push('is-fixed-right')
  if (column.type === 'control' || column.type === 'index') classes.push('is-narrow')
  return classes
}

function cellClass(column: TableColumn): string[] {
  const classes = [alignClass(column)]
  if (column.fixed === 'left') classes.push('is-fixed-left')
  if (column.fixed === 'right') classes.push('is-fixed-right')
  if (column.type === 'control' || column.type === 'index') classes.push('is-narrow')
  return classes
}

function fixedStyle(column: TableColumn, isHeader: boolean): Record<string, string> {
  const z = isHeader ? '18' : '6'
  if (column.fixed === 'left') {
    return {
      position: 'sticky',
      left: `${fixedLeftOffsets.value.get(column.key) ?? 0}px`,
      zIndex: z,
    }
  }
  if (column.fixed === 'right') {
    return {
      position: 'sticky',
      right: `${fixedRightOffsets.value.get(column.key) ?? 0}px`,
      zIndex: z,
    }
  }
  return {}
}

function headerStyle(column: TableColumn): Record<string, string> {
  const width = resolveColumnWidth(column)
  return {
    width: `${width}px`,
    minWidth: `${width}px`,
    ...fixedStyle(column, true),
  }
}

function cellStyle(column: TableColumn): Record<string, string> {
  const width = resolveColumnWidth(column)
  return {
    width: `${width}px`,
    minWidth: `${width}px`,
    ...fixedStyle(column, false),
  }
}

function controlIndentStyle(level: number): Record<string, string> {
  if (!props.tree) return {}
  return { paddingLeft: `${Math.max(0, level) * 14}px` }
}
</script>

<style scoped lang="scss">
.ui-table {
  width: 100%;
  box-sizing: border-box;
  background: #fff;
  border-radius: 8px;
  border-left: 1px solid #e9e9e9;
  border-right: 1px solid #e9e9e9;
}

.ui-table.has-border-top {
  border-top: 1px solid #e9e9e9;
}

.ui-table.has-border-bottom {
  border-bottom: 1px solid #e9e9e9;
}

.ui-table__scroll {
  width: 100%;
  overflow: auto;
  scrollbar-gutter: stable only-if-overflowing;
  position: relative;
}

.ui-table__inner {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  table-layout: fixed;
}

.ui-table__head.is-sticky .ui-table__th {
  position: sticky;
  top: 0;
  z-index: 14;
}

.ui-table__th,
.ui-table__td {
  padding: 10px 6px;
  border-bottom: 1px solid #f0f0f0;
  border-right: 1px solid #f0f0f0;
  background: #fff;
  color: #2f2f2f;
  font-size: 12px;
  line-height: 1.4;
  box-sizing: border-box;
}

.ui-table__th:first-child,
.ui-table__td:first-child {
  border-left: 1px solid #f0f0f0;
}

.ui-table__th {
  font-size: 12px;
  font-weight: 700;
  color: #555;
  background: #efefee;
  border-left: 1px solid #e2e2e2;
  border-right: 1px solid #e2e2e2;
}

.ui-table__tr:hover .ui-table__td {
  background: #fcfcfc;
}

.ui-table__tr.is-level-0 .ui-table__td {
  background: #fafafa;
}

.ui-table__tr.is-level-0:hover .ui-table__td {
  background: #fafafa;
}

.ui-table__tr.is-level-1 .ui-table__td {
  background: #fff;
}

.ui-table__tr.is-level-1:hover .ui-table__td {
  background: #fff;
}

.ui-table__td.is-fixed-left,
.ui-table__th.is-fixed-left {
  box-shadow: 1px 0 0 #f0f0f0;
}

.ui-table__td.is-fixed-right,
.ui-table__th.is-fixed-right {
  box-shadow: -1px 0 0 #f0f0f0;
}

.ui-table__th.is-fixed-left,
.ui-table__th.is-fixed-right {
  background: #efefee;
}

.ui-table__td.align-center,
.ui-table__th.align-center {
  text-align: center;
}

.ui-table__td.align-right,
.ui-table__th.align-right {
  text-align: right;
}

.ui-table__td.align-left,
.ui-table__th.align-left {
  text-align: left;
}

.ui-table__cell-content {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
}

.ui-table__cell-content.is-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-table__control {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.ui-table__index {
  color: #111;
  font-size: 12px;
  font-weight: 400;
  font-family: inherit;
  line-height: 1.4;
  min-width: 22px;
}

.ui-table__drag {
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: grab;

  &:active {
    cursor: grabbing;
  }

  img {
    width: 14px;
    height: 14px;
    display: block;
    opacity: 0.72;
  }
}

.ui-table__tree-toggle {
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.ui-table__tree-icon {
  width: 10px;
  height: 10px;
  display: block;
  opacity: 0.62;
  transform: rotate(0deg);
  transition: transform 0.14s ease;
}

.ui-table__tree-icon.expanded {
  transform: rotate(90deg);
}

.ui-table__tree-spacer {
  width: 16px;
  flex-shrink: 0;
}

.ui-table__td.is-narrow,
.ui-table__th.is-narrow {
  padding-left: 8px;
  padding-right: 8px;
}

.ui-table__pagination {
  display: flex;
  justify-content: flex-end;
  padding: 10px 12px;
}

.ui-table__row-ghost td {
  opacity: 0.45;
}

.ui-table__row-chosen td {
  background: #f3f3f4;
}

.ui-table__row-drag td {
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
}
</style>
