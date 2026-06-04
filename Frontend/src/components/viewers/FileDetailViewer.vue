<template>
  <Dialog
    :open="open"
    title="文件详情"
    width="420px"
    height="auto"
    :body-padded="false"
    :show-actions="false"
    :z-index="2800"
    @close="emitClose"
  >
    <div v-if="payload" class="file-detail">
      <GalleryFilePreview
        show-details
        :src="payload.fullSrc"
        :file-name="displayName"
        :format="payload.format"
        :media-kind="payload.mediaKind"
        :file-size="payload.fileSize"
        :uploaded-at="payload.uploadedAt"
        :updated-at="payload.updatedAt"
      />

      <div class="file-detail__grid">
        <div v-for="row in detailRows" :key="row.label" class="file-detail__row">
          <span class="file-detail__label">{{ row.label }}</span>
          <button
            v-if="row.copyable"
            type="button"
            class="file-detail__value file-detail__value--copy"
            :title="row.value"
            @click="copyValue(row.value)"
          >
            {{ row.value }}
          </button>
          <span v-else class="file-detail__value" :title="row.value">{{ row.value }}</span>
        </div>
      </div>

      <div class="file-detail__actions">
        <Button native-type="button" @click="openFile">打开文件</Button>
        <Button type="info" native-type="button" @click="emit('download')">下载</Button>
        <Button type="info" native-type="button" @click="emit('copy-link')">复制链接</Button>
      </div>
    </div>
  </Dialog>

  <Teleport to="body">
    <div
      v-if="open"
      class="file-detail-nav-layer"
      @mousemove="onNavLayerMouseMove"
      @mouseleave="onViewerMouseLeave"
    >
      <ViewerChrome
        :show-close="false"
        :show-prev="canPrev"
        :show-next="canNext"
        :show-prev-highlight="showPrevNav"
        :show-next-highlight="showNextNav"
        :show-bottom-highlight="showBottomBar || toolbarPinned"
        :current="navCurrent ?? 0"
        :total="navTotal ?? 0"
        @prev="emitNavigate('prev')"
        @next="emitNavigate('next')"
      >
        <template v-if="$slots.toolbar" #toolbar>
          <slot name="toolbar" />
        </template>
      </ViewerChrome>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onUnmounted, watch } from 'vue'
import Dialog from '@/components/shared-ui/Dialog.vue'
import Button from '@/components/shared-ui/Button.vue'
import GalleryFilePreview from '@/components/gallery/GalleryFilePreview.vue'
import ViewerChrome from './shared/ViewerChrome.vue'
import { useViewerEdgeNav } from './shared/useViewerEdgeNav'
import { galleryExtFromUrl, galleryMediaKindFromUrl, galleryMediaKindLabel } from '@/utils/galleryMedia'
import { formatFileSize } from '@/utils/formatFileSize'
import { copyTextToClipboard } from '@/utils/clipboard'
import { buildShortLinkPreview, linkNameStem, toAbsoluteResourceUrl } from '@/utils/resourceUrl'
import { useMessageStore } from '@/stores/message'

export type GalleryFileDetailPayload = {
  fullSrc: string
  originalUrl?: string
  shortUrl?: string
  linkName?: string
  title?: string
  format?: string
  mediaKind?: string
  fileSize?: number
  uploadedAt?: string
  updatedAt?: string
  categoryName?: string
}

const props = withDefaults(
  defineProps<{
    open: boolean
    payload: GalleryFileDetailPayload | null
    navCurrent?: number
    navTotal?: number
    /** 工具栏交互中（如删除确认）时保持底部工具栏可见 */
    toolbarPinned?: boolean
  }>(),
  {
    toolbarPinned: false,
  },
)

const emit = defineEmits<{
  close: []
  download: []
  'copy-link': []
  navigate: [direction: 'prev' | 'next']
}>()

const messageStore = useMessageStore()
const { showPrevNav, showNextNav, showBottomBar, onViewerMouseMove, onViewerMouseLeave } =
  useViewerEdgeNav()
const toolbarPinned = computed(() => props.toolbarPinned === true)

const navEligible = computed(() => (props.navTotal ?? 0) > 1)
const canPrev = computed(() => navEligible.value && (props.navCurrent ?? 0) > 1)
const canNext = computed(() => navEligible.value && (props.navCurrent ?? 0) < (props.navTotal ?? 0))

