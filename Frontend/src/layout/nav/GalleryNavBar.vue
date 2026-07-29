<template>
  <nav
    class="header-classify-sticky"
    :class="{ 'is-tools-expanded': toolsExpanded }"
    aria-label="相册分类与布局"
  >
    <div class="container">
      <div class="classify">
        <div class="classify-inner">
          <div class="classify-links">
            <template v-if="showGalleryNavCustomize">
              <div class="classify-links-custom">
                <draggable
                  v-model="galleryList"
                  item-key="id"
                  tag="div"
                  class="classify-tg"
                  handle=".gallery-nav-drag-handle"
                  :disabled="!galleryNavEditEnabled"
                  :animation="200"
                  :delay="100"
                  :delay-on-touch-only="true"
                  ghost-class="classify-drag-ghost"
                  chosen-class="classify-drag-chosen"
                  @end="onGalleryDragEnd"
                >
                  <template #item="{ element }">
                    <div class="classify-link-wrap">
                      <router-link class="classify-link" :to="element.link">
                        <img
                          v-if="galleryNavEditEnabled && !element.isSystem"
                          class="gallery-nav-drag-handle"
                          src="@/assets/icon/drag.svg"
                          alt=""
                          width="14"
                          height="14"
                          aria-hidden="true"
                        />
                        <span class="classify-link__label">{{ element.name }}</span>
                        <img
                          v-if="element.showLock"
                          class="classify-link__lock"
                          src="@/assets/icon/lock.svg"
                          alt=""
                          width="15"
                          height="15"
                          aria-hidden="true"
                        />
                      </router-link>
                    </div>
                  </template>
                </draggable>
                <Button
                  class="classify-add-btn"
                  icon-only
                  :icon-src="addIcon"
                  title="添加画廊"
                  aria-label="添加画廊"
                  @click="openGalleryNavAdd"
                />
              </div>
            </template>
            <template v-else>
              <router-link
                v-for="item in galleryLinks"
                :key="item.folderKey"
                class="classify-link"
                :to="item.link"
              >
                <span class="classify-link__label">{{ item.name }}</span>
                <img
                  v-if="item.showLock"
                  class="classify-link__lock"
                  src="@/assets/icon/lock.svg"
                  alt=""
                  width="15"
                  height="15"
                  aria-hidden="true"
                />
              </router-link>
            </template>
          </div>
          <div class="gallery-layout-tools">
            <div
              ref="toolsPanelRef"
              class="gallery-layout-tools__collapsed"
              :class="{ 'is-open': toolsExpanded }"
            >
              <ButtonGroup variant="toolbar" aria-label="画廊布局" class="layout-mode-toggle">
                <Button
                  class="layout-mode-btn"
                  :class="{ 'is-active': masonryLayoutStore.galleryLayoutMode === 'masonry' }"
                  width="40px"
                  icon-only
                  :icon-src="flowIcon"
                  title="瀑布流"
                  aria-label="瀑布流布局"
                  @click="masonryLayoutStore.setGalleryLayoutMode('masonry')"
                />
                <Button
                  class="layout-mode-btn"
                  :class="{ 'is-active': masonryLayoutStore.galleryLayoutMode === 'grid' }"
                  width="40px"
                  icon-only
                  :icon-src="gridIcon"
                  title="网格"
                  aria-label="网格布局"
                  @click="masonryLayoutStore.setGalleryLayoutMode('grid')"
                />
                <Button
                  class="layout-mode-btn"
                  :class="{ 'is-active': masonryLayoutStore.galleryLayoutMode === 'card' }"
                  width="40px"
                  icon-only
                  :icon-src="cardSwitchingIcon"
                  title="卡牌切换"
                  aria-label="卡牌切换"
                  @click="masonryLayoutStore.setGalleryLayoutMode('card')"
                />
              </ButtonGroup>
              <ToolbarSelect
                v-if="masonryLayoutStore.galleryLayoutMode !== 'card'"
                ref="columnPickerRef"
                :open="columnMenuOpen"
                :model-value="columnSelect"
                :display-label="columnDisplayLabel"
                trigger-label="列数"
                menu-aria-label="列数选项"
                :icon-src="columnIcon"
                :options="columnOptions"
                @toggle="toggleColumnMenu"
                @select="pickColumn"
              />
              <ToolbarSelect
                ref="itemSortPickerRef"
                :open="itemSortMenuOpen"
                :model-value="itemSortValue"
                :display-label="itemSortDisplayLabel"
                trigger-label="资源排序"
                menu-aria-label="资源排序选项"
                icon-text="序"
                :options="itemSortSelectOptions"
                @toggle="toggleItemSortMenu"
                @select="pickItemSort"
              />
              <Button
                v-if="showGalleryNavCustomize"
                class="gallery-edit-btn"
                :class="{ 'is-active': dragSortEditEnabled }"
                width="40px"
                icon-only
                :icon-src="editIcon"
                :title="dragSortEditEnabled ? '退出拖拽编辑' : '进入拖拽编辑'"
                :aria-label="dragSortEditEnabled ? '退出拖拽编辑' : '进入拖拽编辑'"
                @click="toggleDragSortEdit"
              />
            </div>
            <Button
              class="gallery-tools-toggle-btn"
              :class="{ 'is-open': toolsExpanded }"
              width="40px"
              icon-only
              :icon-src="moreIcon"
              :title="toolsExpanded ? '收起布局工具' : '展开布局工具'"
              :aria-label="toolsExpanded ? '收起布局工具' : '展开布局工具'"
              :aria-expanded="toolsExpanded"
              @click="toggleToolsExpanded"
            />
            <div class="gallery-search-slot">
              <GallerySearch ref="gallerySearchRef" v-model:open="searchOpen" />
            </div>
            <button
              v-if="hasActiveSearch"
              type="button"
              class="gallery-search-reset"
              @click="gallerySearchRef?.reset()"
            >
              重置搜索
            </button>
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute } from 'vue-router'
import draggable from 'vuedraggable'
import { useMasonryLayoutStore, type MasonryColumnChoice } from '@/stores/masonryLayout'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { useGalleryItemSortStore } from '@/stores/galleryItemSort'
import { patchCategoriesNavOrder } from '@/api/adminApi'
import { useNavAddModalStore } from '@/stores/navAddModal'
import { useAuthStore } from '@/stores/auth'
import { useDragSortEditStore } from '@/stores/dragSortEdit'
import type { GalleryClassificationLink } from '@/stores/categoryNav'
import { useGallerySearchStore } from '@/stores/gallerySearch'
import { isGallerySearchFolderKey } from '@/utils/gallerySearchFolder'
import type { GalleryItemSortMode } from '@/utils/galleryItemOrder'
import Button from '@/components/shared-ui/Button.vue'
import ButtonGroup from '@/components/shared-ui/ButtonGroup.vue'
import GallerySearch from '@/components/gallery/GallerySearch.vue'
import ToolbarSelect from '@/components/shared-ui/ToolbarSelect.vue'
import addIcon from '@/assets/icon/add.svg'
import columnIcon from '@/assets/icon/column.svg'
import editIcon from '@/assets/icon/edit.svg'
import flowIcon from '@/assets/icon/Flow.svg'
import gridIcon from '@/assets/icon/Grid.svg'
import cardSwitchingIcon from '@/assets/icon/card-switching.png'
import moreIcon from '@/assets/icon/more.svg'

