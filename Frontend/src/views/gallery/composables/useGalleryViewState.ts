import {
  ref,
  shallowRef,
  onMounted,
  onUnmounted,
  computed,
  nextTick,
  watch,
  type Ref,
  type ComputedRef,
} from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute } from 'vue-router'
import axios from 'axios'
import type { GalleryFileDetailPayload } from '@/components/viewers'
import { useMasonryLayoutStore, type MasonryColumnChoice } from '@/stores/masonryLayout'
import {
  useGallerySearchStore,
  filterGalleryRemoteRows,
  buildFilterOptionsFromRows,
  type GalleryRemoteRow,
  type GallerySearchFilters,
} from '@/stores/gallerySearch'
import { GALLERY_SEARCH_FOLDER_KEY, isGallerySearchFolderKey } from '@/utils/gallerySearchFolder'
import {
  parseGallerySearchQuery,
  gallerySearchQueryHasCriteria,
} from '@/utils/gallerySearchQuery'
import {
  fetchCategoryDetail,
  postCategoryViewUnlock,
  patchCategoryItemOrder,
  patchCategoryLayout,
  deleteCategoryItem,
  transferCategoryItem,
  type MasonryPlacementMap,
} from '@/api/gallery'
import type { ApiLayout } from '@/api/types'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { useAuthStore } from '@/stores/auth'
import { useGalleryItemSortStore } from '@/stores/galleryItemSort'
import { useGalleryItemsSyncStore } from '@/stores/galleryItemsSync'
import { useDragSortEditStore } from '@/stores/dragSortEdit'
import { useNavAddModalStore } from '@/stores/navAddModal'
import { isMajorPlaceholderFolderKey } from '@/utils/categoryFolderKey'
import { sortGalleryRows, type GalleryItemSortMode } from '@/utils/galleryItemOrder'
import { galleryMediaKindFromUrl, isGalleryImageItem } from '@/utils/galleryMedia'
import { copyTextToClipboard } from '@/utils/clipboard'
import { resolveEditorOriginalUrl, buildShareableResourceUrl } from '@/utils/resourceUrl'
import type { GalleryItemEditPayload } from '@/components/gallery/GalleryItemEditDialog.vue'
import { useMessageStore } from '@/stores/message'
import type GalleryStage from '../GalleryStage.vue'
import type { GalleryListItem, LoadPageOpts } from '../types'
import {
  formatCardDateFromIso,
  linkNameFromResourceUrl,
  resolveGalleryItemContext,
  toAbsoluteResourceUrl,
  isGalleryVideoItem,
} from '../utils'
import {
  getGalleryViewGrant,
  setGalleryViewGrant,
  clearGalleryViewGrant,
} from '@/utils/galleryViewGrant'
import { folderResourceRequiresViewKeyByFolderKey } from '@/utils/galleryAccess'
import { useGalleryUploadPanel } from './useGalleryUploadPanel'
import {
  buildGridPageSizeOptions,
  snapPageSizeToGridOptions,
} from '@/utils/gridPagination'

const CARD_INTRO_ANIM_SEC = 0.45

const PAGE_SIZE_OPTIONS = [10, 20, 30, 50, 100] as const
const DEFAULT_PAGE_SIZE = 20
const MAX_PAGE_SIZE = 150

/** Append ?_t=updatedAt for cache-busting after file replacement */
function appendCacheBust(u: string, updatedAt?: string): string {
  if (!u || !updatedAt) return u
  return u + (u.includes('?') ? '&' : '?') + '_t=' + encodeURIComponent(updatedAt)
}


const NEW_ITEM_STYLE_BY_BATCH_INDEX: ReadonlyArray<Readonly<Record<string, string>>> = Array.from(
  { length: MAX_PAGE_SIZE },
  (_, batchIndex) => Object.freeze({ animationDelay: `${batchIndex * 0.05}s` }),
)

