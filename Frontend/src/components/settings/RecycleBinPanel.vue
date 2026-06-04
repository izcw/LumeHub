<template>
  <div class="recycle-bin">
    <div class="recycle-bin__head">
      <p class="recycle-bin__desc">
        删除的文件按原目录分组展示，可单张恢复或整组恢复；目录已删除时恢复会自动重建。永久删除或超过 30 天将不可恢复。
      </p>
      <div class="recycle-bin__toolbar">
        <Popconfirm
          v-if="items.length > 0"
          title="确定清空回收站吗？所有文件将被永久删除，此操作不可恢复。"
          confirm-text="清空"
          @confirm="onClearAll"
        >
          <template #trigger>
            <Button
              size="small"
              class="recycle-bin__clear-btn"
              native-type="button"
              :disabled="clearing"
            >
              {{ clearing ? '清空中…' : '一键清空' }}
            </Button>
          </template>
        </Popconfirm>
        <Input
          v-model="searchQuery"
          type="search"
          class="recycle-bin__search"
          placeholder="搜索图片名称、标题或分类…"
          aria-label="搜索回收站"
        />
        <span v-if="items.length > 0" class="recycle-bin__count">
          共 {{ items.length }} 项 · {{ folderGroups.length }} 个目录
        </span>
      </div>
    </div>

    <div v-if="loadErr" class="status-message is-error">{{ loadErr }}</div>
    <div v-else-if="loading" class="recycle-bin__empty">加载中…</div>
    <div v-else-if="folderGroups.length === 0" class="recycle-bin__empty">
      {{ searchQuery.trim() ? '没有匹配的回收站文件' : '回收站为空' }}
    </div>
    <div v-else ref="scrollRootRef" class="recycle-bin__groups">
      <section v-for="group in folderGroups" :key="group.folderKey" class="recycle-bin__group">
        <div class="recycle-bin__group-head">
          <div class="recycle-bin__group-main">
            <p class="recycle-bin__group-title">{{ group.label }}</p>
            <p class="recycle-bin__group-meta">
              {{ group.items.length }} 项 · <span class="is-mono">{{ group.folderKey }}</span>
              <span v-if="group.categoryMissing" class="recycle-bin__missing-tag">目录已删除</span>
            </p>
          </div>
          <div class="recycle-bin__group-actions">
            <Button
              size="small"
              class="recycle-bin__restore-all-btn"
              native-type="button"
              :disabled="restoringFolderKey === group.folderKey || clearingFolderKey === group.folderKey"
              @click="onRestoreFolder(group.folderKey)"
            >
              {{ restoringFolderKey === group.folderKey ? '恢复中…' : '全部恢复' }}
            </Button>
            <Popconfirm
              :title="`确定永久删除「${group.label}」下全部 ${group.items.length} 项吗？此操作不可恢复。`"
              confirm-text="清空"
              @confirm="onClearFolder(group.folderKey)"
            >
              <template #trigger>
                <Button
                  size="small"
                  class="recycle-bin__clear-folder-btn"
                  native-type="button"
                  :disabled="restoringFolderKey === group.folderKey || clearingFolderKey === group.folderKey"
                >
                  {{ clearingFolderKey === group.folderKey ? '清空中…' : '全部清空' }}
                </Button>
              </template>
            </Popconfirm>
          </div>
        </div>
        <div class="recycle-bin__grid">
          <RecycleBinCell
            v-for="item in group.items"
            :key="item.id"
            :item="item"
            @delete="onPermanentDelete(item)"
            @restore="onRestore(item)"
          />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, provide, ref } from 'vue'
import Input from '@/components/shared-ui/Input.vue'
import Button from '@/components/shared-ui/Button.vue'
import Popconfirm from '@/components/shared-ui/Popconfirm.vue'
import RecycleBinCell from '@/components/settings/RecycleBinCell.vue'
import {
  fetchTrashItems,
  permanentDeleteTrashItem,
  clearAllTrashItems,
  clearTrashFolder,
  restoreTrashFolder,
  restoreTrashItem,
} from '@/api/trashApi'
import type { ApiTrashItem } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { useMessageStore } from '@/stores/message'
import { useGalleryItemsSyncStore } from '@/stores/galleryItemsSync'
import { formatCardDateFromIso } from '@/views/gallery/utils'

