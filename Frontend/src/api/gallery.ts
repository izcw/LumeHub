import { http } from '@/api/http'
import type { ApiCategoriesDoc, ApiCategoryDetail, ApiCategoryItem, ApiLayout } from '@/api/types'
import { getDeviceId } from '@/utils/deviceId'
import { captureVideoPosterBlob, isVideoUploadFile } from '@/utils/videoPoster'
import type { AxiosProgressEvent } from 'axios'

export type MasonryPlacementMap = Record<string, { col: number; row: number }>

/** 超过此大小走分片 + 断点续传（服务端合并）；总用量受存储配额限制 */
const RESUMABLE_THRESHOLD_BYTES = 5 * 1024 * 1024
/** 过大会阻塞主线程，仅对较小文件计算客户端 SHA-256 供服务端提前校验 */
const CLIENT_SHA256_MAX_BYTES = 64 * 1024 * 1024

export async function fetchCategories(): Promise<ApiCategoriesDoc> {
  const { data } = await http.get<ApiCategoriesDoc>('/api/categories')
  return data
}

export async function fetchCategoryDetail(
  folderKey: string,
  opts?: { grant?: string; viewPassword?: string },
): Promise<ApiCategoryDetail> {
  const url = `/api/category/${encodeURIComponent(folderKey)}`
  const params: Record<string, string> = {}
  const grant = opts?.grant?.trim()
  const viewPassword = opts?.viewPassword?.trim()
  if (grant) {
    params.vg = grant
  } else if (viewPassword) {
    params.vp = viewPassword
  }
  const { data } = await http.get<ApiCategoryDetail>(url, {
    params: Object.keys(params).length > 0 ? params : undefined,
  })
  return data
}

export async function postCategoryViewUnlock(folderKey: string, password: string) {
  const { data } = await http.post<{ grant?: string; expiresAt?: string; error?: string }>(
    `/api/category/${encodeURIComponent(folderKey)}/view-unlock`,
    { password, deviceId: getDeviceId() },
  )
  return data
}

export async function patchCategoryLayout(folderKey: string, layout: ApiLayout): Promise<void> {
  await http.patch(`/api/category/${encodeURIComponent(folderKey)}/layout`, { layout })
}

export async function patchCategoryItemOrder(
  folderKey: string,
  orderedItemIds: readonly string[],
  masonryPlacement?: MasonryPlacementMap,
): Promise<void> {
  await http.patch(`/api/category/${encodeURIComponent(folderKey)}/item-order`, {
    orderedItemIds: [...orderedItemIds],
    masonryPlacement,
  })
}

async function sha256HexOptional(file: File): Promise<string | undefined> {
  if (file.size > CLIENT_SHA256_MAX_BYTES) return undefined
  const subtle = globalThis.crypto?.subtle
  if (!subtle) return undefined
  const buf = await file.arrayBuffer()
  const digest = await subtle.digest('SHA-256', buf)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

type UploadSessionCreateRes = {
  uploadId: string
  chunkSize: number
}

type UploadSessionStatus = {
  filename: string
  size: number
  chunkSize: number
  totalChunks: number
  received: number[]
}

function chunkByteLength(total: number, chunkSize: number, index: number, nChunks: number): number {
  if (index < 0 || index >= nChunks) return 0
  if (index === nChunks - 1) {
    const r = total % chunkSize
    return r === 0 ? chunkSize : r
  }
  return chunkSize
}

async function sleep(ms: number): Promise<void> {
  await new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms)
  })
}

async function putChunkWithRetry(
  url: string,
  blob: Blob,
  signal: AbortSignal | undefined,
): Promise<void> {
  let lastErr: unknown
  for (let attempt = 0; attempt < 3; attempt++) {
    if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
    try {
      await http.put(url, blob, {
        signal,
        timeout: 0,
        headers: { 'Content-Type': 'application/octet-stream' },
        maxBodyLength: Infinity,
        maxContentLength: Infinity,
      })
      return
    } catch (e) {
      lastErr = e
      if (attempt < 2) await sleep(380 * (attempt + 1))
    }
  }
  throw lastErr
}

