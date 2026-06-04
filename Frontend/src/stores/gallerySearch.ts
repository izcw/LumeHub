import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  GALLERY_SEARCH_FOLDER_KEY,
  isGallerySearchFolderKey,
  isReservedGalleryFolderKey,
} from '@/utils/gallerySearchFolder'

export { GALLERY_SEARCH_FOLDER_KEY, isGallerySearchFolderKey, isReservedGalleryFolderKey }

/**
 * 大类：选中后等价于允许该组内所有扩展名（与下方「后缀」取并集）
 */
export const GALLERY_CATEGORY_OPTIONS = [
  {
    id: 'image',
    label: '图片',
    extensions: ['jpg', 'jpeg', 'png', 'webp', 'gif', 'svg', 'bmp', 'avif', 'ico', 'heic'],
  },
  {
    id: 'video',
    label: '视频',
    extensions: ['mp4', 'webm', 'mov', 'mkv', 'avi', 'm4v', 'wmv', 'flv'],
  },
  {
    id: 'audio',
    label: '音频',
    extensions: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a', 'wma'],
  },
  {
    id: 'archive',
    label: '压缩包',
    extensions: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz'],
  },
  {
    id: 'document',
    label: '文档',
    extensions: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'rtf', 'csv'],
  },
] as const

/** 编辑页标签建议（搜索筛选项改由资源动态生成） */
export const GALLERY_TAG_OPTIONS = [
  { id: 'images', label: 'images' },
  { id: 'assets', label: 'assets' },
  { id: '人物', label: '人物' },
  { id: '风景', label: '风景' },
  { id: '静物', label: '静物' },
  { id: '夜景', label: '夜景' },
] as const

export type GalleryFilterOption = { id: string; label: string }

export type GalleryFilterOptions = {
  categories: GalleryFilterOption[]
  extensions: GalleryFilterOption[]
  tags: GalleryFilterOption[]
}

const EMPTY_FILTER_OPTIONS: GalleryFilterOptions = {
  categories: [],
  extensions: [],
  tags: [],
}

const CATEGORY_LABEL_MAP = new Map<string, string>(
  GALLERY_CATEGORY_OPTIONS.map((c) => [c.id, c.label]),
)
CATEGORY_LABEL_MAP.set('other', '其他')

const CATEGORY_ORDER = ['image', 'video', 'audio', 'archive', 'document', 'other'] as const

/** 远程条目：在 URL 之外同时匹配标题与标签；时间与 sort 用于列表排序 */
export type GalleryRemoteRow = {
  id: string
  url: string
  /** 基于 linkName 的较短资源 URL */
  shortUrl?: string
  /** items.json 中的 linkName（含扩展名，如 123.jpg） */
  linkName?: string
  thumbnailUrl?: string
  title?: string
  tags?: readonly string[]
  /** 主资源文件大小（字节），后端 items.json fileSize */
  fileSize?: number
  /** ISO8601，上传时间 */
  uploadedAt?: string
  /** ISO8601，更新时间 */
  updatedAt?: string
  /** 自定义排序权重，与后端 items.sort 一致 */
  sort?: number
  /** 瀑布流自定义列位置（仅 sort 模式使用） */
  masonryCol?: number
  masonryRow?: number
  originalUrl?: string
  editedUrl?: string
  useEdited?: boolean
  format?: string
  mediaKind?: string
  categoryName?: string
  isLivePhoto?: boolean
  liveVideoUrl?: string
}

function normalizeExtFromFilename(base: string): string {
  const m = base.split('?')[0]!.match(/\.([a-z0-9]+)$/i)
  let ext = (m?.[1] || '').toLowerCase()
  if (ext === 'jpeg') ext = 'jpg'
  return ext
}

const EXT_TO_CATEGORY = new Map<string, string>()
for (const cat of GALLERY_CATEGORY_OPTIONS) {
  for (const e of cat.extensions) {
    EXT_TO_CATEGORY.set(e.toLowerCase(), cat.id)
    if (e === 'jpeg') EXT_TO_CATEGORY.set('jpg', cat.id)
  }
}

function mediaKindFromRow(row: GalleryRemoteRow): string {
  const mk = row.mediaKind?.trim().toLowerCase()
  if (mk && mk !== 'other') return mk
  const path = (row.linkName || row.url).split('?')[0]!
  const base = path.split(/[/\\]/).pop() || path
  const ext = normalizeExtFromFilename(base)
  if (!ext) return 'other'
  return EXT_TO_CATEGORY.get(ext) ?? 'other'
}

