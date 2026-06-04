import { http } from '@/api/http'
import { parseApiErrorMessage } from '@/utils/apiError'
import type { ApiAccountPublic, ApiCategoriesDoc, ApiStorageStatus } from '@/api/types'

export type CategoryVisibilityPatch = {
  scope: 'major' | 'sub'
  majorId: number
  subId?: number
  public?: boolean
  encrypted?: boolean
  encryptedPassword?: string
}

export type CategoryNamePatch = {
  scope: 'major' | 'sub'
  majorId: number
  subId?: number
  name: string
}

export type CategoryFolderKeyPatch = {
  majorId: number
  subId: number
  folderKey: string
}

export type CategorySubMajorPatch = {
  majorId: number
  subId: number
  targetMajorId: number
}

export async function fetchAuthAccounts(): Promise<ApiAccountPublic[]> {
  const { data } = await http.get<ApiAccountPublic[]>('/api/auth/accounts')
  return data
}

export async function patchAuthAccount(
  id: string,
  body: {
    displayName: string
    email: string
    roles: string[]
    permissions: string[]
    currentPassword?: string
    newPassword?: string
  },
): Promise<ApiAccountPublic> {
  try {
    const { data } = await http.patch<ApiAccountPublic>(`/api/auth/accounts/${encodeURIComponent(id)}`, body)
    return data
  } catch (e) {
    throw new Error(parseApiErrorMessage(e, '保存失败'))
  }
}

export async function postAuthAccount(body: {
  displayName: string
  email: string
  password: string
  roles: string[]
  permissions: string[]
}): Promise<ApiAccountPublic> {
  const { data } = await http.post<ApiAccountPublic>('/api/auth/accounts', body)
  return data
}

export async function deleteAuthAccount(id: string): Promise<void> {
  await http.delete(`/api/auth/accounts/${encodeURIComponent(id)}`)
}

export async function patchCategoriesVisibility(
  patches: CategoryVisibilityPatch[],
): Promise<ApiCategoriesDoc> {
  const { data } = await http.patch<ApiCategoriesDoc>('/api/categories/visibility', { patches })
  return data
}

export async function patchCategoriesNames(
  patches: CategoryNamePatch[],
): Promise<ApiCategoriesDoc> {
  const { data } = await http.patch<ApiCategoriesDoc>('/api/categories/name', { patches })
  return data
}

export async function patchCategoriesFolderKeys(
  patches: CategoryFolderKeyPatch[],
): Promise<ApiCategoriesDoc> {
  const { data } = await http.patch<ApiCategoriesDoc>('/api/categories/folder-key', { patches })
  return data
}

export async function patchCategoriesSubMajor(
  patches: CategorySubMajorPatch[],
): Promise<ApiCategoriesDoc> {
  const { data } = await http.patch<ApiCategoriesDoc>('/api/categories/sub-major', { patches })
  return data
}

export async function postCategoryMajor(body: {
  majorName: string
  subName?: string
  folderKey?: string
  public: boolean
}): Promise<ApiCategoriesDoc> {
  const { data } = await http.post<ApiCategoriesDoc>('/api/categories/major', body)
  return data
}

export async function postCategorySub(body: {
  majorId: number
  name: string
  folderKey: string
  public: boolean
}): Promise<ApiCategoriesDoc> {
  const { data } = await http.post<ApiCategoriesDoc>('/api/categories/sub', body)
  return data
}

export type CategoryDeleteResult = ApiCategoriesDoc & { trashedItems: number }

export async function deleteCategoryMajor(majorId: number): Promise<CategoryDeleteResult> {
  try {
    const { data } = await http.delete<CategoryDeleteResult>('/api/categories/major', {
      data: { majorId },
    })
    return data
  } catch (e) {
    throw new Error(parseApiErrorMessage(e, '删除失败'))
  }
}

export async function deleteCategorySub(majorId: number, subId: number): Promise<CategoryDeleteResult> {
  try {
    const { data } = await http.delete<CategoryDeleteResult>('/api/categories/sub', {
      data: { majorId, subId },
    })
    return data
  } catch (e) {
    throw new Error(parseApiErrorMessage(e, '删除失败'))
  }
}

/** 写回 categories.json 的 sort：大类须传全量 id 列表；二级须传该大类下全量子分类 id 列表 */
export type CategoriesNavOrderPatch = {
  primaryMajorIds?: number[]
  subOrders?: { majorId: number; subIds: number[] }[]
}

export async function patchCategoriesNavOrder(
  body: CategoriesNavOrderPatch,
): Promise<ApiCategoriesDoc> {
  const { data } = await http.patch<ApiCategoriesDoc>('/api/categories/nav-order', body)
  return data
}

export async function fetchStorageStatus(): Promise<ApiStorageStatus> {
  const { data } = await http.get<ApiStorageStatus>('/api/storage')
  return data
}

export async function patchStorageQuota(quotaGb: number): Promise<ApiStorageStatus> {
  const { data } = await http.patch<ApiStorageStatus>('/api/storage', { quotaGb })
  return data
}

export async function recalculateStorage(): Promise<ApiStorageStatus> {
  const { data } = await http.post<ApiStorageStatus>('/api/storage/recalculate')
  return data
}