const route = useRoute()
const categoryNavStore = useCategoryNavStore()
const gallerySearchStore = useGallerySearchStore()
const gallerySearchRef = ref<InstanceType<typeof GallerySearch> | null>(null)
const navAddModalStore = useNavAddModalStore()
const authStore = useAuthStore()
const dragSortEditStore = useDragSortEditStore()
const itemSortStore = useGalleryItemSortStore()
const { overrides: itemSortOverrides, serverDefaults: itemSortServerDefaults } =
  storeToRefs(itemSortStore)

const { authenticated, authConfigured } = storeToRefs(authStore)
const { enabled: dragSortEditEnabled } = storeToRefs(dragSortEditStore)
const showGalleryNavCustomize = computed(() => authConfigured.value && authenticated.value)
const galleryNavEditEnabled = computed(() => showGalleryNavCustomize.value && dragSortEditEnabled.value)
const hasActiveSearch = computed(
  () =>
    Boolean(gallerySearchStore.query.trim()) ||
    gallerySearchStore.selectedCategories.length > 0 ||
    gallerySearchStore.selectedExtensions.length > 0 ||
    gallerySearchStore.selectedTags.length > 0,
)

const activeFolderKey = computed(() => {
  const fk = route.params.folderKey
  if (typeof fk === 'string' && fk) return fk
  return categoryNavStore.homeFolderKey
})