function filenameFromRow(row: GalleryRemoteRow): string {
  const path = (row.linkName || row.url).split('?')[0]!
  return path.split(/[/\\]/).pop() || path
}

/** 从画廊条目汇总当前可用的分类 / 后缀 / 标签筛选项 */
export function buildFilterOptionsFromRows(rows: readonly GalleryRemoteRow[]): GalleryFilterOptions {
  const categoryIds = new Set<string>()
  const extIds = new Set<string>()
  const tagSet = new Set<string>()

  for (const row of rows) {
    categoryIds.add(mediaKindFromRow(row))
    const ext = normalizeExtFromFilename(filenameFromRow(row))
    if (ext) extIds.add(ext)
    for (const tag of row.tags || []) {
      const t = tag.trim()
      if (t) tagSet.add(t)
    }
  }

  const categories = [...categoryIds]
    .sort((a, b) => {
      const ai = CATEGORY_ORDER.indexOf(a as (typeof CATEGORY_ORDER)[number])
      const bi = CATEGORY_ORDER.indexOf(b as (typeof CATEGORY_ORDER)[number])
      const ao = ai >= 0 ? ai : CATEGORY_ORDER.length
      const bo = bi >= 0 ? bi : CATEGORY_ORDER.length
      return ao - bo
    })
    .map((id) => ({ id, label: CATEGORY_LABEL_MAP.get(id) || id }))

  const extensions = [...extIds]
    .sort((a, b) => a.localeCompare(b))
    .map((id) => ({ id, label: id }))

  const tags = [...tagSet]
    .sort((a, b) => a.localeCompare(b, 'zh-CN'))
    .map((id) => ({ id, label: id }))

  return { categories, extensions, tags }
}

export type GallerySearchScope = 'current' | 'all'

/** @deprecated 使用 GALLERY_SEARCH_FOLDER_KEY */
export const GALLERY_GLOBAL_SEARCH_FOLDER_KEY = GALLERY_SEARCH_FOLDER_KEY

export type GallerySearchFilters = {
  query: string
  categories: string[]
  extensions: string[]
  tags: string[]
  /** 当前分类页内搜索 | 全部分类合并搜索 */
  searchScope?: GallerySearchScope
}


function rowMatchesTypeFilters(row: GalleryRemoteRow, filters: GallerySearchFilters): boolean {
  const hasCategoryFilter = filters.categories.length > 0
  const hasExtensionFilter = filters.extensions.length > 0
  if (!hasCategoryFilter && !hasExtensionFilter) return true

  const kind = mediaKindFromRow(row)
  const catMatch = hasCategoryFilter && filters.categories.includes(kind)

  const ext = normalizeExtFromFilename(filenameFromRow(row))
  const extMatch =
    hasExtensionFilter &&
    Boolean(ext) &&
    filters.extensions.some((e) => e.toLowerCase() === ext)

  return catMatch || extMatch
}

function urlMediaKind(url: string): string {
  const path = url.split('?')[0]!
  const base = path.split(/[/\\]/).pop() || path
  const ext = normalizeExtFromFilename(base)
  if (!ext) return 'other'
  return EXT_TO_CATEGORY.get(ext) ?? 'other'
}

function urlMatchesTypeFilters(src: string, filters: GallerySearchFilters): boolean {
  const hasCategoryFilter = filters.categories.length > 0
  const hasExtensionFilter = filters.extensions.length > 0
  if (!hasCategoryFilter && !hasExtensionFilter) return true

  const kind = urlMediaKind(src)
  const catMatch = hasCategoryFilter && filters.categories.includes(kind)

  const path = src.split('?')[0]!
  const base = path.split(/[/\\]/).pop() || path
  const ext = normalizeExtFromFilename(base)
  const extMatch =
    hasExtensionFilter &&
    Boolean(ext) &&
    filters.extensions.some((e) => e.toLowerCase() === ext)

  return catMatch || extMatch
}

function haystackForRow(row: GalleryRemoteRow): string {
  const t = (row.title || '').toLowerCase()
  const tags = (row.tags || []).join(' ').toLowerCase()
  return `${row.url.toLowerCase()}\n${t}\n${tags}`
}

/** 根据关键词、扩展名（分类∪后缀）、标签筛选远程画廊条目 */
export function filterGalleryRemoteRows<T extends GalleryRemoteRow>(
  rows: readonly T[],
  filters: GallerySearchFilters,
): T[] {
  const tokens = filters.query
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)
  const tagNeedles = filters.tags.map((t) => t.toLowerCase())

  return rows.filter((row) => {
    const hay = haystackForRow(row)
    if (tokens.length && !tokens.every((t) => hay.includes(t))) return false
    if (!rowMatchesTypeFilters(row, filters)) return false
    if (tagNeedles.length) {
      if (!tagNeedles.some((tag) => hay.includes(tag))) return false
    }
    return true
  })
}