type RecycleBinDisplayItem = {
  id: string
  folderKey: string
  itemId: string
  src: string
  fullSrc: string
  liveVideoUrl?: string
  isLivePhoto?: boolean
  cardDate: string
  fileSize?: number
  format?: string
  mediaKind?: string
  title?: string
  linkName?: string
  categoryLabel: string
  categoryMissing: boolean
  searchText: string
}

type FolderGroup = {
  folderKey: string
  label: string
  categoryMissing: boolean
  items: RecycleBinDisplayItem[]
}

const auth = useAuthStore()
const categoryNav = useCategoryNavStore()
const messageStore = useMessageStore()
const galleryItemsSyncStore = useGalleryItemsSyncStore()

const scrollRootRef = ref<HTMLElement | null>(null)
provide('recycleBinScrollRoot', scrollRootRef)
const loading = ref(false)
const clearing = ref(false)
const restoringFolderKey = ref('')
const clearingFolderKey = ref('')
const loadErr = ref('')
const items = ref<RecycleBinDisplayItem[]>([])
const searchQuery = ref('')

function mapTrashItem(it: ApiTrashItem): RecycleBinDisplayItem {
  const url = auth.appendAccessToResourceUrl(it.url)
  const thumb = it.thumbnailUrl ? auth.appendAccessToResourceUrl(it.thumbnailUrl) : ''
  const liveVideoUrl = it.liveVideoUrl
    ? auth.appendAccessToResourceUrl(it.liveVideoUrl)
    : undefined
  const categoryLabel = `${it.majorName} / ${it.subName}`
  const title = (it.title ?? '').trim()
  const linkName = (it.linkName ?? '').trim()
  const searchText = [title, linkName, it.majorName, it.subName, it.folderKey, ...(it.tags ?? [])]
    .join(' ')
    .toLowerCase()

  return {
    id: `${it.folderKey}::${it.id}`,
    folderKey: it.folderKey,
    itemId: it.id,
    src: thumb || url,
    fullSrc: url,
    liveVideoUrl,
    isLivePhoto: it.isLivePhoto,
    cardDate: formatCardDateFromIso(it.deletedAt || it.uploadedAt || it.updatedAt),
    fileSize: it.fileSize,
    format: it.format,
    mediaKind: it.mediaKind,
    title,
    linkName,
    categoryLabel,
    categoryMissing: it.categoryMissing === true,
    searchText,
  }
}

function matchesSearch(item: RecycleBinDisplayItem, query: string): boolean {
  if (item.searchText.includes(query)) return true
  if (item.mediaKind === 'image' && (item.title || item.linkName)) {
    return (item.title ?? '').toLowerCase().includes(query) || (item.linkName ?? '').toLowerCase().includes(query)
  }
  return false
}

const folderGroups = computed<FolderGroup[]>(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const groups = new Map<string, FolderGroup>()

  for (const item of items.value) {
    if (q && !matchesSearch(item, q)) continue

    const existing = groups.get(item.folderKey)
    if (existing) {
      existing.items.push(item)
      continue
    }
    groups.set(item.folderKey, {
      folderKey: item.folderKey,
      label: item.categoryLabel,
      categoryMissing: item.categoryMissing,
      items: [item],
    })
  }

  return [...groups.values()]
    .map((group) => ({
      ...group,
      items: [...group.items].sort((a, b) => b.cardDate.localeCompare(a.cardDate)),
    }))
    .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN'))
})

function restoreSuccessMessage(restored: number, categoryRecreated: boolean): string {
  if (categoryRecreated) {
    return restored > 1
      ? `已恢复 ${restored} 项，原目录已重新创建`
      : '已恢复，原目录已重新创建'
  }
  return restored > 1 ? `已恢复 ${restored} 项` : '已恢复到原分类'
}

async function afterRestore(folderKey: string) {
  galleryItemsSyncStore.markCategoryItemsChanged(folderKey)
  await categoryNav.reloadFromServer()
}

async function loadItems() {
  loading.value = true
  loadErr.value = ''
  try {
    const list = await fetchTrashItems()
    items.value = list.map(mapTrashItem)
  } catch (e) {
    loadErr.value = e instanceof Error ? e.message : '加载失败'
    items.value = []
  } finally {
    loading.value = false
  }
}