async function uploadVideoPosterIfNeeded(
  folderKey: string,
  itemId: string,
  file: File,
  signal?: AbortSignal,
): Promise<void> {
  if (!itemId || !isVideoUploadFile(file)) return
  const poster = await captureVideoPosterBlob(file)
  if (!poster) return
  const fd = new FormData()
  fd.append('poster', poster, 'poster.jpg')
  try {
    await http.post(
      `/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}/thumbnail`,
      fd,
      {
        signal,
        timeout: 120_000,
        maxBodyLength: Infinity,
        maxContentLength: Infinity,
      },
    )
  } catch {
    /* 封面失败不影响主文件已入�?*/
  }
}

async function deleteUploadSession(folderKey: string, sessionId: string): Promise<void> {
  try {
    await http.delete(`/api/category/${encodeURIComponent(folderKey)}/upload/session/${encodeURIComponent(sessionId)}`)
  } catch {
    /* 清理失败忽略 */
  }
}

async function uploadCategoryFileResumableInner(
  folderKey: string,
  file: File,
  onProgress?: (progress: { loaded: number; total: number }) => void,
  signal?: AbortSignal,
  motionFile?: File,
): Promise<void> {
  const preHash = await sha256HexOptional(file)
  const { data: created } = await http.post<UploadSessionCreateRes>(
    `/api/category/${encodeURIComponent(folderKey)}/upload/session`,
    {
      filename: file.name,
      size: file.size,
      ...(preHash ? { sha256: preHash } : {}),
    },
    { timeout: 0, signal },
  )
  const uploadId = created.uploadId
  const chunkSize = created.chunkSize
  let finished = false
  try {
    const { data: status0 } = await http.get<UploadSessionStatus>(
      `/api/category/${encodeURIComponent(folderKey)}/upload/session/${encodeURIComponent(uploadId)}`,
      { timeout: 0, signal },
    )
    const nChunks = status0.totalChunks
    const received = new Set<number>(status0.received)

    const report = (loaded: number) => {
      onProgress?.({ loaded, total: file.size + (motionFile?.size ?? 0) })
    }

    let loaded = 0
    for (const i of received) {
      loaded += chunkByteLength(file.size, chunkSize, i, nChunks)
    }
    report(loaded)

    for (let i = 0; i < nChunks; i++) {
      if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
      if (received.has(i)) continue
      const len = chunkByteLength(file.size, chunkSize, i, nChunks)
      const start = i * chunkSize
      const slice = file.slice(start, start + len)
      const url = `/api/category/${encodeURIComponent(folderKey)}/upload/session/${encodeURIComponent(
        uploadId,
      )}/chunk/${i}`
      await putChunkWithRetry(url, slice, signal)
      received.add(i)
      loaded += len
      report(Math.min(loaded, file.size))
    }

    const { data: completed } = await http.post<{ id: string }>(
      `/api/category/${encodeURIComponent(folderKey)}/upload/session/${encodeURIComponent(uploadId)}/complete`,
      preHash ? { sha256: preHash } : {},
      { timeout: 0, signal },
    )
    report(file.size)
    if (completed?.id) {
      await uploadVideoPosterIfNeeded(folderKey, completed.id, file, signal)
    }
    if (motionFile) {
      await attachItemCompanion(folderKey, completed.id, motionFile, signal, (motionLoaded) => {
        onProgress?.({
          loaded: file.size + motionLoaded,
          total: file.size + motionFile.size,
        })
      })
    }
    finished = true
  } finally {
    if (!finished) void deleteUploadSession(folderKey, uploadId)
  }
}

