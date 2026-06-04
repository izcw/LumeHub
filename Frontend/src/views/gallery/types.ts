import type { GalleryDisplayItem } from '@/components/gallery/types'

export type GalleryListItem = GalleryDisplayItem

export type LoadPageOpts = { scrollToWaterfall?: boolean; silent?: boolean }

export type UploadTaskStatus = 'uploading' | 'success' | 'error' | 'canceled'

export type UploadTask = {
  id: string
  name: string
  loaded: number
  total: number
  progress: number
  status: UploadTaskStatus
  errorText: string
}