const galleryLinks = computed(() =>
  categoryNavStore.galleryStripLinksForFolder(activeFolderKey.value),
)

const galleryMajorId = computed(() => {
  let fk = activeFolderKey.value
  if (isGallerySearchFolderKey(fk)) {
    fk = gallerySearchStore.searchReturnFolderKey.trim() || categoryNavStore.homeFolderKey
  }
  return categoryNavStore.galleryMajorForFolderKey(fk)?.id ?? null
})

const galleryList = ref<GalleryClassificationLink[]>([])

watch(
  () =>
    `${galleryMajorId.value ?? 'x'}:${galleryLinks.value.map((l) => l.id).join('|')}`,
  () => {
    galleryList.value = galleryLinks.value.map((x) => ({ ...x }))
  },
  { immediate: true },
)

async function onGalleryDragEnd() {
  if (!galleryNavEditEnabled.value) return
  const mid = galleryMajorId.value
  if (mid == null) return
  const ordered = galleryList.value.filter((l) => !l.isSystem).map((l) => l.id)
  const prev = galleryLinks.value.filter((l) => !l.isSystem).map((l) => l.id)
  if (ordered.length === prev.length && ordered.every((id, i) => id === prev[i])) return
  try {
    const doc = await patchCategoriesNavOrder({
      subOrders: [{ majorId: mid, subIds: ordered }],
    })
    categoryNavStore.replaceDoc(doc)
  } catch (e) {
    console.error(e)
    galleryList.value = galleryLinks.value.map((x) => ({ ...x }))
    window.alert('画廊分类顺序保存失败，请检查权限或网络')
  }
}

function toggleDragSortEdit() {
  dragSortEditStore.toggle()
}

const itemSortPickerRef = ref<{ rootEl: HTMLElement | null; panelEl: HTMLElement | null } | null>(null)
const itemSortMenuOpen = ref(false)
const toolsExpanded = ref(false)
const toolsPanelRef = ref<HTMLElement | null>(null)

const itemSortOptions = [
  { value: 'uploaded_at' as const, label: '上传时间' },
  { value: 'updated_at' as const, label: '更新时间' },
  { value: 'sort' as const, label: '自定义' },
]
const itemSortSelectOptions = computed(() =>
  itemSortOptions.map((opt) => ({ value: opt.value as string, label: opt.label })),
)

const itemSortValue = computed(() => {
  void itemSortOverrides.value
  void itemSortServerDefaults.value
  return itemSortStore.effectiveMode(activeFolderKey.value)
})

const itemSortDisplayLabel = computed(() => {
  const v = itemSortValue.value
  const hit = itemSortOptions.find((o) => o.value === v)
  return hit?.label ?? '上传时间'
})

function pickItemSort(mode: string) {
  if (!isGalleryItemSortMode(mode)) return
  itemSortStore.setOverride(activeFolderKey.value, mode)
  itemSortMenuOpen.value = false
}

function isGalleryItemSortMode(v: string): v is GalleryItemSortMode {
  return v === 'uploaded_at' || v === 'updated_at' || v === 'sort'
}

