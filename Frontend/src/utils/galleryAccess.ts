import type { ApiCategoryGroup, ApiSubcategory } from '@/api/types'
import { parseMajorPlaceholderFolderKey } from '@/utils/categoryFolderKey'
import { isGallerySearchFolderKey } from '@/utils/gallerySearchFolder'

/** 目录访问策略（由 public + encrypted 组合编码，与账户中心选项一致） */
export type FolderAccessPolicy = 'open' | 'encrypted_public' | 'private' | 'encrypted_hidden'

/**
 * 访问策略语义：
 * - 公开可见：菜单与链接均可直接访问
 * - 加密公开：菜单可见；直链需 ?k= 密钥
 * - 私密（隐藏菜单）：菜单登录后可见；直链可直接访问
 * - 加密隐藏：菜单登录后可见；直链需 ?k= 密钥
 */
export const FOLDER_ACCESS_POLICY_OPTIONS: readonly {
  value: FolderAccessPolicy
  label: string
}[] = [
  { value: 'open', label: '公开可见' },
  { value: 'encrypted_public', label: '加密公开（链接需密钥）' },
  { value: 'private', label: '隐藏菜单' },
  { value: 'encrypted_hidden', label: '加密隐藏菜单（链接需密钥）' },
]

export function folderAccessPolicyLabel(policy: FolderAccessPolicy): string {
  return FOLDER_ACCESS_POLICY_OPTIONS.find((item) => item.value === policy)?.label ?? policy
}

export function folderAccessPolicyFrom(x: {
  public?: boolean
  encrypted?: boolean
  encryptedPasswordHash?: string
}): FolderAccessPolicy {
  const pub = x.public !== false
  const enc = !!x.encrypted || !!x.encryptedPasswordHash?.trim()
  if (enc && pub) return 'encrypted_public'
  if (enc && !pub) return 'encrypted_hidden'
  if (!enc && !pub) return 'private'
  return 'open'
}

export function applyFolderAccessPolicy(x: { public?: boolean; encrypted?: boolean }, p: FolderAccessPolicy) {
  switch (p) {
    case 'open':
      delete x.public
      x.encrypted = false
      break
    case 'private':
      x.public = false
      x.encrypted = false
      break
    case 'encrypted_public':
      delete x.public
      x.encrypted = true
      break
    case 'encrypted_hidden':
      x.public = false
      x.encrypted = true
      break
  }
}

/** 未登录访客是否能在主导航中看到「尚无二级画廊」的大分类 */
export function majorNavVisibleToGuest(major: ApiCategoryGroup): boolean {
  return major.public !== false
}

/** 当前登录态下主导航是否展示尚无二级画廊的大分类 */
export function majorNavEntryVisible(major: ApiCategoryGroup, authenticated: boolean): boolean {
  if (authenticated) return true
  return majorNavVisibleToGuest(major)
}

/** 未登录访客是否能在主导航 / 画廊条中看到该二级（私密、加密隐藏不展示；加密公开仍可见） */
export function navVisibleToGuest(major: ApiCategoryGroup, sub: ApiSubcategory): boolean {
  if (major.public === false) return false
  if (sub.public === false) return false
  return true
}

/** 浏览画廊页 / API 列表是否需要登录（私密、加密隐藏） */
export function folderGalleryBrowseRequiresLogin(
  major: ApiCategoryGroup,
  sub: ApiSubcategory,
): boolean {
  if (major.public === false) return true
  if (sub.public === false) return true
  return false
}

/** @deprecated 请使用 folderGalleryBrowseRequiresLogin */
export function folderContentRequiresLogin(major: ApiCategoryGroup, sub: ApiSubcategory): boolean {
  return folderGalleryBrowseRequiresLogin(major, sub)
}

/** 是否开启加密（加密公开、加密隐藏；含旧数据仅写哈希的情况） */
export function folderContentRequiresPassword(major: ApiCategoryGroup, sub: ApiSubcategory): boolean {
  if (major.encrypted || sub.encrypted) return true
  return !!sub.encryptedPasswordHash?.trim()
}

/** 直接访问 /resource/ 链接是否需要 ?k= 查看密钥 */
export function folderResourceRequiresViewKey(
  major: ApiCategoryGroup,
  sub: ApiSubcategory,
): boolean {
  return folderContentRequiresPassword(major, sub)
}

/** 根据 folderKey 查找所属大分类与二级目录 */
export function lookupGalleryFolder(
  doc: { categories?: ApiCategoryGroup[] } | null | undefined,
  folderKey: string,
): { major: ApiCategoryGroup; sub: ApiSubcategory } | null {
  const fk = folderKey.trim()
  if (!fk || !doc?.categories?.length) return null
  for (const major of doc.categories) {
    const sub = (major.subcategories ?? []).find((s) => s.folderKey === fk)
    if (sub) return { major, sub }
  }
  return null
}

/** 合并大分类与子级设置后的实际访问策略 */
export function effectiveFolderAccessPolicy(
  major: ApiCategoryGroup,
  sub: ApiSubcategory,
): FolderAccessPolicy {
  const encrypted = folderContentRequiresPassword(major, sub)
  const pub = major.public !== false && sub.public !== false
  if (encrypted && pub) return 'encrypted_public'
  if (encrypted && !pub) return 'encrypted_hidden'
  if (!encrypted && !pub) return 'private'
  return 'open'
}

export function folderAccessPolicyHint(policy: FolderAccessPolicy): string {
  switch (policy) {
    case 'encrypted_public':
    case 'encrypted_hidden':
      return '分享直链需附带 ?k= 查看密码'
    case 'private':
      return '菜单仅登录后可见，直链可直接打开'
    default:
      return ''
  }
}

/** 根据 folderKey 判断直链是否需要 ?k= */
export function folderResourceRequiresViewKeyByFolderKey(
  doc: { categories?: ApiCategoryGroup[] } | null | undefined,
  folderKey: string,
): boolean {
  const fk = folderKey.trim()
  if (!fk || isGallerySearchFolderKey(fk)) return false
  const ctx = lookupGalleryFolder(doc, fk)
  if (!ctx) return false
  return folderResourceRequiresViewKey(ctx.major, ctx.sub)
}

/** @deprecated 请使用 folderGalleryBrowseRequiresLogin，语义相同 */
export function folderEntryRequiresLogin(major: ApiCategoryGroup, sub: ApiSubcategory): boolean {
  return folderGalleryBrowseRequiresLogin(major, sub)
}

export function navEntryVisible(
  major: ApiCategoryGroup,
  sub: ApiSubcategory,
  authenticated: boolean,
): boolean {
  if (authenticated) return true
  return navVisibleToGuest(major, sub)
}

/** 根据 folderKey 判断当前登录态下是否允许在导航/路由中展示该二级目录 */
export function folderNavEntryVisible(
  doc: { categories?: ApiCategoryGroup[] } | null | undefined,
  folderKey: string,
  authenticated: boolean,
): boolean {
  const fk = folderKey.trim()
  if (isGallerySearchFolderKey(fk)) return true
  if (!fk || !doc?.categories?.length) return true
  const placeholderMajorId = parseMajorPlaceholderFolderKey(fk)
  if (placeholderMajorId != null) {
    const major = doc.categories.find((m) => m.id === placeholderMajorId)
    return major ? majorNavEntryVisible(major, authenticated) : false
  }
  for (const major of doc.categories) {
    const sub = (major.subcategories ?? []).find((s) => s.folderKey === fk)
    if (sub) return navEntryVisible(major, sub, authenticated)
  }
  return true
}
