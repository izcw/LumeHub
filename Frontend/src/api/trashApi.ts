import { http } from '@/api/http'
import { parseApiErrorMessage } from '@/utils/apiError'
import type { ApiTrashItem, TrashRestoreFolderResult, TrashRestoreResult } from '@/api/types'

export async function fetchTrashItems(): Promise<ApiTrashItem[]> {
  const { data } = await http.get<ApiTrashItem[]>('/api/trash')
  return data
}

export async function permanentDeleteTrashItem(folderKey: string, itemId: string): Promise<void> {
  await http.delete(
    `/api/trash/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}`,
  )
}

export async function clearAllTrashItems(): Promise<number> {
  const { data } = await http.delete<{ deleted: number }>('/api/trash')
  return data.deleted ?? 0
}

export async function clearTrashFolder(folderKey: string): Promise<number> {
  try {
    const { data } = await http.delete<{ deleted: number }>(
      `/api/trash/${encodeURIComponent(folderKey)}`,
    )
    return data.deleted ?? 0
  } catch (e) {
    throw new Error(parseApiErrorMessage(e, '清空失败'))
  }
}

export async function restoreTrashItem(folderKey: string, itemId: string): Promise<TrashRestoreResult> {
  try {
    const { data } = await http.post<TrashRestoreResult>(
      `/api/trash/${encodeURIComponent(folderKey)}/items/${encodeURIComponent(itemId)}/restore`,
    )
    return data
  } catch (e) {
    throw new Error(parseApiErrorMessage(e, '恢复失败'))
  }
}

export async function restoreTrashFolder(folderKey: string): Promise<TrashRestoreFolderResult> {
  try {
    const { data } = await http.post<TrashRestoreFolderResult>(
      `/api/trash/${encodeURIComponent(folderKey)}/restore`,
    )
    return data
  } catch (e) {
    throw new Error(parseApiErrorMessage(e, '恢复失败'))
  }
}