function toggleItemSortMenu() {
  if (!toolsExpanded.value) return
  columnMenuOpen.value = false
  itemSortMenuOpen.value = !itemSortMenuOpen.value
}

function toggleToolsExpanded() {
  toolsExpanded.value = !toolsExpanded.value
  if (!toolsExpanded.value) {
    collapseTools()
  }
}

function collapseTools() {
  toolsExpanded.value = false
  columnMenuOpen.value = false
  itemSortMenuOpen.value = false
  dragSortEditStore.setEnabled(false)
}

function onDocPointerdownGalleryToolbars(e: PointerEvent) {
  const target = e.target as Element | null
  const targetNode = e.target as Node | null
  const insideColumnMenu = Boolean(
    (columnPickerRef.value?.rootEl && targetNode && columnPickerRef.value.rootEl.contains(targetNode)) ||
      (columnPickerRef.value?.panelEl && targetNode && columnPickerRef.value.panelEl.contains(targetNode)),
  )
  const insideItemSortMenu = Boolean(
    (itemSortPickerRef.value?.rootEl && targetNode && itemSortPickerRef.value.rootEl.contains(targetNode)) ||
      (itemSortPickerRef.value?.panelEl && targetNode && itemSortPickerRef.value.panelEl.contains(targetNode)),
  )
  if (
    toolsExpanded.value &&
    !dragSortEditEnabled.value &&
    !toolsPanelRef.value?.contains(target) &&
    !target?.closest('.gallery-tools-toggle-btn') &&
    !insideColumnMenu &&
    !insideItemSortMenu
  ) {
    collapseTools()
    return
  }
  if (
    columnMenuOpen.value &&
    !(
      (columnPickerRef.value?.rootEl && columnPickerRef.value.rootEl.contains(e.target as Node)) ||
      (columnPickerRef.value?.panelEl && columnPickerRef.value.panelEl.contains(e.target as Node))
    )
  ) {
    columnMenuOpen.value = false
  }
  if (
    itemSortMenuOpen.value &&
    !(
      (itemSortPickerRef.value?.rootEl && itemSortPickerRef.value.rootEl.contains(e.target as Node)) ||
      (itemSortPickerRef.value?.panelEl && itemSortPickerRef.value.panelEl.contains(e.target as Node))
    )
  ) {
    itemSortMenuOpen.value = false
  }
}

function openGalleryNavAdd() {
  const major = categoryNavStore.galleryMajorForFolderKey(activeFolderKey.value)
  navAddModalStore.openGallery(major?.id, major?.name)
}

/** 与 header 同步：搜索打开时参与全局滚动锁与 Esc */
const searchOpen = defineModel<boolean>('searchOpen', { default: false })

const masonryLayoutStore = useMasonryLayoutStore()

const columnPickerRef = ref<{ rootEl: HTMLElement | null; panelEl: HTMLElement | null } | null>(null)
const columnMenuOpen = ref(false)

const fixedColumnOptions = [1, 2, 3, 4, 5, 6] as const
const columnOptions = computed(() => [
  { value: 'auto', label: '自动' },
  ...fixedColumnOptions.map((n) => ({ value: String(n), label: String(n) })),
])

const columnSelect = computed({
  get(): string {
    const c = masonryLayoutStore.columnChoice
    return c === 'auto' ? 'auto' : String(c)
  },
  set(v: string) {
    if (v === 'auto') {
      masonryLayoutStore.setColumnChoice('auto')
      return
    }
    const n = Number(v)
    if (n >= 1 && n <= 6)
      masonryLayoutStore.setColumnChoice(n as Exclude<MasonryColumnChoice, 'auto'>)
  },
})

const columnDisplayLabel = computed(() =>
  columnSelect.value === 'auto' ? '自动' : columnSelect.value,
)

function pickColumn(v: string) {
  columnSelect.value = v
  columnMenuOpen.value = false
}

function toggleColumnMenu() {
  if (!toolsExpanded.value) return
  itemSortMenuOpen.value = false
  columnMenuOpen.value = !columnMenuOpen.value
}