/** 根据关键词、扩展名（分类∪后缀）、标签筛选图片 URL 列表 */
export function filterGalleryImageSources(
  sources: readonly string[],
  filters: GallerySearchFilters,
): string[] {
  const tokens = filters.query
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)
  const tagNeedles = filters.tags.map((t) => t.toLowerCase())

  return sources.filter((src) => {
    const path = src.toLowerCase()
    if (tokens.length && !tokens.every((t) => path.includes(t))) return false
    if (!urlMatchesTypeFilters(src, filters)) return false
    if (tagNeedles.length) {
      if (!tagNeedles.some((tag) => path.includes(tag))) return false
    }
    return true
  })
}

export const useGallerySearchStore = defineStore('gallerySearch', () => {
  const query = ref('')
  const selectedCategories = ref<string[]>([])
  const selectedExtensions = ref<string[]>([])
  const selectedTags = ref<string[]>([])
  const searchScope = ref<GallerySearchScope>('current')
  const filterOptionsCurrent = ref<GalleryFilterOptions>({ ...EMPTY_FILTER_OPTIONS })
  const filterOptionsAll = ref<GalleryFilterOptions>({ ...EMPTY_FILTER_OPTIONS })
  /** 由画廊页监听：拉取全部分类条目以生成「全部」范围的筛选项 */
  const globalPoolRefreshNonce = ref(0)
  /** 从某分类/首页进入「全部」搜索前所在 folderKey，用于返回 */
  const searchReturnFolderKey = ref('')

  function setSearchReturnFolderKey(folderKey: string) {
    searchReturnFolderKey.value = folderKey.trim()
  }

  function hasFilterOptions(options: GalleryFilterOptions): boolean {
    return (
      options.categories.length > 0 ||
      options.extensions.length > 0 ||
      options.tags.length > 0
    )
  }

  function requestGlobalPoolRefresh() {
    globalPoolRefreshNonce.value += 1
  }

  function pruneSelectedFilters(options: GalleryFilterOptions) {
    const catIds = new Set(options.categories.map((o) => o.id))
    const extIds = new Set(options.extensions.map((o) => o.id))
    const tagIds = new Set(options.tags.map((o) => o.id))
    selectedCategories.value = selectedCategories.value.filter((id) => catIds.has(id))
    selectedExtensions.value = selectedExtensions.value.filter((id) => extIds.has(id))
    selectedTags.value = selectedTags.value.filter((id) => tagIds.has(id))
  }

  function setFilterOptions(scope: GallerySearchScope, options: GalleryFilterOptions) {
    const next = {
      categories: [...options.categories],
      extensions: [...options.extensions],
      tags: [...options.tags],
    }
    if (scope === 'current') {
      filterOptionsCurrent.value = next
    } else {
      filterOptionsAll.value = next
    }
    if (searchScope.value === scope) {
      pruneSelectedFilters(next)
    }
  }

  function filterOptionsForScope(scope: GallerySearchScope): GalleryFilterOptions {
    if (scope === 'all') {
      const all = filterOptionsAll.value
      if (hasFilterOptions(all)) return all
      return filterOptionsCurrent.value
    }
    return filterOptionsCurrent.value
  }

  function setFilters(next: GallerySearchFilters) {
    query.value = next.query
    selectedCategories.value = [...next.categories]
    selectedExtensions.value = [...next.extensions]
    selectedTags.value = [...next.tags]
    if (next.searchScope !== undefined) {
      searchScope.value = next.searchScope
    }
    // 全库筛选项未拉取完成时勿按「当前分类」选项裁剪，避免误删 video 等大类
    if (searchScope.value === 'all' && !hasFilterOptions(filterOptionsAll.value)) {
      return
    }
    pruneSelectedFilters(filterOptionsForScope(searchScope.value))
  }

  function resetFilters() {
    query.value = ''
    selectedCategories.value = []
    selectedExtensions.value = []
    selectedTags.value = []
    searchScope.value = 'current'
  }

  return {
    query,
    selectedCategories,
    selectedExtensions,
    selectedTags,
    searchScope,
    filterOptionsCurrent,
    filterOptionsAll,
    globalPoolRefreshNonce,
    searchReturnFolderKey,
    setSearchReturnFolderKey,
    requestGlobalPoolRefresh,
    setFilterOptions,
    filterOptionsForScope,
    setFilters,
    resetFilters,
  }
})