const WHEEL_TRIGGER_DELTA = 48
let wheelDeltaAccum = 0

function resetWheelAccum() {
  wheelDeltaAccum = 0
}

function emitNavigate(direction: 'prev' | 'next') {
  if (direction === 'prev' && !canPrev.value) return
  if (direction === 'next' && !canNext.value) return
  emit('navigate', direction)
}

function onNavLayerMouseMove(event: MouseEvent) {
  if (!props.open) return
  onViewerMouseMove(event)
}

function onWheel(event: WheelEvent) {
  if (!props.open || !navEligible.value) return
  event.preventDefault()
  wheelDeltaAccum += event.deltaY
  if (Math.abs(wheelDeltaAccum) < WHEEL_TRIGGER_DELTA) return
  const goingNext = wheelDeltaAccum > 0
  resetWheelAccum()
  emitNavigate(goingNext ? 'next' : 'prev')
}

function onKeydown(e: KeyboardEvent) {
  if (!props.open || !navEligible.value) return
  if (e.key === 'ArrowLeft') emitNavigate('prev')
  else if (e.key === 'ArrowRight') emitNavigate('next')
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      resetWheelAccum()
      window.addEventListener('keydown', onKeydown)
      document.addEventListener('wheel', onWheel, { passive: false })
    } else {
      window.removeEventListener('keydown', onKeydown)
      document.removeEventListener('wheel', onWheel)
      onViewerMouseLeave()
      resetWheelAccum()
    }
  },
)

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  document.removeEventListener('wheel', onWheel)
})

const displayName = computed(
  () => props.payload?.linkName?.trim() || props.payload?.title?.trim() || '',
)

function formatDate(iso?: string): string {
  const raw = iso?.trim()
  if (!raw) return '—'
  const t = Date.parse(raw)
  if (Number.isNaN(t)) return raw
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

const detailRows = computed(() => {
  const p = props.payload
  if (!p) return []
  const fmt = (p.format || galleryExtFromUrl(p.fullSrc)).toUpperCase() || '—'
  const kind = p.mediaKind || galleryMediaKindFromUrl(p.fullSrc)
  const original = toAbsoluteResourceUrl(p.originalUrl || p.fullSrc)
  const short =
    (p.shortUrl ? toAbsoluteResourceUrl(p.shortUrl) : '') ||
    (p.linkName ? buildShortLinkPreview(linkNameStem(p.linkName)) : '')
  return [
    { label: '分类', value: p.categoryName?.trim() || '—', copyable: false },
    { label: '类型', value: galleryMediaKindLabel(kind), copyable: false },
    { label: '格式', value: fmt, copyable: false },
    { label: '大小', value: formatFileSize(p.fileSize), copyable: false },
    { label: '上传时间', value: formatDate(p.uploadedAt), copyable: false },
    { label: '更新时间', value: formatDate(p.updatedAt), copyable: false },
    { label: '原版链接', value: original || '—', copyable: original !== '—' && original !== '' },
    { label: '短链接', value: short || '—', copyable: short !== '—' && short !== '' },
  ]
})

function emitClose() {
  emit('close')
}

function openFile() {
  const url = props.payload?.fullSrc?.trim()
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

async function copyValue(value: string) {
  const ok = await copyTextToClipboard(value)
  messageStore.show(ok ? '已复制' : '复制失败', ok ? 'success' : 'error')
}
</script>

<style scoped lang="scss">
.file-detail {
  padding: 8px 20px 20px;
}

.file-detail__grid {
  display: grid;
  gap: 8px;
  margin-top: 8px;
  padding: 14px;
  border-radius: 6px;
  background: #f5f5f7;
}

.file-detail__row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  font-size: 12px;
  line-height: 1.45;
}

.file-detail__label {
  color: #888;
}

.file-detail__value {
  margin: 0;
  padding: 0;
  border: none;
  background: none;
  color: #1d1d1f;
  text-align: left;
  word-break: break-all;
}

.file-detail__value--copy {
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;

  &:hover {
    color: #000;
  }
}

.file-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}

.file-detail-nav-layer {
  position: fixed;
  inset: 0;
  z-index: 2900;
  pointer-events: none;
}
</style>