function closeColumnMenu() {
  columnMenuOpen.value = false
  itemSortMenuOpen.value = false
}

watch([columnMenuOpen, itemSortMenuOpen], ([colOpen, sortOpen]) => {
  document.removeEventListener('pointerdown', onDocPointerdownGalleryToolbars, true)
  if (toolsExpanded.value || colOpen || sortOpen) {
    nextTick(() => document.addEventListener('pointerdown', onDocPointerdownGalleryToolbars, true))
  }
})

watch(toolsExpanded, (expanded) => {
  document.removeEventListener('pointerdown', onDocPointerdownGalleryToolbars, true)
  if (expanded) {
    nextTick(() => document.addEventListener('pointerdown', onDocPointerdownGalleryToolbars, true))
  }
})

watch(
  () => showGalleryNavCustomize.value,
  (ok) => {
    if (!ok) dragSortEditStore.setEnabled(false)
  },
)

watch(
  () => masonryLayoutStore.galleryLayoutMode,
  (mode) => {
    if (mode === 'card') columnMenuOpen.value = false
  },
)

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocPointerdownGalleryToolbars, true)
})

defineExpose({
  /** 列数或资源排序下拉是否打开（供外壳统一处理 Esc） */
  get columnMenuOpen() {
    return columnMenuOpen.value || itemSortMenuOpen.value
  },
  closeColumnMenu,
})
</script>

<style scoped lang="scss">
$ease-brand: cubic-bezier(0.22, 1, 0.36, 1);