async function onPermanentDelete(item: RecycleBinDisplayItem) {
  try {
    await permanentDeleteTrashItem(item.folderKey, item.itemId)
    items.value = items.value.filter((row) => row.id !== item.id)
    messageStore.show('已永久删除', 'success')
  } catch {
    messageStore.show('删除失败', 'error')
  }
}

async function onRestore(item: RecycleBinDisplayItem) {
  try {
    const out = await restoreTrashItem(item.folderKey, item.itemId)
    items.value = items.value.filter((row) => row.id !== item.id)
    await afterRestore(item.folderKey)
    messageStore.show(restoreSuccessMessage(1, out.categoryRecreated), 'success')
  } catch (e) {
    messageStore.show(e instanceof Error ? e.message : '恢复失败', 'error')
  }
}

async function onRestoreFolder(folderKey: string) {
  if (restoringFolderKey.value || clearingFolderKey.value) return
  restoringFolderKey.value = folderKey
  try {
    const out = await restoreTrashFolder(folderKey)
    items.value = items.value.filter((row) => row.folderKey !== folderKey)
    await afterRestore(folderKey)
    messageStore.show(restoreSuccessMessage(out.restored, out.categoryRecreated), 'success')
  } catch (e) {
    messageStore.show(e instanceof Error ? e.message : '恢复失败', 'error')
  } finally {
    restoringFolderKey.value = ''
  }
}

async function onClearFolder(folderKey: string) {
  if (restoringFolderKey.value || clearingFolderKey.value) return
  clearingFolderKey.value = folderKey
  try {
    const deleted = await clearTrashFolder(folderKey)
    items.value = items.value.filter((row) => row.folderKey !== folderKey)
    messageStore.show(deleted > 0 ? `已永久删除 ${deleted} 项` : '该目录回收站已为空', 'success')
  } catch (e) {
    messageStore.show(e instanceof Error ? e.message : '清空失败', 'error')
  } finally {
    clearingFolderKey.value = ''
  }
}

async function onClearAll() {
  if (clearing.value || items.value.length === 0) return
  clearing.value = true
  try {
    const deleted = await clearAllTrashItems()
    items.value = []
    searchQuery.value = ''
    messageStore.show(deleted > 0 ? `已清空 ${deleted} 项` : '回收站已为空', 'success')
  } catch {
    messageStore.show('清空失败', 'error')
  } finally {
    clearing.value = false
  }
}

defineExpose({ reload: loadItems })
</script>

<style scoped lang="scss">
.recycle-bin {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

.recycle-bin__head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.recycle-bin__desc {
  margin: 0;
  font-size: 12px;
  color: #666;
  line-height: 20px;
}

.recycle-bin__toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.recycle-bin__search {
  flex: 1 1 220px;
  min-width: 0;
}

.recycle-bin__count {
  font-size: 12px;
  color: #9ca3af;
  white-space: nowrap;
}

.recycle-bin__clear-btn {
  flex-shrink: 0;
}

.recycle-bin__empty {
  padding: 32px 0;
  text-align: center;
  font-size: 13px;
  color: #9ca3af;
}

.recycle-bin__groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
  max-height: calc(650px - 180px);
  padding-right: 4px;
}

.recycle-bin__group {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid #ececec;
  border-radius: 10px;
  background: #fafafa;
}

.recycle-bin__group-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.recycle-bin__group-main {
  min-width: 0;
}

.recycle-bin__group-title {
  margin: 0;
  font-size: 13px;
  color: #333;
  font-weight: 600;
  line-height: 20px;
}

.recycle-bin__group-meta {
  margin: 4px 0 0;
  font-size: 12px;
  color: #9ca3af;
  line-height: 18px;
}

.recycle-bin__missing-tag {
  margin-left: 6px;
  font-size: 11px;
  color: #b45309;
}

.recycle-bin__restore-all-btn,
.recycle-bin__clear-folder-btn {
  flex-shrink: 0;
}

.recycle-bin__group-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.recycle-bin__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
  align-items: start;
}

.recycle-bin__grid :deep(.recycle-bin__cell) {
  min-width: 0;

  .card {
    box-shadow: none;
    cursor: default;
    transform: none !important;
    will-change: auto;
  }

  .card:hover .image,
  .card:hover .video-poster,
  .card:hover .card-video,
  .card:hover .file-type-panel {
    transform: none !important;
    filter: none !important;
  }
}
</style>
