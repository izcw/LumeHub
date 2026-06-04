import { defineStore, storeToRefs } from 'pinia'
import { computed, ref, shallowRef } from 'vue'
import { fetchCategories } from '@/api/gallery'
import type { ApiCategoriesDoc, ApiCategoryGroup, ApiSubcategory } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import {
  folderContentRequiresLogin,
  folderContentRequiresPassword,
  majorNavEntryVisible,
  majorNavVisibleToGuest,
  navEntryVisible,
  navVisibleToGuest,
  folderNavEntryVisible,
} from '@/utils/galleryAccess'
import {
  majorPlaceholderFolderKey,
  parseMajorPlaceholderFolderKey,
} from '@/utils/categoryFolderKey'
import {
  GALLERY_SEARCH_FOLDER_KEY,
  isGallerySearchFolderKey,
} from '@/utils/gallerySearchFolder'
import { useGallerySearchStore } from '@/stores/gallerySearch'

const FALLBACK_HOME = 'jilu'

/** 一级主导航：未登录时按访问策略隐藏私密/加密隐藏入口；登录后展示完整列表；showInvisibleHint 表示访客看不到该入口 */
export interface PrimaryNavLink {
  id: number
  name: string
  /** 路由路径：首页目录为 `/`，其余为 `/gallery/{folderKey}` */
  link: string
  /** 与后端 `data/resource/{folderKey}` 一致 */
  folderKey: string
  /** 为 true 时在链接右侧显示 invisible 图标（表示未登录时该入口不会出现） */
  showInvisibleHint: boolean
}

/** 画廊顶部分类条（二级）：未登录也显示链接；需登录查看内容时右侧显示锁 */
export interface GalleryClassificationLink {
  id: number
  name: string
  folderKey: string
  link: string
  public: boolean
  sort: string
  showLock: boolean
  /** 系统虚拟画廊（如搜索结果），不可拖拽排序、不可删除 */
  isSystem?: boolean
}

const GALLERY_SEARCH_STRIP_ID = -1

/** 分类导航：按 sort 数值升序（小的在左）；未写 sort 的排最后，再按 id */
function categoryNavOrderKey(s: number | undefined): number {
  if (s == null || Number.isNaN(s)) return Number.POSITIVE_INFINITY
  return s
}

function sortMajorCategories(groups: ApiCategoryGroup[]): ApiCategoryGroup[] {
  return [...groups].sort((a, b) => {
    const d = categoryNavOrderKey(a.sort) - categoryNavOrderKey(b.sort)
    if (d !== 0) return d
    return a.id - b.id
  })
}

function sortSubcategories(subs: ApiSubcategory[]): ApiSubcategory[] {
  return [...subs].sort((a, b) => {
    const d = categoryNavOrderKey(a.sort) - categoryNavOrderKey(b.sort)
    if (d !== 0) return d
    return a.id - b.id
  })
}

function majorContainingFolder(doc: ApiCategoriesDoc | null, folderKey: string): ApiCategoryGroup | undefined {
  if (!doc?.categories) return undefined
  const fk = folderKey.trim()
  const placeholderMajorId = parseMajorPlaceholderFolderKey(fk)
  if (placeholderMajorId != null) {
    return sortMajorCategories(doc.categories).find((m) => m.id === placeholderMajorId)
  }
  const majors = sortMajorCategories(doc.categories)
  for (const m of majors) {
    if (sortSubcategories(m.subcategories ?? []).some((s) => s.folderKey === fk)) {
      return m
    }
  }
  return undefined
}

function majorForSubfolder(doc: ApiCategoriesDoc | null, folderKey: string): ApiCategoryGroup | undefined {
  if (!doc?.categories) return undefined
  for (const m of doc.categories) {
    if (m.subcategories?.some((s) => s.folderKey === folderKey)) return m
  }
  return undefined
}