.header-classify-sticky {
  position: sticky;
  top: env(safe-area-inset-top, 0px);
  z-index: 1000;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  padding-top: 6px;
  padding-bottom: 10px;
  margin: 0;
  backdrop-filter: blur(12px) saturate(1);
  background: rgba(255, 255, 255, 0.7);
  transition: padding-bottom 0.24s $ease-brand;

  .classify {
    width: 100%;
    box-sizing: border-box;

    &-inner {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: clamp(8px, 2vw, 16px);
      flex-wrap: nowrap;
      width: 100%;
    }

    .classify-links {
      display: flex;
      gap: clamp(8px, 2vw, 12px);
      flex-wrap: nowrap;
      align-items: center;
      flex: 1 1 auto;
      min-width: 0;
      overflow-x: auto;
      overflow-y: hidden;
      -webkit-overflow-scrolling: touch;
    }

    .classify-links-custom {
      display: flex;
      flex-wrap: nowrap;
      align-items: center;
      gap: clamp(8px, 2vw, 12px);
      flex: 0 0 auto;
      min-width: max-content;
    }

    .classify-tg {
      display: flex;
      flex-wrap: nowrap;
      align-items: center;
      gap: clamp(8px, 2vw, 12px);
      flex: 0 0 auto;
      min-width: max-content;
    }

    :deep(.classify-drag-ghost) {
      opacity: 0.45;
    }

    :deep(.classify-drag-chosen) {
      cursor: grabbing;
    }

    .classify-link-wrap {
      display: inline-flex;
      flex-direction: row;
      align-items: stretch;
      max-width: 100%;
      border-radius: 4px;
      -webkit-tap-highlight-color: transparent;

      > .classify-link {
        flex: 1;
        min-width: 0;
      }
    }

    .classify-add-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: auto;
      height: auto;
      padding: 6px 8px;
      margin: 0;
      border-radius: 6px;
      background: transparent;
      flex-shrink: 0;
      -webkit-tap-highlight-color: transparent;
      transition: background-color 0.22s $ease-brand;

      &:hover {
        background: rgba(0, 0, 0, 0.06);
      }

      &:focus-visible {
        outline: 2px solid rgba(0, 0, 0, 0.35);
        outline-offset: 2px;
      }

      :deep(.ui-button__icon) {
        display: block;
        width: clamp(16px, 3.2vw, 18px);
        height: clamp(16px, 3.2vw, 18px);
        object-fit: contain;
        opacity: 0.72;
      }

      &:hover :deep(.ui-button__icon) {
        opacity: 1;
      }
    }

    .classify-link {
      display: inline-flex;
      align-items: center;
      justify-content: space-between;
      gap: 0.4rem;
      max-width: none;
      box-sizing: border-box;
      font-size: 16px;
      font-weight: 500;
      color: #404040;
      padding: 6px 0;
      border-radius: 4px;
      transition: color 0.22s $ease-brand;
      text-decoration: none;
      white-space: nowrap;

      &:hover {
        color: #000;
      }

      &.router-link-active {
        color: #000;
        font-weight: 700;
      }
    }

    .gallery-nav-drag-handle {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
      display: block;
      object-fit: contain;
      opacity: 0.62;
      cursor: grab;
    }

    :deep(.classify-drag-chosen) .gallery-nav-drag-handle {
      cursor: grabbing;
      opacity: 0.92;
    }

    .classify-link__label {
      min-width: 0;
    }

    .classify-link__lock {
      flex-shrink: 0;
      width: 15px;
      height: 15px;
      display: block;
      object-fit: contain;
      opacity: 0.52;
    }

    .gallery-layout-tools {
      --gallery-toolbar-h: 36px;
      --gallery-toolbar-inner: 30px;
      --gallery-toolbar-radius: 8px;
      --gallery-toolbar-bg: #111111;

      display: inline-flex;
      align-items: center;
      gap: clamp(8px, 2vw, 12px);
      margin-left: auto;
      flex-shrink: 0;
      min-width: max-content;
      max-width: 100%;
      justify-content: flex-end;
    }

    .gallery-layout-tools__collapsed {
      display: inline-flex;
      align-items: center;
      gap: clamp(8px, 2vw, 12px);
      min-width: 0;
      overflow: visible;
      max-width: 0;
      opacity: 0;
      transform: translateX(8px);
      pointer-events: none;
      transition:
        max-width 0.22s $ease-brand,
        opacity 0.16s ease,
        transform 0.22s $ease-brand;

      &.is-open {
        max-width: min(480px, calc(100vw - 24px));
        opacity: 1;
        transform: translateX(0);
        pointer-events: auto;
      }
    }

    .gallery-search-slot {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
      min-width: 0;
    }

    .gallery-search-reset {
      margin-left: 6px;
      padding: 4px 0;
      border: 0;
      background: transparent;
      color: #666;
      font-size: 12px;
      cursor: pointer;
      white-space: nowrap;

      &:hover {
        color: #111;
        text-decoration: underline;
      }
    }

    .layout-mode-toggle {
      height: var(--gallery-toolbar-h);
      min-height: var(--gallery-toolbar-h);
      border-radius: var(--gallery-toolbar-radius);
      background: var(--gallery-toolbar-bg);
    }

    .layout-mode-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 34px;
      height: var(--gallery-toolbar-inner);
      padding: 0;
      border-radius: 6px;
      background: transparent;
      transition:
        background 0.2s $ease-brand,
        box-shadow 0.2s $ease-brand;

      &:hover {
        background: rgba(255, 255, 255, 0.08);
      }

      &.is-active {
        background: #2c2c2c;
      }

      :deep(.ui-button__icon) {
        width: 18px;
        height: 18px;
        display: block;
        object-fit: contain;
        opacity: 0.55;
        filter: invert(1);
      }

      &.is-active :deep(.ui-button__icon) {
        opacity: 1;
      }
    }

    :deep(.toolbar-select) {
      height: var(--gallery-toolbar-h);
      min-height: var(--gallery-toolbar-h);
      border-radius: var(--gallery-toolbar-radius);
      background: var(--gallery-toolbar-bg);
    }

    :deep(.toolbar-select__trigger) {
      height: var(--gallery-toolbar-inner);
      min-height: var(--gallery-toolbar-inner);
      line-height: var(--gallery-toolbar-inner);
    }

    .gallery-edit-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: var(--gallery-toolbar-h);
      height: var(--gallery-toolbar-h);
      min-height: var(--gallery-toolbar-h);
      border-radius: var(--gallery-toolbar-radius);
      background: var(--gallery-toolbar-bg);
      box-shadow: 0 10px 28px rgba(0, 0, 0, 0.22);
      padding: 0;

      :deep(.ui-button__icon) {
        width: 18px;
        height: 18px;
        display: block;
        object-fit: contain;
        opacity: 0.62;
        filter: invert(1);
      }

      &:hover :deep(.ui-button__icon),
      &.is-active :deep(.ui-button__icon) {
        opacity: 1;
      }
    }

    .gallery-tools-toggle-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: var(--gallery-toolbar-h);
      height: var(--gallery-toolbar-h);
      min-height: var(--gallery-toolbar-h);
      border-radius: var(--gallery-toolbar-radius);
      background: var(--gallery-toolbar-bg);
      box-shadow: 0 10px 28px rgba(0, 0, 0, 0.22);
      padding: 0;
      flex-shrink: 0;

      :deep(.ui-button__icon) {
        width: 18px;
        height: 18px;
        display: block;
        object-fit: contain;
        opacity: 0.62;
        filter: invert(1);
        transition:
          transform 0.22s $ease-brand,
          opacity 0.16s ease;
      }

      &:hover :deep(.ui-button__icon),
      &.is-open :deep(.ui-button__icon) {
        opacity: 1;
      }

      &.is-open :deep(.ui-button__icon) {
        transform: rotate(90deg);
      }
    }
  }
}