export async function attachItemCompanion(
  folderKey: string,
  itemId: string,
  motionFile: File,
  signal?: AbortSignal,
  onProgress?: (loaded: number) => void,
): Promise<void> {
  const fd = new FormData()
  fd.append('rawFile', motionFile)
  await http.post(
    `/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}/companion`,
    fd,
    {
      signal,
      timeout: 0,
      onUploadProgress: (evt: AxiosProgressEvent) => {
        onProgress?.(evt.loaded ?? 0)
      },
      maxBodyLength: Infinity,
      maxContentLength: Infinity,
    },
  )
}

export async function uploadCategoryFile(
  folderKey: string,
  file: File,
  onProgress?: (progress: { loaded: number; total: number }) => void,
  signal?: AbortSignal,
  motionFile?: File,
): Promise<void> {
  const totalBytes = file.size + (motionFile?.size ?? 0)
  const report = (mainLoaded: number, motionLoaded = motionFile?.size ?? 0) => {
    onProgress?.({ loaded: mainLoaded + motionLoaded, total: totalBytes })
  }

  if (file.size > RESUMABLE_THRESHOLD_BYTES) {
    return uploadCategoryFileResumableInner(folderKey, file, onProgress, signal, motionFile)
  }
  const fd = new FormData()
  fd.append('file', file)
  if (motionFile) fd.append('rawFile', motionFile)
  const { data: created } = await http.post<ApiCategoryItem>(
    `/api/category/${encodeURIComponent(folderKey)}/upload`,
    fd,
    {
      signal,
      timeout: 0,
      onUploadProgress: (evt: AxiosProgressEvent) => {
        report(evt.loaded ?? 0, 0)
      },
      maxBodyLength: Infinity,
      maxContentLength: Infinity,
    },
  )
  report(file.size, motionFile?.size ?? 0)
  if (created?.id) {
    await uploadVideoPosterIfNeeded(folderKey, created.id, file, signal)
  }
}

export type PatchCategoryItemBody = {
  linkName?: string
  title?: string
  tags?: string[]
  revertEdited?: boolean
}

export async function deleteCategoryItem(folderKey: string, itemId: string): Promise<void> {
  await http.delete(`/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}`)
}

export async function transferCategoryItem(
  folderKey: string,
  itemId: string,
  targetFolderKey: string,
): Promise<void> {
  await http.post(
    `/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}/transfer`,
    { targetFolderKey },
  )
}

export async function patchCategoryItem(
  folderKey: string,
  itemId: string,
  body: PatchCategoryItemBody,
): Promise<import('@/api/types').ApiCategoryItem> {
  const { data } = await http.patch<import('@/api/types').ApiCategoryItem>(
    `/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}`,
    body,
  )
  return data
}

export async function patchCategoryItemWithFile(
  folderKey: string,
  itemId: string,
  file: File,
  fields?: { linkName?: string; title?: string; tags?: string },
  opts?: { saveMode?: 'edit' | 'replace'; signal?: AbortSignal },
): Promise<import('@/api/types').ApiCategoryItem> {
  const fd = new FormData()
  fd.append('file', file)
  fd.append('saveMode', opts?.saveMode ?? 'edit')
  if (fields?.linkName != null) fd.append('linkName', fields.linkName)
  if (fields?.title != null) fd.append('title', fields.title)
  if (fields?.tags != null) fd.append('tags', fields.tags)
  const { data } = await http.patch<import('@/api/types').ApiCategoryItem>(
    `/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}`,
    fd,
    {
      signal: opts?.signal,
      timeout: 0,
      maxBodyLength: Infinity,
      maxContentLength: Infinity,
    },
  )
  return data
}

export async function revertCategoryItemEdit(
  folderKey: string,
  itemId: string,
): Promise<import('@/api/types').ApiCategoryItem> {
  const { data } = await http.patch<import('@/api/types').ApiCategoryItem>(
    `/api/category/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}`,
    { revertEdited: true },
  )
  return data
}