export const useCategoryNavStore = defineStore('categoryNav', () => {
  const authStore = useAuthStore()
  const { canAccessPrivate } = storeToRefs(authStore)

  const doc = shallowRef<ApiCategoriesDoc | null>(null)
  const loadError = ref<string | null>(null)
  const loaded = ref(false)
  let inflight: Promise<void> | null = null

  async function fetchFromServer() {
    if (loaded.value) return
    if (inflight) return inflight
    inflight = (async () => {
      try {
        doc.value = await fetchCategories()
        loadError.value = null
        loaded.value = true
      } catch {
        loadError.value = '导航数据加载失败'
      } finally {
        inflight = null
      }
    })()
    return inflight
  }

  /** 强制从服务端重新拉取分类导航（结构变更后使用） */
  async function reloadFromServer() {
    if (inflight) await inflight
    try {
      doc.value = await fetchCategories()
      loadError.value = null
      loaded.value = true
    } catch {
      loadError.value = '导航数据加载失败'
    }
  }

  const homeFolderKey = computed(() => {
    const d = doc.value
    if (d?.categories?.length) {
      const authed = canAccessPrivate.value
      const majors = sortMajorCategories(d.categories)
      for (const m of majors) {
        const subs = sortSubcategories(m.subcategories ?? [])
        const first = subs.find(
          (s) => navEntryVisible(m, s, authed) && !isGallerySearchFolderKey(s.folderKey),
        )
        if (first) return first.folderKey
        if (majorNavEntryVisible(m, authed)) return majorPlaceholderFolderKey(m.id)
      }
    }
    return FALLBACK_HOME
  })

  function linkForFolder(folderKey: string): string {
    return folderKey === homeFolderKey.value ? '/' : `/gallery/${folderKey}`
  }

  /** 全部二级（跨大分类），顺序：先按大分类 sort，再按组内二级 sort */
  const allLeafSubcategories = computed<ApiSubcategory[]>(() => {
    const d = doc.value
    if (!d?.categories?.length) return []
    const out: ApiSubcategory[] = []
    for (const m of sortMajorCategories(d.categories)) {
      for (const s of sortSubcategories(m.subcategories ?? [])) {
        if (!isGallerySearchFolderKey(s.folderKey)) out.push(s)
      }
    }
    return out
  })

  /** 当前登录态下可在导航与搜索「全部」中出现的二级 */
  const visibleLeafSubcategories = computed<ApiSubcategory[]>(() => {
    const d = doc.value
    const authed = canAccessPrivate.value
    return allLeafSubcategories.value.filter((sub) => {
      const major = majorForSubfolder(d, sub.folderKey)
      if (!major) return true
      return navEntryVisible(major, sub, authed)
    })
  })

  /** 主导航：每个大分类链到其下第一个「当前可见」的二级 */
  const primaryNavLinks = computed<PrimaryNavLink[]>(() => {
    const d = doc.value
    if (!d?.categories?.length) return []
    const majors = sortMajorCategories(d.categories)
    const links: PrimaryNavLink[] = []
    const authed = canAccessPrivate.value
    for (const m of majors) {
      const subs = sortSubcategories(m.subcategories ?? [])
      const first = subs.find(
        (s) => navEntryVisible(m, s, authed) && !isGallerySearchFolderKey(s.folderKey),
      )
      if (first) {
        links.push({
          id: m.id,
          name: m.name,
          folderKey: first.folderKey,
          link: linkForFolder(first.folderKey),
          showInvisibleHint: authed && !navVisibleToGuest(m, first),
        })
        continue
      }
      if (!majorNavEntryVisible(m, authed)) continue
      const placeholderKey = majorPlaceholderFolderKey(m.id)
      links.push({
        id: m.id,
        name: m.name,
        folderKey: placeholderKey,
        link: linkForFolder(placeholderKey),
        showInvisibleHint: authed && !majorNavVisibleToGuest(m),
      })
    }
    return links
  })

  /** 当前目录所在大分类（用于画廊条排序键等） */
  function galleryMajorForFolderKey(activeFolderKey: string): ApiCategoryGroup | undefined {
    const d = doc.value
    if (!d?.categories?.length) return undefined
    const fk = activeFolderKey?.trim() || homeFolderKey.value
    return majorContainingFolder(d, fk)
  }

  function appendSearchGalleryStripLink(
    links: GalleryClassificationLink[],
  ): GalleryClassificationLink[] {
    if (links.some((l) => isGallerySearchFolderKey(l.folderKey))) return links
    return [
      ...links,
      {
        id: GALLERY_SEARCH_STRIP_ID,
        name: '搜索结果',
        folderKey: GALLERY_SEARCH_FOLDER_KEY,
        link: linkForFolder(GALLERY_SEARCH_FOLDER_KEY),
        public: true,
        sort: String(links.length + 1),
        showLock: false,
        isSystem: true,
      },
    ]
  }

  /** 顶部画廊条：当前大分类下的可见二级；「搜索结果」仅在浏览该页时出现 */
  function galleryStripLinksForFolder(activeFolderKey: string): GalleryClassificationLink[] {
    const d = doc.value
    if (!d?.categories?.length) return []
    const routeFk = activeFolderKey?.trim() || homeFolderKey.value
    let contextFk = routeFk
    if (isGallerySearchFolderKey(routeFk)) {
      const returnFk = useGallerySearchStore().searchReturnFolderKey.trim()
      contextFk = returnFk || homeFolderKey.value
    }
    const major = majorContainingFolder(d, contextFk)
    if (!major) return []
    const subs = sortSubcategories(major.subcategories ?? []).filter(
      (s) => !isGallerySearchFolderKey(s.folderKey),
    )
    const authed = canAccessPrivate.value
    const links = subs
      .filter((s) => navEntryVisible(major, s, authed))
      .map((s, i) => ({
        id: s.id,
        name: s.name,
        folderKey: s.folderKey,
        link: linkForFolder(s.folderKey),
        public: s.public !== false,
        sort: String(s.sort ?? i + 1),
        showLock: folderContentRequiresLogin(major, s) || folderContentRequiresPassword(major, s),
      }))
    if (isGallerySearchFolderKey(routeFk)) {
      return appendSearchGalleryStripLink(links)
    }
    return links
  }

  function replaceDoc(next: ApiCategoriesDoc) {
    doc.value = next
    loaded.value = true
    loadError.value = null
  }

  return {
    doc,
    loadError,
    loaded,
    homeFolderKey,
    primaryNavLinks,
    allLeafSubcategories,
    visibleLeafSubcategories,
    galleryStripLinksForFolder,
    galleryMajorForFolderKey,
    fetchFromServer,
    reloadFromServer,
    replaceDoc,
  }
})