@media (max-width: 640px) {
  .header-classify-sticky {
    padding-top: 4px;
    padding-bottom: 8px;

    &.is-tools-expanded {
      padding-bottom: 50px;
    }

    .classify .classify-inner {
      gap: 6px;
      flex-wrap: nowrap;
      align-items: center;
    }

    .classify .gallery-layout-tools {
      position: relative;
      margin-left: auto;
      justify-content: flex-end;
      flex-wrap: nowrap;
      gap: 8px;
    }

    .classify .gallery-layout-tools__collapsed {
      position: absolute;
      top: calc(100% + 8px);
      right: 0;
      justify-content: flex-end;
      flex-wrap: wrap;
      gap: 8px;
      width: max-content;
      max-width: min(calc(100vw - 24px), 420px);
      transform: translateY(-4px);
      z-index: 1200;
    }

    .classify .gallery-layout-tools__collapsed.is-open {
      transform: translateY(0);
    }

    .classify :deep(.toolbar-select__icon) {
      width: 16px;
      height: 16px;
    }

    .classify :deep(.toolbar-select__trigger) {
      font-size: 13px;
      min-width: 4.25rem;
    }

    .classify :deep(.toolbar-select__option) {
      font-size: 13px;
      padding: 7px 8px;
    }
  }
}

@media (max-width: 400px) {
  .header-classify-sticky .classify .gallery-layout-tools {
    --gallery-toolbar-h: 32px;
    --gallery-toolbar-inner: 28px;
    gap: 6px;
  }

  .header-classify-sticky .classify .gallery-layout-tools__collapsed {
    max-width: calc(100vw - 20px);
    gap: 6px;
  }

  .header-classify-sticky .classify .layout-mode-toggle {
    padding: 2px;
    gap: 3px;
  }

  .header-classify-sticky .classify .layout-mode-btn {
    width: 30px;
    height: var(--gallery-toolbar-inner);
  }

  .header-classify-sticky .classify :deep(.toolbar-select__trigger) {
    font-size: 12px;
    min-width: 3.15rem;
    padding: 0 4px 0 4px;
  }

  .header-classify-sticky .classify :deep(.toolbar-select__option) {
    font-size: 12px;
    padding: 6px 8px;
  }

  .header-classify-sticky .classify :deep(.toolbar-select__icon) {
    width: 16px;
    height: 16px;
  }
}
</style>
