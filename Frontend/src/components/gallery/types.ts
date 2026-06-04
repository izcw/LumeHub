/** 画廊列表项（网格 / 瀑布流共用） */
export interface GalleryDisplayItem {
  id: string
  fullSrc: string
  src: string
  shortUrl?: string
  originalUrl?: string
  editedUrl?: string
  useEdited?: boolean
  title?: string
  tags?: readonly string[]
  linkName?: string
  uploadedAt?: string
  updatedAt?: string
  fileSize?: number
  format?: string
  mediaKind?: string
  categoryName?: string
  isLivePhoto?: boolean
  liveVideoUrl?: string
  cardDate?: string
  isNew?: boolean
  loadTime?: number
  masonryAspectHW?: number
  masonryCol?: number
  masonryRow?: number
}

export type GalleryItemStyleFn = (
  item: GalleryDisplayItem,
  batchIndex: number,
) => Readonly<Record<string, string>> | undefined
