export type ApiLayout = {
  mode: 'masonry' | 'grid' | 'card'
  columns: 'auto' | '1' | '2' | '3' | '4' | '5' | '6'
  pageSize?: number
}

export type ApiCategoryItem = {
  id: string
  /** 与后端 items.json 一致，升序；0 或未返回表示未指定 */
  sort?: number
  masonryCol?: number
  masonryRow?: number
  uploadedAt?: string
  updatedAt?: string
  shortUrl?: string
  linkName?: string
  url: string
  thumbnailUrl?: string
  groupId?: string
  rawUrl?: string
  isLivePhoto?: boolean
  liveVideoUrl?: string
  title?: string
  tags?: string[]
  /** 主资源字节数 */
  fileSize?: number
  originalUrl?: string
  editedUrl?: string
  useEdited?: boolean
  format?: string
  mediaKind?: string
}

export type ApiTrashItem = ApiCategoryItem & {
  folderKey: string
  majorName: string
  subName: string
  deletedAt: string
  expiresAt?: string
  categoryMissing?: boolean
}

export type TrashRestoreResult = {
  categoryRecreated: boolean
}

export type TrashRestoreFolderResult = {
  restored: number
  categoryRecreated: boolean
}

export type ApiCategoryDetail = {
  id: number
  name: string
  folderKey: string
  layout: ApiLayout
  /** uploaded_at | updated_at | sort */
  itemSortBy?: string
  items: ApiCategoryItem[]
}

export type ApiSubcategory = {
  id: number
  sort?: number
  name: string
  folderKey: string
  layout: ApiLayout
  itemSortBy?: string
  /** 与 encrypted 组合表示访问策略；false 时访客主导航不展示该二级（私密 / 加密隐藏） */
  public?: boolean
  /** 与 encrypted 组合：公开可见 / 加密公开 / 私密（隐藏菜单）/ 加密隐藏（菜单登录可见，直链需密钥） */
  encrypted?: boolean
  /** 查看密码的 SHA256 哈希（由服务端管理，仅用于判断是否已设置） */
  encryptedPasswordHash?: string
  apiEnabled?: boolean
  apiKeyHash?: string
}

/** 大分类；其下 subcategories 为二级（对应 resource/{folderKey}） */
export type ApiCategoryGroup = {
  id: number
  sort?: number
  name: string
  key?: string
  /** 与 encrypted 组合表示整组默认访问边界；子级可单独覆盖 */
  public?: boolean
  encrypted?: boolean
  encryptedPasswordHash?: string
  apiEnabled?: boolean
  apiKeyHash?: string
  subcategories: ApiSubcategory[]
}

export type ApiCategoriesDoc = {
  version: number
  homeFolderKey?: string
  categories: ApiCategoryGroup[]
}

/** 当前登录用户（不含密码） */
export type ApiAccountPublic = {
  id: string
  username?: string
  email: string
  displayName: string
  avatar?: string
  roles?: string[]
  permissions?: string[]
}

export type ApiAuthStatus = {
  authenticated: boolean
  authConfigured?: boolean
  /** 是否使用 accounts.json 多账号（需邮箱登录） */
  useAccounts?: boolean
  /** 未登录头像 URL（来自 accounts.json） */
  guestAvatarUrl?: string
  /** 已登录且无头像时的占位图 URL（来自 accounts.json） */
  loggedInFallbackAvatarUrl?: string
}

export type ApiAuthMeResponse = {
  authenticated: boolean
  authConfigured?: boolean
  user: ApiAccountPublic | null
}

/** 实例存储配额与用量 */
export type ApiStorageStatus = {
  quotaBytes: number
  usedBytes: number
  availableBytes: number
  usedPercent: number
  calculatedAt?: string
}