export function useGalleryViewState(folderKey: Ref<string> | ComputedRef<string>) {
  const route = useRoute()
  const isGlobalSearchView = computed(() => isGallerySearchFolderKey(folderKey.value))
  const picRows = shallowRef<GalleryRemoteRow[]>([])
  const categoryDisplayName = ref('')

  const pageSize = ref<number>(DEFAULT_PAGE_SIZE)
  const pageSizeMenuOpen = ref(false)
  const gridColumnCount = ref(0)
  const gridRowsPerScreen = ref(0)

  const pageSizeSelectOptions = computed(() => {
    if (
      galleryLayoutMode.value === 'grid' &&
      gridColumnCount.value > 0 &&
      gridRowsPerScreen.value > 0
    ) {
      return buildGridPageSizeOptions(gridColumnCount.value, gridRowsPerScreen.value).map(
        (size) => ({
          value: String(size),
          label: String(size),
        }),
      )
    }
    return PAGE_SIZE_OPTIONS.map((size) => ({
      value: String(size),
      label: String(size),
    }))
  })
  const pageSizeSelectValue = computed(() => String(pageSize.value))

  const masonryLayoutStore = useMasonryLayoutStore()
  const { galleryLayoutMode, columnChoice } = storeToRefs(masonryLayoutStore)

  const gallerySearchStore = useGallerySearchStore()
  const {
    query: searchQuery,
    selectedCategories,
    selectedExtensions,
    selectedTags,
    searchScope,
  } = storeToRefs(gallerySearchStore)
  const categoryNavStore = useCategoryNavStore()
  const navAddModalStore = useNavAddModalStore()
  const authStore = useAuthStore()
  const messageStore = useMessageStore()
  const dragSortEditStore = useDragSortEditStore()
  const { enabled: dragSortEditEnabled } = storeToRefs(dragSortEditStore)
  const galleryItemsSyncStore = useGalleryItemsSyncStore()

  const globalPool = shallowRef<GalleryRemoteRow[]>([])
  const globalPoolLoading = ref(false)
  let globalPoolGen = 0

  async function refreshGlobalPool() {
    const gen = ++globalPoolGen
    globalPoolLoading.value = true
    try {
      if (!categoryNavStore.loaded) {
        await categoryNavStore.fetchFromServer()
      }
      const cats = categoryNavStore.visibleLeafSubcategories
      if (cats.length === 0) {
        if (gen === globalPoolGen) globalPool.value = []
        return
      }
      const settled = await Promise.allSettled(cats.map((c) => fetchCategoryDetail(c.folderKey)))
      if (gen !== globalPoolGen) return
      const rows: GalleryRemoteRow[] = []
      for (const r of settled) {
        if (r.status !== 'fulfilled') continue
        const detail = r.value
        const fk = detail.folderKey
        for (const it of detail.items) {
          rows.push({
            id: `${fk}:${it.id}`,
            url: appendCacheBust(authStore.appendAccessToResourceUrl(it.url), it.updatedAt),
            shortUrl: it.shortUrl ? authStore.appendAccessToResourceUrl(it.shortUrl) : undefined,
            linkName: it.linkName,
            originalUrl: it.originalUrl
              ? authStore.appendAccessToResourceUrl(it.originalUrl)
              : undefined,
            editedUrl: it.editedUrl ? authStore.appendAccessToResourceUrl(it.editedUrl) : undefined,
            useEdited: it.useEdited,
            format: it.format,
            mediaKind: it.mediaKind,
            isLivePhoto: it.isLivePhoto,
            liveVideoUrl: it.liveVideoUrl
              ? authStore.appendAccessToResourceUrl(it.liveVideoUrl)
              : undefined,
            categoryName: detail.name,
            thumbnailUrl: it.thumbnailUrl
              ? appendCacheBust(authStore.appendAccessToResourceUrl(it.thumbnailUrl), it.updatedAt)
              : undefined,
            title: it.title,
            tags: it.tags,
            uploadedAt: it.uploadedAt,
            updatedAt: it.updatedAt,
            sort: it.sort,
            masonryCol: it.masonryCol,
            masonryRow: it.masonryRow,
            fileSize: it.fileSize,
          })
        }
      }
      globalPool.value = rows
    } catch {
      if (gen === globalPoolGen) globalPool.value = []
    } finally {
      if (gen === globalPoolGen) globalPoolLoading.value = false
    }
  }

  const searchSourceRows = computed(() =>
    isGlobalSearchView.value ? globalPool.value : picRows.value,
  )

  const hasActiveFilter = computed(() =>
    gallerySearchQueryHasCriteria({
      query: searchQuery.value,
      categories: selectedCategories.value,
      extensions: selectedExtensions.value,
      tags: selectedTags.value,
    }),
  )

  const activeSearchFilters = computed((): GallerySearchFilters | null => {
    if (isGlobalSearchView.value) {
      if (searchScope.value !== 'all') return null
      if (!hasActiveFilter.value) return null
    } else if (searchScope.value !== 'current') {
      return null
    }
    return {
      query: searchQuery.value,
      categories: selectedCategories.value,
      extensions: selectedExtensions.value,
      tags: selectedTags.value,
    }
  })

  const searchNeedsCriteria = computed(
    () => isGlobalSearchView.value && !hasActiveFilter.value && !loading.value && !globalPoolLoading.value,
  )

  function syncSearchFiltersFromRoute() {
    if (!isGlobalSearchView.value) return
    const parsed = parseGallerySearchQuery(
      route.query as Record<string, string | string[] | undefined | null>,
    )
    if (parsed.from) gallerySearchStore.setSearchReturnFolderKey(parsed.from)
    gallerySearchStore.setFilters({
      query: parsed.query,
      categories: parsed.categories,
      extensions: parsed.extensions,
      tags: parsed.tags,
      searchScope: 'all',
    })
  }

  watch(
    picRows,
    (rows) => {
      gallerySearchStore.setFilterOptions('current', buildFilterOptionsFromRows(rows))
    },
    { immediate: true },
  )

  watch(
    globalPool,
    (rows) => {
      gallerySearchStore.setFilterOptions('all', buildFilterOptionsFromRows(rows))
    },
    { immediate: true },
  )

  const filteredRows = computed(() => {
    const source = searchSourceRows.value
    const filters = activeSearchFilters.value
    if (isGlobalSearchView.value && !filters) return []
    if (!filters) return source
    return filterGalleryRemoteRows(source, filters)
  })

  const itemSortStore = useGalleryItemSortStore()
  const { overrides: itemSortOverrides, serverDefaults: itemSortServerDefaults } =
    storeToRefs(itemSortStore)
  const effectiveItemSort = computed<GalleryItemSortMode>(() => {
    void itemSortOverrides.value
    void itemSortServerDefaults.value
    return itemSortStore.effectiveMode(folderKey.value)
  })

  const orderedFilteredRows = computed(() =>
    sortGalleryRows(filteredRows.value, effectiveItemSort.value),
  )

  const galleryStageRef = ref<InstanceType<typeof GalleryStage> | null>(null)

  const waterfallStageVisible = ref(false)
  const waterfallRendered = ref(false)
  const displayList = ref<GalleryListItem[]>([])
  const currentPage = ref(1)

  const totalPages = computed(() =>
    orderedFilteredRows.value.length === 0
      ? 0
      : Math.ceil(orderedFilteredRows.value.length / pageSize.value),
  )

  const canPrev = computed(() => currentPage.value > 1 && totalPages.value > 0)
  const canNext = computed(() => currentPage.value < totalPages.value)
  const showPagination = computed(
    () => totalPages.value > 0 && waterfallStageVisible.value && waterfallRendered.value,
  )
  const canDragSort = computed(
    () =>
      !isGlobalSearchView.value &&
      authStore.authenticated &&
      dragSortEditEnabled.value &&
      searchScope.value === 'current' &&
      !hasActiveFilter.value &&
      !loading.value &&
      !accessBlocked.value,
  )

  const showGalleryAdminActions = computed(
    () => authStore.authenticated && !loading.value && !accessBlocked.value,
  )

  let loadPageGeneration = 0
  let clearIsNewTimer: ReturnType<typeof setTimeout> | null = null

  function clearClearIsNewTimer() {
    if (clearIsNewTimer !== null) {
      clearTimeout(clearIsNewTimer)
      clearIsNewTimer = null
    }
  }

  const loadPage = async (page: number, opts?: LoadPageOpts) => {
    const source = orderedFilteredRows.value
    if (source.length === 0) {
      displayList.value = []
      return
    }

    const silent = opts?.silent ?? false
    const gen = ++loadPageGeneration
    if (!silent) {
      waterfallRendered.value = false
      waterfallStageVisible.value = false
      clearClearIsNewTimer()
    }

    const size = pageSize.value
    const lastPage = Math.max(1, Math.ceil(source.length / size))
    const p = Math.min(Math.max(1, Math.round(page)), lastPage)
    currentPage.value = p

    const startIdx = (p - 1) * size
    const batch = source.slice(startIdx, startIdx + size)

    displayList.value = batch.map((row, i) => ({
      id: row.id || `${startIdx + i}`,
      fullSrc: row.url,
      shortUrl: row.shortUrl,
      originalUrl: row.originalUrl || row.url,
      editedUrl: row.editedUrl,
      useEdited: row.useEdited,
      title: row.title,
      tags: row.tags,
      linkName: row.linkName || linkNameFromResourceUrl(row.shortUrl || row.url),
      uploadedAt: row.uploadedAt,
      updatedAt: row.updatedAt,
      fileSize: row.fileSize,
      format: row.format,
      mediaKind: row.mediaKind,
      isLivePhoto: row.isLivePhoto,
      liveVideoUrl: row.liveVideoUrl,
      categoryName: row.categoryName || categoryDisplayName.value,
      /** 列表只加载缩略图（无缩略图时才用原图 URL） */
      src: row.thumbnailUrl || row.url,
      cardDate: formatCardDateFromIso(row.uploadedAt || row.updatedAt),
      isNew: !silent,
      loadTime: Date.now(),
      masonryAspectHW: isGalleryVideoItem({ fullSrc: row.url, mediaKind: row.mediaKind })
        ? 9 / 16
        : galleryMediaKindFromUrl(row.url) === 'image'
          ? 4 / 3
          : 3 / 4,
      masonryCol: row.masonryCol,
      masonryRow: row.masonryRow,
    }))

    if (silent) return

    await nextTick()
    if (gen !== loadPageGeneration) return

    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (gen !== loadPageGeneration) return
        waterfallStageVisible.value = true
        if (opts?.scrollToWaterfall) {
          galleryStageRef.value?.waterfallScrollAnchor?.scrollIntoView({
            behavior: 'smooth',
            block: 'start',
          })
        }
      })
    })

    const settleMs = CARD_INTRO_ANIM_SEC * 1000 + 120
    clearIsNewTimer = window.setTimeout(() => {
      clearIsNewTimer = null
      if (gen !== loadPageGeneration) return
      const list = displayList.value
      let changed = false
      const next = list.map((item) => {
        if (!item.isNew) return item
        changed = true
        return { ...item, isNew: false }
      })
      if (changed) displayList.value = next
    }, settleMs)
  }

  watch(
    [searchQuery, selectedCategories, selectedExtensions, selectedTags],
    () => {
      void loadPage(1)
    },
    { deep: true },
  )

  watch(
    () => searchScope.value,
    async (scope) => {
      if (isGlobalSearchView.value) {
        if (scope === 'all') await reloadGlobalSearchResults()
        return
      }
      if (scope === 'current') void loadPage(1)
    },
  )

  watch(
    () => gallerySearchStore.globalPoolRefreshNonce,
    () => {
      if (isGlobalSearchView.value) void reloadGlobalSearchResults()
      else void refreshGlobalPool()
    },
  )

  watch(
    () => route.query,
    () => {
      if (!isGlobalSearchView.value) return
      void reloadGlobalSearchResults()
    },
    { deep: true },
  )

  watch(
    globalPool,
    () => {
      if (!isGlobalSearchView.value || globalPoolLoading.value || loading.value) return
      if (!hasActiveFilter.value) return
      void loadPage(1)
    },
  )

  let skipNextSortModeReload = false
  watch(effectiveItemSort, () => {
    if (skipNextSortModeReload) {
      skipNextSortModeReload = false
      return
    }
    void loadPage(1)
  })

  watch(pageSize, () => {
    pageSizeMenuOpen.value = false
    void loadPage(1)
  })

  watch(showPagination, (visible) => {
    if (!visible) pageSizeMenuOpen.value = false
  })
  watch(dragSortEditEnabled, (enabled) => {
    if (!enabled) pageSizeMenuOpen.value = false
  })

  function togglePageSizeMenu() {
    pageSizeMenuOpen.value = !pageSizeMenuOpen.value
  }

  function selectPageSize(size: string) {
    const next = Number(size)
    if (!Number.isFinite(next) || next <= 0) return
    const allowed = pageSizeSelectOptions.value.map((opt) => Number(opt.value))
    if (!allowed.includes(next)) return
    pageSize.value = next
  }

  function onPageSizeMenuPointerDown(event: PointerEvent) {
    if (!pageSizeMenuOpen.value) return
    const target = event.target
    if (!(target instanceof Node)) return
    const picker = galleryStageRef.value?.pageSizePickerRef
    const root = picker?.rootEl
    const panel = picker?.panelEl
    if (root?.contains(target) || panel?.contains(target)) return
    pageSizeMenuOpen.value = false
  }

  function handleGalleryLayout() {
    if (displayList.value.length === 0) return
    waterfallRendered.value = true
    void syncGridPaginationIfNeeded()
  }

  async function syncGridPaginationIfNeeded() {
    if (galleryLayoutMode.value !== 'grid') return
    await nextTick()
    const grid = galleryStageRef.value?.gridLayoutRef
    if (!grid) return
    grid.syncGridMetrics()
    const cols = grid.gridColumnCount
    const rows = grid.gridRowsPerScreen
    if (cols <= 0 || rows <= 0) return

    const prevBlock = gridColumnCount.value * gridRowsPerScreen.value
    const nextBlock = cols * rows
    gridColumnCount.value = cols
    gridRowsPerScreen.value = rows

    const options = buildGridPageSizeOptions(cols, rows)
    const snapped = snapPageSizeToGridOptions(pageSize.value, options)
    if (pageSize.value !== snapped) {
      pageSize.value = snapped
      return
    }
    if (prevBlock > 0 && prevBlock !== nextBlock) {
      void loadPage(currentPage.value)
    }
  }

  function onGridViewportResize() {
    if (galleryLayoutMode.value !== 'grid' || displayList.value.length === 0) return
    void syncGridPaginationIfNeeded()
  }

  const draggingCards = ref(false)
  let suppressCardClickUntil = 0
  let orderPersistTimer: ReturnType<typeof setTimeout> | null = null
  let pendingOrderPersistIds: string[] | null = null
  let pendingOrderPersistPlacement: MasonryPlacementMap | undefined
  const orderPersisting = ref(false)
  const orderPersistHint = ref('')
  let orderPersistHintTimer: ReturnType<typeof setTimeout> | null = null

  function setOrderPersistHint(msg: string, ms = 1800) {
    orderPersistHint.value = msg
    if (orderPersistHintTimer) clearTimeout(orderPersistHintTimer)
    if (!msg) return
    orderPersistHintTimer = window.setTimeout(() => {
      orderPersistHintTimer = null
      orderPersistHint.value = ''
    }, ms)
  }

  async function commitPendingOrderPersist() {
    const ids = pendingOrderPersistIds
    const placement = pendingOrderPersistPlacement
    pendingOrderPersistIds = null
    pendingOrderPersistPlacement = undefined
    if (!ids || ids.length === 0) return
    orderPersisting.value = true
    try {
      await patchCategoryItemOrder(folderKey.value, ids, placement)
      setOrderPersistHint('排序已保存')
    } catch {
      setOrderPersistHint('排序保存失败，已自动回滚', 2600)
      await loadCategory()
    } finally {
      orderPersisting.value = false
    }
  }

  function handleDragStart() {
    draggingCards.value = true
  }

  function sameIdOrder(a: readonly string[], b: readonly string[]): boolean {
    if (a.length !== b.length) return false
    for (let i = 0; i < a.length; i++) {
      if (a[i] !== b[i]) return false
    }
    return true
  }

  function makeMergedOrderedIdsByPage(orderedPageIds: readonly string[]): string[] {
    const allIds = orderedFilteredRows.value.map((r) => r.id).filter(Boolean)
    if (allIds.length === 0) return []
    const start = (currentPage.value - 1) * pageSize.value
    const end = Math.min(allIds.length, start + orderedPageIds.length)
    if (start >= 0 && start < allIds.length && end > start) {
      allIds.splice(start, end - start, ...orderedPageIds)
    }
    return allIds
  }

  function applyLocalSortOrder(orderedIds: readonly string[]) {
    if (!orderedIds.length) return
    const rank = new Map<string, number>()
    for (let i = 0; i < orderedIds.length; i++) rank.set(orderedIds[i]!, (i + 1) * 10)
    picRows.value = picRows.value.map((row) => {
      const nextSort = rank.get(row.id)
      if (nextSort == null) return row
      return { ...row, sort: nextSort }
    })
  }

  function applyLocalMasonryPlacement(masonryPlacement?: MasonryPlacementMap) {
    if (!masonryPlacement) return
    picRows.value = picRows.value.map((row) => {
      const hit = masonryPlacement[row.id]
      if (!hit) return row
      return { ...row, masonryCol: hit.col, masonryRow: hit.row }
    })
  }

  function hasMasonryPlacementChange(masonryPlacement?: MasonryPlacementMap): boolean {
    if (!masonryPlacement) return false
    if (Object.keys(masonryPlacement).length === 0) return false
    const byId = new Map(picRows.value.map((row) => [row.id, row]))
    for (const [id, pos] of Object.entries(masonryPlacement)) {
      const row = byId.get(id)
      if (!row) continue
      const oldCol = typeof row.masonryCol === 'number' ? row.masonryCol : -1
      const oldRow = typeof row.masonryRow === 'number' ? row.masonryRow : -1
      if (oldCol !== pos.col || oldRow !== pos.row) return true
    }
    return false
  }

  async function persistItemOrderFromPage(
    orderedPageIds: readonly string[],
    masonryPlacement?: MasonryPlacementMap,
  ) {
    const mergedIds = makeMergedOrderedIdsByPage(orderedPageIds)
    if (!mergedIds.length) return
    const currentIds = orderedFilteredRows.value.map((row) => row.id).filter(Boolean)
    const orderChanged = !sameIdOrder(currentIds, mergedIds)
    const placementChanged = hasMasonryPlacementChange(masonryPlacement)
    if (!orderChanged && !placementChanged) return
    applyLocalSortOrder(mergedIds)
    applyLocalMasonryPlacement(masonryPlacement)
    itemSortStore.setServerDefault(folderKey.value, 'sort')
    if (itemSortStore.effectiveMode(folderKey.value) !== 'sort') {
      skipNextSortModeReload = true
      itemSortStore.setOverride(folderKey.value, 'sort')
    }
    pendingOrderPersistIds = mergedIds
    pendingOrderPersistPlacement = masonryPlacement
    if (orderPersistTimer) clearTimeout(orderPersistTimer)
    orderPersistTimer = window.setTimeout(() => {
      orderPersistTimer = null
      void commitPendingOrderPersist()
    }, 220)
  }

  function handleDragEnd() {
    draggingCards.value = false
    suppressCardClickUntil = Date.now() + 160
  }

  async function handleGridDragEnd() {
    handleDragEnd()
    if (!canDragSort.value) return
    await nextTick()
    const pageIds = displayList.value.map((item) => item.id)
    void persistItemOrderFromPage(pageIds)
  }

  function onStageDragEnd() {
    if (galleryLayoutMode.value === 'grid') {
      void handleGridDragEnd()
      return
    }
    handleDragEnd()
  }

  function handleMasonryReorder(payload: {
    orderedIds: string[]
    masonryPlacement: MasonryPlacementMap
  }) {
    const { orderedIds, masonryPlacement } = payload
    if (orderedIds.length === 0) return
    const idSet = new Set(orderedIds)
    const tail = displayList.value.filter((item) => !idSet.has(item.id))
    const byId = new Map(displayList.value.map((item) => [item.id, item]))
    const reordered = orderedIds
      .map((id) => byId.get(id))
      .filter((item): item is GalleryListItem => item !== undefined)
      .map((item) => {
        const hit = masonryPlacement[item.id]
        return hit ? { ...item, masonryCol: hit.col, masonryRow: hit.row } : item
      })
    displayList.value = [...reordered, ...tail]
    if (!canDragSort.value) return
    void persistItemOrderFromPage(
      displayList.value.map((item) => item.id),
      masonryPlacement,
    )
  }

  const goPrevPage = () => {
    if (canPrev.value) void loadPage(currentPage.value - 1, { scrollToWaterfall: true })
  }

  const goNextPage = () => {
    if (canNext.value) void loadPage(currentPage.value + 1, { scrollToWaterfall: true })
  }

  function updateItemMasonryAspectHW(itemId: string, hw: number) {
    if (!Number.isFinite(hw) || hw <= 0) return
    const list = displayList.value
    const idx = list.findIndex((it) => it.id === itemId)
    if (idx < 0) return
    const cur = list[idx]!
    if (typeof cur.masonryAspectHW === 'number' && Math.abs(cur.masonryAspectHW - hw) < 0.02) {
      return
    }
    const next = list.slice()
    next[idx] = { ...cur, masonryAspectHW: hw }
    displayList.value = next
  }

  function getItemStyle(
    item: GalleryListItem,
    batchIndex: number,
  ): Readonly<Record<string, string>> | undefined {
    if (!item.isNew) return undefined
    return NEW_ITEM_STYLE_BY_BATCH_INDEX[batchIndex % NEW_ITEM_STYLE_BY_BATCH_INDEX.length]!
  }

  const viewerVisible = ref(false)
  const viewerImages = shallowRef<string[]>([])
  const viewerLiveVideoUrls = shallowRef<(string | undefined)[]>([])
  const viewerImageItemIds = shallowRef<string[]>([])
  const viewerInitialIndex = ref(0)

  const videoViewerVisible = ref(false)
  const videoViewerUrls = shallowRef<string[]>([])
  const videoViewerItemIds = shallowRef<string[]>([])
  const videoViewerInitialIndex = ref(0)

  const fileDetailOpen = ref(false)
  const fileDetailPayload = ref<GalleryFileDetailPayload | null>(null)
  const fileDetailItemId = ref<string | null>(null)

  const viewerNavIndex = ref(0)
  const viewerNavCurrent = computed(() => viewerNavIndex.value + 1)
  const viewerNavTotal = computed(() => displayList.value.length)
  const viewerSlideDirection = ref<'left' | 'right'>('right')
  const viewerSkipExit = ref(false)

  const viewerToolbarItem = computed(() => {
    if (!viewerVisible.value && !videoViewerVisible.value && !fileDetailOpen.value) {
      return null
    }
    return displayList.value[viewerNavIndex.value] ?? null
  })

  function galleryAdjacentPeekSrc(direction: 'prev' | 'next'): string | undefined {
    if (!viewerVisible.value) return undefined
    const item = displayList.value[viewerNavIndex.value]
    if (!item) return undefined
    const adjacent = findAdjacentGalleryItem(item.id, direction)
    if (!adjacent) return undefined
    if (isGalleryImageItem(adjacent)) {
      const full = adjacent.fullSrc?.trim()
      return full || undefined
    }
    const thumb = adjacent.src?.trim()
    return thumb || undefined
  }

  const viewerAdjacentPrevSrc = computed(() => galleryAdjacentPeekSrc('prev'))
  const viewerAdjacentNextSrc = computed(() => galleryAdjacentPeekSrc('next'))

  const startRect = ref<DOMRect | null>(null)

  function getViewerKind(item: GalleryListItem): 'image' | 'video' | 'file' {
    if (isGalleryVideoItem(item)) return 'video'
    if (isGalleryImageItem(item)) return 'image'
    return 'file'
  }

  function syncViewerNavIndex(item: GalleryListItem) {
    const idx = displayList.value.findIndex((row) => row.id === item.id)
    if (idx >= 0) viewerNavIndex.value = idx
  }

  function closeAllViewers() {
    viewerVisible.value = false
    videoViewerVisible.value = false
    fileDetailOpen.value = false
  }

  function buildFileDetailPayload(item: GalleryListItem): GalleryFileDetailPayload {
    return {
      fullSrc: item.fullSrc,
      originalUrl: item.originalUrl,
      shortUrl: item.shortUrl,
      linkName: item.linkName,
      title: item.title,
      format: item.format,
      mediaKind: item.mediaKind,
      fileSize: item.fileSize,
      uploadedAt: item.uploadedAt,
      updatedAt: item.updatedAt,
      categoryName: item.categoryName || categoryDisplayName.value,
    }
  }

  function openFileDetail(item: GalleryListItem) {
    syncViewerNavIndex(item)
    fileDetailItemId.value = item.id
    viewerVisible.value = false
    videoViewerVisible.value = false
    fileDetailPayload.value = buildFileDetailPayload(item)
    fileDetailOpen.value = true
  }

  function closeFileDetail() {
    fileDetailOpen.value = false
    fileDetailPayload.value = null
    fileDetailItemId.value = null
  }

  function handleFileDetailDownload() {
    const p = fileDetailPayload.value
    if (!p?.fullSrc.trim()) return
    const a = document.createElement('a')
    a.href = p.fullSrc.trim()
    a.download = p.linkName || linkNameFromResourceUrl(p.shortUrl || p.fullSrc)
    a.target = '_blank'
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
  }

  async function handleFileDetailCopyLink() {
    const p = fileDetailPayload.value
    if (!p) return
    const id = fileDetailItemId.value
    const item = id ? displayList.value.find((row) => row.id === id) : null
    const fk = item
      ? resolveGalleryItemContext(item.id, folderKey.value).folderKey
      : folderKey.value
    const requiresKey = folderResourceRequiresViewKeyByFolderKey(categoryNavStore.doc, fk)
    const viewKey = requiresKey ? getGalleryViewGrant(fk) : ''
    const abs = buildShareableResourceUrl(p.originalUrl || p.fullSrc, {
      requiresViewKey: requiresKey,
      viewKey,
    })
    if (!abs) return
    const ok = await copyTextToClipboard(abs)
    if (!ok) {
      messageStore.show('复制失败', 'error')
      return
    }
    if (requiresKey && !viewKey) {
      messageStore.show('链接已复制；加密相册需在链接后加 ?k=查看密码', 'success')
      return
    }
    messageStore.show('链接已复制', 'success')
  }

  function findAdjacentGalleryItem(
    fromItemId: string,
    direction: 'prev' | 'next',
  ): GalleryListItem | null {
    const list = displayList.value
    const idx = list.findIndex((row) => row.id === fromItemId)
    if (idx < 0) return null
    const step = direction === 'next' ? 1 : -1
    for (let i = idx + step; i >= 0 && i < list.length; i += step) {
      return list[i]!
    }
    return null
  }

  function openViewerForGalleryItem(item: GalleryListItem, fromRect: DOMRect | null) {
    syncViewerNavIndex(item)
    if (isGalleryVideoItem(item)) {
      openVideoViewer(item, fromRect)
      return
    }
    if (isGalleryImageItem(item)) {
      openImageViewer(item, fromRect)
      return
    }
    openFileDetail(item)
  }

  function openVideoViewer(item: GalleryListItem, fromRect: DOMRect | null) {
    syncViewerNavIndex(item)
    viewerVisible.value = false
    fileDetailOpen.value = false
    startRect.value = fromRect
    videoViewerUrls.value = [item.fullSrc]
    videoViewerItemIds.value = [item.id]
    videoViewerInitialIndex.value = 0
    videoViewerVisible.value = true
  }

  function openImageViewer(item: GalleryListItem, fromRect: DOMRect | null) {
    syncViewerNavIndex(item)
    videoViewerVisible.value = false
    fileDetailOpen.value = false
    startRect.value = fromRect
    viewerImages.value = [item.fullSrc]
    viewerLiveVideoUrls.value = [item.liveVideoUrl]
    viewerImageItemIds.value = [item.id]
    viewerInitialIndex.value = 0
    viewerVisible.value = true
  }

  function handleViewerNavigate(direction: 'prev' | 'next', anchorItemId?: string) {
    const anchorId = anchorItemId ?? fileDetailItemId.value ?? undefined
    if (!anchorId) return
    const adjacent = findAdjacentGalleryItem(anchorId, direction)
    if (!adjacent) return

    viewerSlideDirection.value = direction === 'next' ? 'right' : 'left'
    const nextKind = getViewerKind(adjacent)

    if (nextKind === 'image' && viewerVisible.value) {
      syncViewerNavIndex(adjacent)
      viewerImages.value = [adjacent.fullSrc]
      viewerLiveVideoUrls.value = [adjacent.liveVideoUrl]
      viewerImageItemIds.value = [adjacent.id]
      return
    }

    if (nextKind === 'video' && videoViewerVisible.value) {
      syncViewerNavIndex(adjacent)
      videoViewerUrls.value = [adjacent.fullSrc]
      videoViewerItemIds.value = [adjacent.id]
      return
    }

    if (nextKind === 'file' && fileDetailOpen.value) {
      openFileDetail(adjacent)
      return
    }

    void switchViewerTo(adjacent)
  }

  async function switchViewerTo(item: GalleryListItem) {
    viewerSkipExit.value = true
    closeAllViewers()
    await nextTick()
    viewerSkipExit.value = false
    openViewerForGalleryItem(item, null)
  }

  function handleCardView(item: GalleryListItem) {
    openViewerForGalleryItem(item, null)
  }

  function handleCardClick(event: MouseEvent, index: number) {
    if (draggingCards.value || Date.now() < suppressCardClickUntil) return
    const list = displayList.value
    const item = list[index]
    if (!item) return

    const targetCard = event.currentTarget as HTMLElement

    if (isGalleryVideoItem(item)) {
      const videoEl = targetCard.querySelector('.card-video') as HTMLVideoElement | null
      const posterEl = targetCard.querySelector('.video-poster') as HTMLImageElement | null
      const fromRect =
        videoEl?.getBoundingClientRect() ??
        posterEl?.getBoundingClientRect() ??
        targetCard.getBoundingClientRect()
      openVideoViewer(item, fromRect)
      return
    }

    if (!isGalleryImageItem(item)) {
      openFileDetail(item)
      return
    }

    const imgElement = targetCard.querySelector('.image') as HTMLImageElement | null
    const fromRect = imgElement
      ? imgElement.getBoundingClientRect()
      : targetCard.getBoundingClientRect()

    openImageViewer(item, fromRect)
  }

  function handleViewerClose() {
    /* ImageViewer 已解锁滚动 */
  }

  function handleVideoViewerClose() {
    /* VideoViewer 已解锁滚动 */
  }

  const editDialogOpen = ref(false)
  const editPayload = ref<GalleryItemEditPayload | null>(null)

  function openItemEdit(item: GalleryListItem) {
    const { folderKey: fk, itemId } = resolveGalleryItemContext(item.id, folderKey.value)
    editPayload.value = {
      folderKey: fk,
      itemId,
      categoryName: item.categoryName || categoryDisplayName.value,
      fullSrc: item.fullSrc,
      thumbSrc: item.src,
      shortUrl: item.shortUrl,
      originalUrl: item.originalUrl || resolveEditorOriginalUrl(item),
      editedUrl: item.editedUrl,
      useEdited: item.useEdited,
      linkName: item.linkName,
      title: item.title,
      tags: item.tags,
      uploadedAt: item.uploadedAt,
      updatedAt: item.updatedAt,
      fileSize: item.fileSize,
      format: item.format,
      mediaKind: item.mediaKind,
      isLivePhoto: item.isLivePhoto,
      liveVideoUrl: item.liveVideoUrl,
    }
    editDialogOpen.value = true
  }

  function closeItemEdit() {
    editDialogOpen.value = false
    editPayload.value = null
  }

  async function handleItemEditSaved() {
    messageStore.show('保存成功', 'success')
    const fk = editPayload.value?.folderKey
    if (isGlobalSearchView.value) {
      await refreshGlobalPool()
      void loadPage(currentPage.value)
      return
    }
    if (fk === folderKey.value) {
      await loadCategory()
    }
  }

  async function handleItemDelete(item: GalleryListItem) {
    const { folderKey: fk, itemId } = resolveGalleryItemContext(item.id, folderKey.value)
    try {
      await deleteCategoryItem(fk, itemId)
      messageStore.show('已移入回收站', 'success')
      if (isGlobalSearchView.value) {
        await refreshGlobalPool()
      } else {
        picRows.value = picRows.value.filter((row) => {
          const ctx = resolveGalleryItemContext(row.id, folderKey.value)
          return ctx.folderKey !== fk || ctx.itemId !== itemId
        })
      }
      const lastPage = Math.max(1, Math.ceil(orderedFilteredRows.value.length / pageSize.value))
      const page = Math.min(currentPage.value, lastPage)
      void loadPage(page, { silent: true })
    } catch {
      messageStore.show('删除失败', 'error')
    }
  }

  const transferDialogOpen = ref(false)
  const transferPayload = ref<{ folderKey: string; itemId: string; itemLabel: string } | null>(
    null,
  )
  const transferSubmitting = ref(false)

  function siblingTransferOptions(forFolderKey: string) {
    const major = categoryNavStore.galleryMajorForFolderKey(forFolderKey)
    if (!major?.subcategories?.length) return []
    const fk = forFolderKey.trim()
    return [...major.subcategories]
      .sort((a, b) => {
        const sa = a.sort ?? 0
        const sb = b.sort ?? 0
        if (sa !== sb) return sa - sb
        return a.id - b.id
      })
      .filter((s) => s.folderKey.trim() && s.folderKey !== fk)
      .map((s) => ({ label: s.name, value: s.folderKey }))
  }

  function canTransferGalleryItem(item: GalleryListItem): boolean {
    if (!showGalleryAdminActions.value) return false
    const { folderKey: fk } = resolveGalleryItemContext(item.id, folderKey.value)
    return siblingTransferOptions(fk).length > 0
  }

  const transferDialogOptions = computed(() => {
    const fk = transferPayload.value?.folderKey?.trim()
    if (!fk) return []
    const major = categoryNavStore.galleryMajorForFolderKey(fk)
    if (!major?.subcategories?.length) return []
    return [...major.subcategories]
      .sort((a, b) => {
        const sa = a.sort ?? 0
        const sb = b.sort ?? 0
        if (sa !== sb) return sa - sb
        return a.id - b.id
      })
      .filter((s) => s.folderKey.trim())
      .map((s) => ({
        label: s.name,
        value: s.folderKey,
        disabled: s.folderKey === fk,
      }))
  })

  function openItemTransfer(item: GalleryListItem) {
    const { folderKey: fk, itemId } = resolveGalleryItemContext(item.id, folderKey.value)
    const options = siblingTransferOptions(fk)
    if (options.length === 0) return
    transferPayload.value = {
      folderKey: fk,
      itemId,
      itemLabel: item.title?.trim() || item.linkName?.trim() || '该文件',
    }
    transferDialogOpen.value = true
  }

  function closeItemTransfer() {
    transferDialogOpen.value = false
    transferPayload.value = null
    transferSubmitting.value = false
  }

  async function handleItemTransferConfirm(targetFolderKey: string) {
    const payload = transferPayload.value
    if (!payload || transferSubmitting.value) return
    transferSubmitting.value = true
    try {
      await transferCategoryItem(payload.folderKey, payload.itemId, targetFolderKey)
      messageStore.show('已转移', 'success')
      closeItemTransfer()
      if (isGlobalSearchView.value) {
        await refreshGlobalPool()
      } else {
        picRows.value = picRows.value.filter((row) => {
          const ctx = resolveGalleryItemContext(row.id, folderKey.value)
          return ctx.folderKey !== payload.folderKey || ctx.itemId !== payload.itemId
        })
      }
      const lastPage = Math.max(1, Math.ceil(orderedFilteredRows.value.length / pageSize.value))
      void loadPage(Math.min(currentPage.value, lastPage), { silent: true })
    } catch {
      transferSubmitting.value = false
      messageStore.show('转移失败', 'error')
    }
  }

  function handleItemDownload(item: GalleryListItem) {
    const url = item.fullSrc.trim()
    if (!url) return
    const a = document.createElement('a')
    a.href = url
    a.download = item.linkName || linkNameFromResourceUrl(item.shortUrl || item.fullSrc)
    a.target = '_blank'
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    a.remove()
  }

  async function handleItemCopyLink(item: GalleryListItem) {
    const { folderKey: fk } = resolveGalleryItemContext(item.id, folderKey.value)
    const requiresKey = folderResourceRequiresViewKeyByFolderKey(categoryNavStore.doc, fk)
    const viewKey = requiresKey ? getGalleryViewGrant(fk) : ''
    const abs = buildShareableResourceUrl(item.originalUrl || item.fullSrc, {
      requiresViewKey: requiresKey,
      viewKey,
    })
    if (!abs) return
    const ok = await copyTextToClipboard(abs)
    if (!ok) {
      messageStore.show('复制失败', 'error')
      return
    }
    if (requiresKey && !viewKey) {
      messageStore.show('链接已复制；加密相册需在链接后加 ?k=查看密码', 'success')
      return
    }
    messageStore.show('链接已复制', 'success')
  }

  const loading = ref(true)
  const loadError = ref('')
  const accessBlocked = ref(false)
  const accessBlockedMode = ref<'auth' | 'password'>('auth')
  const passwordDialogInput = ref('')
  const passwordDialogError = ref('')

  const accessBlockedMessage = computed(() =>
    accessBlockedMode.value === 'password'
      ? '该相册已加密，请输入查看密码。'
      : '该相册需要登录后才能查看。',
  )

  const accessBlockedActionText = computed(() =>
    accessBlockedMode.value === 'password' ? '输入查看密码' : '登录',
  )
  const majorForCurrentFolder = computed(() =>
    categoryNavStore.galleryMajorForFolderKey(folderKey.value),
  )
  const showAddGalleryEmpty = computed(() => {
    if (isGlobalSearchView.value) return false
    if (loading.value || accessBlocked.value) return false
    // 导航未加载完成时不能判定「无画廊」，否则会误显示「添加画廊」
    if (!categoryNavStore.loaded) return false
    const major = majorForCurrentFolder.value
    if (!major) return categoryNavStore.primaryNavLinks.length === 0
    return (major.subcategories?.length ?? 0) === 0
  })
  const showUploadFloatingPanel = computed(
    () =>
      !isGlobalSearchView.value && authStore.authenticated && !accessBlocked.value,
  )

  const {
    setup: uploadPanelSetup,
    teardown: uploadPanelTeardown,
    ...uploadPanel
  } = useGalleryUploadPanel({
    folderKey: () => folderKey.value,
    enabled: () => showUploadFloatingPanel.value,
    onReloadCategory: () => loadCategory(),
  })

  function onLoginClick() {
    authStore.openLoginModal('该相册为私有目录，请先登录。若目录为加密类型，还需输入查看密码。')
  }

  function onBlockedActionClick() {
    if (accessBlockedMode.value === 'password') return
    onLoginClick()
  }

  function onAddGalleryClick() {
    const major = majorForCurrentFolder.value
    if (major) {
      navAddModalStore.openGallery(major.id, major.name)
      return
    }
    navAddModalStore.openPrimary()
  }

  function currentFolderViewGrant(): string {
    return getGalleryViewGrant(folderKey.value)
  }

  async function submitFolderPassword() {
    const pwd = passwordDialogInput.value.trim()
    if (pwd.length < 4) {
      passwordDialogError.value = '查看密码至少 4 位'
      return
    }
    passwordDialogError.value = ''
    try {
      const unlocked = await postCategoryViewUnlock(folderKey.value, pwd)
      const grant = unlocked.grant?.trim() ?? ''
      const expiresAt = unlocked.expiresAt?.trim() ?? ''
      const expiresMs = Date.parse(expiresAt)
      if (!grant || Number.isNaN(expiresMs)) {
        throw new Error('解锁响应无效')
      }
      setGalleryViewGrant(folderKey.value, grant, expiresMs)
      passwordDialogInput.value = ''
      await loadCategory()
    } catch (e: unknown) {
      if (axios.isAxiosError(e) && e.response?.status === 429) {
        const body = e.response.data as { error?: string } | undefined
        messageStore.show(body?.error?.trim() || '查看密码失败次数过多，请稍后再试', 'error')
        return
      }
      if (axios.isAxiosError(e) && e.response?.status === 403) {
        passwordDialogError.value = '查看密码不正确'
        return
      }
      messageStore.show(e instanceof Error ? e.message : '解锁失败', 'error')
    }
  }

  function layoutToStore(L: ApiLayout) {
    if (L.mode === 'masonry' || L.mode === 'grid') {
      masonryLayoutStore.setGalleryLayoutMode(L.mode)
    }
    const c = L.columns
    if (c === 'auto') {
      masonryLayoutStore.setColumnChoice('auto')
    } else if (c === '1' || c === '2' || c === '3' || c === '4' || c === '5' || c === '6') {
      masonryLayoutStore.setColumnChoice(Number(c) as Exclude<MasonryColumnChoice, 'auto'>)
    }
  }

  function storeLayoutPayload(): ApiLayout {
    const mode = galleryLayoutMode.value
    const col = columnChoice.value
    return {
      mode,
      columns: col === 'auto' ? 'auto' : (String(col) as ApiLayout['columns']),
    }
  }

  const layoutHydrating = ref(true)
  let persistTimer: ReturnType<typeof setTimeout> | null = null

  function scheduleLayoutPersist() {
    const key = folderKey.value
    if (!key || layoutHydrating.value || isGlobalSearchView.value) return
    if (persistTimer) clearTimeout(persistTimer)
    persistTimer = window.setTimeout(() => {
      persistTimer = null
      void patchCategoryLayout(key, storeLayoutPayload()).catch(() => {
        /* 静默失败，避免打断浏览 */
      })
    }, 650)
  }

  watch(columnChoice, () => {
    if (galleryLayoutMode.value === 'grid') {
      void syncGridPaginationIfNeeded()
    }
  })

  watch([galleryLayoutMode, columnChoice], () => {
    scheduleLayoutPersist()
  })

  watch(galleryLayoutMode, async (mode) => {
    if (mode === 'grid') {
      await nextTick()
      void syncGridPaginationIfNeeded()
      return
    }
    gridColumnCount.value = 0
    gridRowsPerScreen.value = 0
  })

  async function reloadGlobalSearchResults() {
    if (!isGlobalSearchView.value) return
    syncSearchFiltersFromRoute()
    if (!hasActiveFilter.value) {
      displayList.value = []
      waterfallStageVisible.value = false
      waterfallRendered.value = false
      return
    }
    await refreshGlobalPool()
    void loadPage(1)
  }

  async function loadGlobalSearchView() {
    loading.value = true
    loadError.value = ''
    accessBlocked.value = false
    try {
      if (!categoryNavStore.loaded) {
        await categoryNavStore.fetchFromServer()
      }
      syncSearchFiltersFromRoute()
      if (!hasActiveFilter.value) {
        displayList.value = []
        return
      }
      await refreshGlobalPool()
      void loadPage(1)
    } catch {
      loadError.value = '无法加载搜索结果，请稍后再试。'
      globalPool.value = []
      displayList.value = []
    } finally {
      loading.value = false
    }
  }

  async function loadCategory() {
    if (isGlobalSearchView.value) {
      await loadGlobalSearchView()
      return
    }
    loading.value = true
    loadError.value = ''
    accessBlocked.value = false
    layoutHydrating.value = true
    const grant = currentFolderViewGrant()
    try {
      if (!categoryNavStore.loaded) {
        await categoryNavStore.fetchFromServer()
      }
      if (isMajorPlaceholderFolderKey(folderKey.value)) {
        picRows.value = []
        categoryDisplayName.value = ''
        return
      }
      const detail = await fetchCategoryDetail(
        folderKey.value,
        grant ? { grant } : undefined,
      )
      layoutToStore(detail.layout)
      itemSortStore.setServerDefault(folderKey.value, detail.itemSortBy)
      await nextTick()
      categoryDisplayName.value = detail.name
      picRows.value = detail.items.map((it) => ({
        id: it.id,
        url: appendCacheBust(authStore.appendAccessToResourceUrl(it.url), it.updatedAt),
        shortUrl: it.shortUrl ? authStore.appendAccessToResourceUrl(it.shortUrl) : undefined,
        linkName: it.linkName,
        originalUrl: it.originalUrl ? authStore.appendAccessToResourceUrl(it.originalUrl) : undefined,
        editedUrl: it.editedUrl ? authStore.appendAccessToResourceUrl(it.editedUrl) : undefined,
        useEdited: it.useEdited,
        format: it.format,
        mediaKind: it.mediaKind,
        isLivePhoto: it.isLivePhoto,
        liveVideoUrl: it.liveVideoUrl
          ? authStore.appendAccessToResourceUrl(it.liveVideoUrl)
          : undefined,
        categoryName: detail.name,
        thumbnailUrl: it.thumbnailUrl
          ? appendCacheBust(authStore.appendAccessToResourceUrl(it.thumbnailUrl), it.updatedAt)
          : undefined,
        title: it.title,
        tags: it.tags,
        uploadedAt: it.uploadedAt,
        updatedAt: it.updatedAt,
        sort: it.sort,
        masonryCol: it.masonryCol,
        masonryRow: it.masonryRow,
        fileSize: it.fileSize,
      }))
      void loadPage(1)
    } catch (e) {
      if (axios.isAxiosError(e) && e.response?.status === 403) {
        accessBlocked.value = true
        const body = e.response.data as { requiresPassword?: boolean } | undefined
        accessBlockedMode.value = body?.requiresPassword ? 'password' : 'auth'
        if (body?.requiresPassword && grant) {
          clearGalleryViewGrant(folderKey.value)
        }
        picRows.value = []
      } else {
        loadError.value = '无法加载该分类，请确认后端已启动且数据目录正确。'
        picRows.value = []
      }
    } finally {
      loading.value = false
      await nextTick()
      layoutHydrating.value = false
    }
  }

  watch(folderKey, () => {
    if (isGlobalSearchView.value) void loadGlobalSearchView()
    else void loadCategory()
  })

  watch(
    () => galleryItemsSyncStore.reloadNonce,
    async () => {
      const needCategory = galleryItemsSyncStore.takeFolderDirty(folderKey.value)
      const needGlobal = galleryItemsSyncStore.takeGlobalPoolDirty()
      if (needCategory) {
        await loadCategory()
      }
      if (needGlobal && isGlobalSearchView.value) {
        await refreshGlobalPool()
        void loadPage(currentPage.value, { silent: true })
      }
    },
  )

  watch(
    () => [authStore.authConfigured, authStore.authenticated] as const,
    async ([cfg, ok]) => {
      if (!cfg || !ok) return
      if (accessBlocked.value) {
        accessBlocked.value = false
        await loadCategory()
        return
      }
      if (isGlobalSearchView.value) {
        await refreshGlobalPool()
      }
    },
  )

  onMounted(() => {
    window.addEventListener('pointerdown', onPageSizeMenuPointerDown)
    window.addEventListener('resize', onGridViewportResize)
    uploadPanelSetup()
    if (isGlobalSearchView.value) void loadGlobalSearchView()
    else void loadCategory()
  })

  onUnmounted(() => {
    window.removeEventListener('pointerdown', onPageSizeMenuPointerDown)
    window.removeEventListener('resize', onGridViewportResize)
    uploadPanelTeardown()
    clearClearIsNewTimer()
    loadPageGeneration++
    globalPoolGen++
    if (persistTimer) clearTimeout(persistTimer)
    if (orderPersistTimer) {
      clearTimeout(orderPersistTimer)
      orderPersistTimer = null
    }
    void commitPendingOrderPersist()
    if (orderPersistHintTimer) clearTimeout(orderPersistHintTimer)
  })

  const showGalleryContent = computed(() => {
    if (isGlobalSearchView.value) {
      return (
        !loading.value &&
        !loadError.value &&
        !globalPoolLoading.value &&
        searchSourceRows.value.length > 0 &&
        filteredRows.value.length > 0
      )
    }
    return (
      !loading.value &&
      !accessBlocked.value &&
      !showAddGalleryEmpty.value &&
      !loadError.value &&
      searchSourceRows.value.length > 0 &&
      filteredRows.value.length > 0
    )
  })

  const pageLoading = computed(() =>
    isGlobalSearchView.value ? loading.value || globalPoolLoading.value : loading.value,
  )

  const navLoadError = computed(() => categoryNavStore.loadError ?? '')

  return {
    galleryStageRef,
    showGalleryContent,
    pageLoading,
    navLoadError,
    loading,
    loadError,
    accessBlocked,
    accessBlockedMode,
    accessBlockedMessage,
    accessBlockedActionText,
    onBlockedActionClick,
    showAddGalleryEmpty,
    onAddGalleryClick,
    isGlobalSearchView,
    searchNeedsCriteria,
    searchScope,
    globalPoolLoading,
    hasActiveFilter,
    searchSourceRows,
    filteredRows,
    waterfallStageVisible,
    draggingCards,
    galleryLayoutMode,
    displayList,
    canDragSort,
    effectiveItemSort,
    showGalleryAdminActions,
    getItemStyle,
    handleGalleryLayout,
    handleMasonryReorder,
    handleDragStart,
    onStageDragEnd,
    handleCardClick,
    handleItemDelete,
    openItemTransfer,
    transferDialogOpen,
    transferPayload,
    transferDialogOptions,
    transferSubmitting,
    closeItemTransfer,
    handleItemTransferConfirm,
    openItemEdit,
    handleCardView,
    handleItemDownload,
    handleItemCopyLink,
    updateItemMasonryAspectHW,
    showPagination,
    orderPersisting,
    orderPersistHint,
    dragSortEditEnabled,
    pageSizeMenuOpen,
    pageSizeSelectValue,
    pageSize,
    pageSizeSelectOptions,
    togglePageSizeMenu,
    selectPageSize,
    currentPage,
    totalPages,
    orderedFilteredRows,
    goPrevPage,
    goNextPage,
    showUploadFloatingPanel,
    viewerVisible,
    viewerImages,
    viewerLiveVideoUrls,
    viewerImageItemIds,
    viewerInitialIndex,
    videoViewerVisible,
    videoViewerUrls,
    videoViewerItemIds,
    videoViewerInitialIndex,
    fileDetailOpen,
    fileDetailPayload,
    viewerNavCurrent,
    viewerNavTotal,
    viewerToolbarItem,
    viewerAdjacentPrevSrc,
    viewerAdjacentNextSrc,
    canTransferGalleryItem,
    viewerSlideDirection,
    viewerSkipExit,
    startRect,
    handleViewerNavigate,
    handleViewerClose,
    handleVideoViewerClose,
    editDialogOpen,
    editPayload,
    closeItemEdit,
    handleItemEditSaved,
    closeFileDetail,
    handleFileDetailDownload,
    handleFileDetailCopyLink,
    passwordDialogInput,
    passwordDialogError,
    submitFolderPassword,
    ...uploadPanel,
  }
}
