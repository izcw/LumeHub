<template>
  <div class="card-wrapper">
    <div
      ref="cardRef"
      class="card"
      :class="{
        'card--lite': liteMode,
        'card--fixed-landscape-43': fixedLandscape43,
        'card--file-kind': isFileKind,
        'card--video-kind': isVideoKind,
        'is-tilting': isTilting,
      }"
      :style="{ borderRadius: `${cardBorderRadius}px` }"
      @mouseenter="onCardMouseEnter"
      @mousemove.passive="onMove"
      @mouseleave="onCardMouseLeave"
    >
      <div
        class="image-container"
        :class="{
          'is-image-ready': imageSlotReady,
          'is-load-failed': mediaLoadFailed,
          'is-lite': liteMode,
          'is-file-kind': isFileKind,
          'is-video-kind': isVideoKind,
        }"
      >
        <div
          class="image-slot"
          :class="{
            'has-media': mediaSlotReady,
            'is-load-failed': mediaLoadFailed,
            'is-file-placeholder': isFileKind,
            'is-video-slot': isVideoKind,
          }"
          :style="videoSlotStyle"
        >
          <template v-if="isFileKind">
            <div class="file-type-panel" :style="{ backgroundColor: fileTheme.bg }">
              <img
                class="file-type-icon-img"
                :src="fileIconUrl"
                alt=""
                draggable="false"
              />
            </div>
          </template>
          <template v-else-if="isVideoKind">
            <img
              v-if="videoPosterSrc"
              ref="imgRef"
              :src="videoPosterSrc"
              class="image video-poster"
              :class="{ 'is-video-hidden': cardVideoPlaying }"
              decoding="async"
              loading="lazy"
              alt=""
              draggable="false"
              @load="onVideoPosterLoad"
              @error="onVideoPosterError"
            />
            <video
              ref="cardVideoRef"
              class="card-video"
              :class="{ 'is-playing': cardVideoPlaying, 'is-standalone': !videoPosterSrc }"
              :src="videoPlaySrc"
              muted
              playsinline
              loop
              :preload="videoPreload"
              @loadedmetadata="onCardVideoReady"
              @loadeddata="onCardVideoReady"
              @error="onCardVideoError"
            />
          </template>
          <template v-else>
            <img
              ref="imgRef"
              :src="renderSrc || undefined"
              class="image"
              :class="{ 'is-live-hidden': liveVideoPlaying }"
              decoding="async"
              loading="lazy"
              @load="onImgLoad"
              @error="onImgError"
            />
            <video
              v-if="isLivePhotoActive"
              ref="liveVideoRef"
              class="live-video"
              :class="{ 'is-playing': liveVideoPlaying }"
              :src="liveVideoUrlResolved"
              muted
              playsinline
              loop
              :preload="videoPreload"
            />
            <!-- Apple 风格磨砂材质：vibrancy + 细腻颗粒 + 发丝高光边 -->
            <div v-if="!liteMode && !mediaLoadFailed" class="glass-overlay" aria-hidden="true">
              <span class="glass-overlay__grain"></span>
              <span class="glass-overlay__sheen"></span>
            </div>
            <div v-if="mediaLoadFailed" class="media-load-failed" role="status">
              图片加载失败
            </div>
          </template>
        </div>

        <div v-if="isLivePhotoActive" class="card-live-badge" aria-label="实况图">
          <img :src="iconLive" alt="" draggable="false" />
        </div>
        <div v-if="isVideoKind" class="card-video-badge" aria-label="视频">
          <img :src="iconVideo" alt="" draggable="false" />
        </div>

        <!-- 底部：默认右下日期；悬停时同高度显示工具栏（左侧大小/格式，右侧操作）并隐藏日期 -->
        <div v-if="date || canShowToolbar" class="card-footer">
          <Transition name="card-actions-fade">
            <PictureCardToolbar
              v-if="showToolbar"
              :src="src"
              :viewer-full-src="viewerFullSrc"
              :file-size="fileSize"
              :format="format"
              :show-restore="showRestore"
              :show-transfer="showTransfer"
              :show-admin-actions="showAdminActions"
              :delete-only="deleteOnly"
              :delete-confirm-title="deleteConfirmTitle"
              :view-href="viewHref"
              :surface="toolbarSurface"
              @restore="emit('restore')"
              @transfer="emit('transfer')"
              @delete="emit('delete')"
              @edit="emit('edit')"
              @download="emit('download')"
              @copy-link="emit('copy-link')"
              @pin-change="deleteConfirmOpen = $event"
            />
          </Transition>
          <div
            v-if="date"
            class="date-tag"
            :class="{ 'date-tag--hidden-by-toolbar': showToolbar }"
          >
            {{ date }}
          </div>
        </div>

        <div v-if="!liteMode" ref="glowRef" class="glow-layer"></div>
        <div v-if="!liteMode" class="ambient-reflection"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted, computed } from 'vue'
import { warmViewerImage } from '@/utils/viewerImageWarm'
import {
  galleryExtFromUrl,
  galleryMediaKindFromUrl,
  type GalleryMediaKind,
} from '@/utils/galleryMedia'
import { formatFileSize } from '@/utils/formatFileSize'

import iconCompress from '@/assets/icon/Compress.svg?url'
import iconVideo from '@/assets/icon/video.svg?url'
import iconAudio from '@/assets/icon/Audio.svg?url'
import iconFile from '@/assets/icon/File.svg?url'
import iconMarkdown from '@/assets/icon/markdown.svg?url'
import iconOther from '@/assets/icon/other.svg?url'
import iconTransfer from '@/assets/icon/transfer.svg?url'
import iconLive from '@/assets/icon/live.svg?url'
import PictureCardToolbar from '@/components/gallery/PictureCardToolbar.vue'

const emit = defineEmits<{
  load: []
  /** 视频元数据就绪：高/宽，供瀑布流占位 */
  'aspect-hw': [hw: number]
  error: []
  delete: []
  restore: []
  edit: []
  transfer: []
  download: []
  'copy-link': []
  view: []
}>()

const props = defineProps<{
  src: string
  /** 查看器用原图 URL：列表仅显示 src（多为缩略图），成功后后台预取此地址以便点开大图更快 */
  viewerFullSrc?: string
  date?: string
  lite?: boolean
  /** 网格等场景：固定横向 4:3 裁切，不随原图比例撑开 */
  fixedLandscape43?: boolean
  /** 登录后显示卡片管理操作 */
  showAdminActions?: boolean
  /** 仅显示删除操作（回收站等场景） */
  deleteOnly?: boolean
  /** 显示恢复操作（回收站） */
  showRestore?: boolean
  /** 显示转移到同一大分类下其他画廊 */
  showTransfer?: boolean
  /** 删除确认文案 */
  deleteConfirmTitle?: string
  /** 新标签页查看地址 */
  viewHref?: string
  fileSize?: number
  format?: string
  mediaKind?: string
  isLivePhoto?: boolean
  liveVideoUrl?: string
}>()

const fixedLandscape43 = computed(() => props.fixedLandscape43 === true)

const mediaKind = computed<GalleryMediaKind>(() => {
  const declared = props.mediaKind?.trim().toLowerCase()
  if (
    declared === 'video' ||
    declared === 'audio' ||
    declared === 'archive' ||
    declared === 'document' ||
    declared === 'other'
  ) {
    return declared
  }
  const fmt = props.format?.trim().toLowerCase()
  if (fmt) {
    const fromFmt = galleryMediaKindFromUrl(`file.${fmt}`)
    if (fromFmt !== 'other' || fmt === 'jpeg') return fromFmt
  }
  const full = props.viewerFullSrc?.trim() ?? ''
  if (full) {
    const k = galleryMediaKindFromUrl(full)
    if (k !== 'other') return k
  }
  return galleryMediaKindFromUrl(props.src)
})
const isVideoKind = computed(() => mediaKind.value === 'video')
const isFileKind = computed(() => mediaKind.value !== 'image' && !isVideoKind.value)

const FILE_PALETTE: Record<Exclude<GalleryMediaKind, 'image'>, { bg: string }> = {
  archive: { bg: '#ffd766' },
  video: { bg: '#dde8ff' },
  audio: { bg: '#ede9fe' },
  document: { bg: '#fff2e8' },
  other: { bg: '#eef1f5' },
}

const fileTheme = computed(() => {
  const k = mediaKind.value
  if (k === 'image') return FILE_PALETTE.other
  return FILE_PALETTE[k]
})

const fileIconUrl = computed(() => {
  const k = mediaKind.value
  const ext = galleryExtFromUrl(props.src)
  if (k === 'document' && ext === 'md') return iconMarkdown
  if (k === 'archive') return iconCompress
  if (k === 'video') return iconVideo
  if (k === 'audio') return iconAudio
  if (k === 'document') return iconFile
  return iconOther
})

/** 文件卡无「从毛玻璃揭晓」过程，直接视为就绪 */
const imageSlotReady = computed(() => {
  if (isFileKind.value) return true
  return imageReady.value
})
const mediaSlotReady = computed(() => {
  if (isFileKind.value) return true
  if (isVideoKind.value) return videoReady.value
  return mediaReady.value
})
const videoReady = ref(false)

const isLivePhotoActive = computed(
  () => !isVideoKind.value && props.isLivePhoto === true && Boolean(props.liveVideoUrl?.trim()),
)
const liveVideoUrlResolved = computed(() => props.liveVideoUrl?.trim() ?? '')

const videoPlaySrc = computed(() => {
  const full = props.viewerFullSrc?.trim() ?? ''
  const primary = props.src?.trim() ?? ''
  if (full && galleryMediaKindFromUrl(full) === 'video') return full
  if (primary && galleryMediaKindFromUrl(primary) === 'video') return primary
  return full || primary
})
const videoPosterSrc = computed(() => {
  const primary = props.src?.trim() ?? ''
  if (primary && galleryMediaKindFromUrl(primary) === 'image') return primary
  return ''
})

async function playLiveVideoPreview() {
  if (!isLivePhotoActive.value) return
  const video = liveVideoRef.value
  if (!video) return
  liveVideoPlaying.value = true
  video.currentTime = 0
  try {
    await video.play()
  } catch {
    liveVideoPlaying.value = false
  }
}

function stopLiveVideoPreview() {
  liveVideoPlaying.value = false
  const video = liveVideoRef.value
  if (!video) return
  video.pause()
  video.currentTime = 0
}

async function playCardVideoPreview() {
  if (!isVideoKind.value) return
  const video = cardVideoRef.value
  if (!video) return
  cardVideoPlaying.value = true
  video.currentTime = 0
  try {
    await video.play()
  } catch {
    cardVideoPlaying.value = false
  }
}

function stopCardVideoPreview() {
  cardVideoPlaying.value = false
  const video = cardVideoRef.value
  if (!video) return
  video.pause()
  video.currentTime = 0
}

const imgRef = ref<HTMLImageElement | null>(null)
const liveVideoRef = ref<HTMLVideoElement | null>(null)
const cardVideoRef = ref<HTMLVideoElement | null>(null)
const liveVideoPlaying = ref(false)
const cardVideoPlaying = ref(false)
/** CSS aspect-ratio：宽/高，默认竖屏 9:16 */
const cardVideoAspectRatio = ref('9 / 16')
const renderSrc = ref('')

const videoSlotStyle = computed(() => {
  if (!isVideoKind.value || fixedLandscape43.value) return undefined
  return { aspectRatio: cardVideoAspectRatio.value }
})

function applyVideoAspectFromSize(width: number, height: number) {
  if (width <= 0 || height <= 0) return
  const nextCss = `${width} / ${height}`
  if (cardVideoAspectRatio.value !== nextCss) {
    cardVideoAspectRatio.value = nextCss
  }
  emit('aspect-hw', height / width)
}

function syncVideoAspectFromElement(el?: HTMLVideoElement | null) {
  const video = el ?? cardVideoRef.value
  if (!video || video.videoWidth <= 0 || video.videoHeight <= 0) return
  applyVideoAspectFromSize(video.videoWidth, video.videoHeight)
}
const imageReady = ref(false)
const mediaLoadFailed = ref(false)
const liteMode = computed(() => props.lite === true)
const videoPreload = computed(() => (liteMode.value ? 'none' : 'metadata'))

/** 解码成功后短暂停留再开始淡出，避免缓存命中时「看不出毛玻璃」 */
const MIN_HOLD_BEFORE_GLASS_FADE_MS = 160

const mediaReady = ref(false)

let loadEmittedForSrc = false
let decodeSeq = 0
let revealGeneration = 0
let revealTimer: ReturnType<typeof setTimeout> | null = null
/** 列表用缩略图失败时，已尝试改用 viewerFullSrc 显示一次 */
let triedViewerFullAsDisplay = false
/** 切换 src 清空 renderSrc 时忽略 img error，避免误判为加载失败 */
let suppressImageErrors = false

function runAfterPaint(fn: () => void) {
  requestAnimationFrame(() => {
    requestAnimationFrame(fn)
  })
}

function clearRevealTimer() {
  if (revealTimer) {
    clearTimeout(revealTimer)
    revealTimer = null
  }
}

function emitLoadForLayoutOnce() {
  if (loadEmittedForSrc) return
  loadEmittedForSrc = true
  emit('load')
}

function emitErrorForLayout() {
  emit('error')
}

/**
 * 仅在图片加载并解码成功后调用：先保持毛玻璃不透明，
 * 再在下一帧挂上 is-image-ready，触发 1.5s 淡出（刷新/缓存命中也会先 paint 再过渡）。
 */
function scheduleGlassFadeAfterSuccessfulLoad() {
  if (imageReady.value || !mediaSlotReady.value) return

  clearRevealTimer()
  revealGeneration++
  const gen = revealGeneration

  revealTimer = window.setTimeout(() => {
    revealTimer = null
    if (gen !== revealGeneration || imageReady.value) return
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (gen !== revealGeneration || imageReady.value) return
        imageReady.value = true
      })
    })
  }, MIN_HOLD_BEFORE_GLASS_FADE_MS)
}

async function finalizeImageReady(success: boolean) {
  if (mediaReady.value) return
  const seq = ++decodeSeq

  const el = imgRef.value
  if (success && el && typeof el.decode === 'function') {
    try {
      await el.decode()
    } catch {
      /* 个别 SVG/损坏帧可跳过 */
    }
  }

  if (seq !== decodeSeq || mediaReady.value) return

  mediaReady.value = true
  if (success) {
    mediaLoadFailed.value = false
    emitLoadForLayoutOnce()
    const thumb = props.src?.trim() ?? ''
    const full = props.viewerFullSrc?.trim() ?? ''
    if (full && full !== thumb && galleryMediaKindFromUrl(full) === 'image') {
      warmViewerImage(full)
    } else if (thumb) {
      warmViewerImage(thumb)
    }
    scheduleGlassFadeAfterSuccessfulLoad()
  } else {
    mediaLoadFailed.value = true
    emitErrorForLayout()
  }
}

const tryFinishFromCache = () => {
  const el = imgRef.value
  if (el?.complete && el.naturalWidth > 0) {
    void finalizeImageReady(true)
  }
}

function onImgLoad() {
  void finalizeImageReady(true)
}

function onImgError() {
  if (suppressImageErrors) return
  const primary = props.src?.trim() ?? ''
  const full = props.viewerFullSrc?.trim() ?? ''
  if (
    !triedViewerFullAsDisplay &&
    full &&
    full !== primary &&
    full !== renderSrc.value &&
    galleryMediaKindFromUrl(full) === 'image'
  ) {
    triedViewerFullAsDisplay = true
    suppressImageErrors = true
    renderSrc.value = full
    imageReady.value = false
    mediaReady.value = false
    mediaLoadFailed.value = false
    loadEmittedForSrc = false
    decodeSeq++
    revealGeneration++
    clearRevealTimer()
    nextTick(() => {
      suppressImageErrors = false
      runAfterPaint(tryFinishFromCache)
    })
    return
  }
  void finalizeImageReady(false)
}

function resetForNewSrc() {
  suppressImageErrors = true
  renderSrc.value = ''
  imageReady.value = false
  mediaReady.value = false
  mediaLoadFailed.value = false
  loadEmittedForSrc = false
  triedViewerFullAsDisplay = false
  decodeSeq++
  revealGeneration++
  clearRevealTimer()
}

function revealImageSrc() {
  if (galleryMediaKindFromUrl(props.src) !== 'image') return
  if (renderSrc.value === props.src) {
    suppressImageErrors = false
    return
  }
  /** 确保每张新图都从“毛玻璃可见”状态开始，避免复用节点跳过动画 */
  imageReady.value = false
  mediaReady.value = false
  mediaLoadFailed.value = false
  loadEmittedForSrc = false
  triedViewerFullAsDisplay = false
  decodeSeq++
  revealGeneration++
  clearRevealTimer()
  renderSrc.value = props.src
  nextTick(() => {
    suppressImageErrors = false
    runAfterPaint(tryFinishFromCache)
  })
}

function applyFileKindLayout() {
  clearRevealTimer()
  resetForNewSrc()
  suppressImageErrors = false
  renderSrc.value = ''
  imageReady.value = true
  mediaReady.value = true
  loadEmittedForSrc = false
  nextTick(() => {
    emitLoadForLayoutOnce()
  })
}

function resetVideoCardState() {
  cardVideoPlaying.value = false
  cardVideoAspectRatio.value = '9 / 16'
  videoReady.value = false
  imageReady.value = false
  loadEmittedForSrc = false
}

function markVideoCardReady() {
  if (videoReady.value) return
  clearVideoReadyFallback()
  videoReady.value = true
  imageReady.value = true
  emitLoadForLayoutOnce()
}

function onVideoPosterLoad() {
  const img = imgRef.value
  if (img && img.naturalWidth > 0 && img.naturalHeight > 0) {
    const video = cardVideoRef.value
    if (!video || video.videoWidth <= 0) {
      applyVideoAspectFromSize(img.naturalWidth, img.naturalHeight)
    }
  }
  markVideoCardReady()
}

function onVideoPosterError() {
  markVideoCardReady()
}

function onCardVideoReady() {
  syncVideoAspectFromElement()
  markVideoCardReady()
}

function onCardVideoError() {
  markVideoCardReady()
}

let videoReadyFallbackTimer: ReturnType<typeof setTimeout> | null = null

function clearVideoReadyFallback() {
  if (videoReadyFallbackTimer) {
    clearTimeout(videoReadyFallbackTimer)
    videoReadyFallbackTimer = null
  }
}

function scheduleVideoReadyFallback() {
  clearVideoReadyFallback()
  videoReadyFallbackTimer = setTimeout(() => {
    videoReadyFallbackTimer = null
    markVideoCardReady()
  }, 600)
}

function applyVideoKindLayout() {
  clearRevealTimer()
  clearVideoReadyFallback()
  resetForNewSrc()
  suppressImageErrors = false
  resetVideoCardState()
  renderSrc.value = ''
  scheduleVideoReadyFallback()
  nextTick(() => {
    const el = cardVideoRef.value
    if (el && el.readyState >= 1) {
      syncVideoAspectFromElement(el)
      markVideoCardReady()
      return
    }
    if (videoPosterSrc.value && imgRef.value?.complete && imgRef.value.naturalWidth > 0) {
      markVideoCardReady()
    }
  })
}

function setupMediaForCurrentSrc() {
  const s = props.src?.trim() ?? ''
  const full = props.viewerFullSrc?.trim() ?? ''
  if (!s && !full) {
    resetForNewSrc()
    suppressImageErrors = false
    resetVideoCardState()
    return
  }
  if (isVideoKind.value) {
    applyVideoKindLayout()
    return
  }
  if (galleryMediaKindFromUrl(s) === 'image') {
    resetVideoCardState()
    resetForNewSrc()
    nextTick(() => {
      revealImageSrc()
    })
    return
  }
  resetVideoCardState()
  applyFileKindLayout()
}

watch(
  () => [props.src, props.viewerFullSrc] as const,
  () => {
    nextTick(() => {
      setupMediaForCurrentSrc()
    })
  },
)

onMounted(() => {
  nextTick(() => {
    setupMediaForCurrentSrc()
    if (!liteMode.value) setupCardRadiusObserver()
  })
})

onUnmounted(() => {
  clearRevealTimer()
  clearVideoReadyFallback()
  disconnectCardRadiusObserver()
  cancelMotionFrame()
})

const date = computed(() => (props.date ?? '').trim())
const deleteConfirmTitle = computed(
  () => props.deleteConfirmTitle?.trim() || '确定删除该文件吗？',
)
const deleteOnly = computed(() => props.deleteOnly === true)
const showRestore = computed(() => props.showRestore === true)
const showTransfer = computed(() => props.showTransfer === true)
const showAdminActions = computed(() => props.showAdminActions === true)
const hasPublicToolbarActions = computed(() => !deleteOnly.value)
const canShowToolbar = computed(
  () =>
    hasMetaStack.value ||
    showRestore.value ||
    showAdminActions.value ||
    hasPublicToolbarActions.value,
)
const showToolbar = computed(
  () => canShowToolbar.value && (cardHovered.value || deleteConfirmOpen.value),
)
const viewHref = computed(() => (props.viewHref ?? '').trim())

const metaFormatText = computed(
  () => (props.format || galleryExtFromUrl(props.viewerFullSrc || props.src)).toUpperCase(),
)
const metaSizeText = computed(() => {
  const size = formatFileSize(props.fileSize)
  return size === '—' ? '' : size
})
const hasMetaStack = computed(() => Boolean(metaSizeText.value || metaFormatText.value))
const toolbarSurface = computed<'dark' | 'light'>(() =>
  isFileKind.value ? 'light' : 'dark',
)

const cardHovered = ref(false)
const deleteConfirmOpen = ref(false)

function onCardMouseEnter() {
  cardHovered.value = true
  if (isVideoKind.value) void playCardVideoPreview()
  else void playLiveVideoPreview()
  onEnter()
}

function onCardMouseLeave() {
  cardHovered.value = false
  stopCardVideoPreview()
  stopLiveVideoPreview()
  onLeave()
}

/** 窄卡可用较大倾角；单列宽卡缩小倾角，避免 3D 投影盖住相邻卡片 */
const TILT_ROTATE_MAX = 12
const TILT_ROTATE_MIN = 3
const TILT_WIDTH_NARROW = 160
const TILT_WIDTH_WIDE = 640
const TILT_PERSPECTIVE_MIN = 900
const TILT_PERSPECTIVE_MAX = 2200

function maxRotateForWidth(width: number): number {
  if (!Number.isFinite(width) || width <= 0) return TILT_ROTATE_MAX
  if (width <= TILT_WIDTH_NARROW) return TILT_ROTATE_MAX
  if (width >= TILT_WIDTH_WIDE) return TILT_ROTATE_MIN
  const t = (width - TILT_WIDTH_NARROW) / (TILT_WIDTH_WIDE - TILT_WIDTH_NARROW)
  return TILT_ROTATE_MAX + t * (TILT_ROTATE_MIN - TILT_ROTATE_MAX)
}

function perspectiveForWidth(width: number): number {
  if (!Number.isFinite(width) || width <= 0) return TILT_PERSPECTIVE_MIN
  const scaled = width * 2.1
  return Math.round(Math.max(TILT_PERSPECTIVE_MIN, Math.min(TILT_PERSPECTIVE_MAX, scaled)))
}

/** 随卡片变窄缩小圆角（如手机 6 列），避免相对比例过大显得「过圆」 */
const CARD_RADIUS_MIN = 4
const CARD_RADIUS_MAX = 12
const CARD_RADIUS_WIDTH_RATIO = 0.048
const cardBorderRadius = ref(12)
const cardMaxRotate = ref(TILT_ROTATE_MAX)
const cardPerspective = ref(TILT_PERSPECTIVE_MIN)
const isTilting = ref(false)

function updateCardMetricsFromWidth(width: number) {
  if (!Number.isFinite(width) || width <= 0) return
  const r = Math.round(
    Math.min(CARD_RADIUS_MAX, Math.max(CARD_RADIUS_MIN, width * CARD_RADIUS_WIDTH_RATIO)),
  )
  if (r !== cardBorderRadius.value) cardBorderRadius.value = r
  const nextRotate = maxRotateForWidth(width)
  if (nextRotate !== cardMaxRotate.value) cardMaxRotate.value = nextRotate
  const nextPerspective = perspectiveForWidth(width)
  if (nextPerspective !== cardPerspective.value) cardPerspective.value = nextPerspective
}

let cardResizeObserver: ResizeObserver | null = null

function setupCardRadiusObserver() {
  const el = cardRef.value
  if (!el) return
  updateCardMetricsFromWidth(el.getBoundingClientRect().width)
  if (typeof ResizeObserver === 'undefined') return
  cardResizeObserver = new ResizeObserver((entries) => {
    const w = entries[0]?.contentRect.width
    if (w != null && w > 0) updateCardMetricsFromWidth(w)
  })
  cardResizeObserver.observe(el)
}

function disconnectCardRadiusObserver() {
  cardResizeObserver?.disconnect()
  cardResizeObserver = null
}

const LERP = 0.12
const INERTIA = 0.9

const cardRef = ref<HTMLDivElement>()
const glowRef = ref<HTMLDivElement>()

let rect!: DOMRect
let isHovering = false
let input = { x: 0, y: 0 }
let cur = { rx: 0, ry: 0 }
let target = { ...cur }

let mouseXPercent = 50
let mouseYPercent = 50

let motionRafId: number | null = null

const lerp = (a: number, b: number, t: number) => a + (b - a) * t
const ease = (v: number) => Math.sign(v) * Math.pow(Math.abs(v), 1.3)

function cancelMotionFrame() {
  if (motionRafId !== null) {
    cancelAnimationFrame(motionRafId)
    motionRafId = null
  }
}

function scheduleMotionFrame() {
  if (motionRafId !== null) return
  motionRafId = requestAnimationFrame(runMotionFrame)
}

function runMotionFrame() {
  motionRafId = null

  if (!isHovering) {
    input.x *= INERTIA
    input.y *= INERTIA
  }

  target.rx = input.y * cardMaxRotate.value
  target.ry = -input.x * cardMaxRotate.value

  cur.rx = lerp(cur.rx, target.rx, LERP)
  cur.ry = lerp(cur.ry, target.ry, LERP)

  const IDLE_EPS = 0.025
  const settled =
    !isHovering &&
    Math.abs(input.x) < 1e-4 &&
    Math.abs(input.y) < 1e-4 &&
    Math.abs(cur.rx) < IDLE_EPS &&
    Math.abs(cur.ry) < IDLE_EPS &&
    Math.abs(target.rx) < IDLE_EPS &&
    Math.abs(target.ry) < IDLE_EPS

  if (settled) {
    isTilting.value = false
    if (cardRef.value?.style.transform) {
      cardRef.value.style.transform = ''
    }
    return
  }

  const card = cardRef.value
  const glow = glowRef.value

  if (card) {
    isTilting.value = true
    card.style.transform = `perspective(${cardPerspective.value}px) rotateX(${cur.rx}deg) rotateY(${cur.ry}deg) scale(1)`
  }

  if (isHovering && glow) {
    glow.style.setProperty('--glow-x', `${mouseXPercent}%`)
    glow.style.setProperty('--glow-y', `${mouseYPercent}%`)
    glow.style.opacity = '0.95'
  }

  scheduleMotionFrame()
}

const onEnter = () => {
  if (liteMode.value) return
  const el = cardRef.value
  if (!el) return
  updateCardMetricsFromWidth(el.getBoundingClientRect().width)
  rect = el.getBoundingClientRect()
  isHovering = true
  isTilting.value = true
  scheduleMotionFrame()
}

const onMove = (e: MouseEvent) => {
  if (liteMode.value) return
  const r = rect
  const x = ((e.clientX - r.left) / r.width) * 2 - 1
  const y = ((e.clientY - r.top) / r.height) * 2 - 1
  input.x = ease(x)
  input.y = ease(y)

  mouseXPercent = ((e.clientX - r.left) / r.width) * 100
  mouseYPercent = ((e.clientY - r.top) / r.height) * 100
  scheduleMotionFrame()
}

const onLeave = () => {
  if (liteMode.value) return
  isHovering = false
  input.x = 0
  input.y = 0

  const glow = glowRef.value
  if (glow) {
    glow.style.transition = 'opacity 0.35s ease'
    glow.style.opacity = '0'
    window.setTimeout(() => {
      if (glow.style.opacity === '0') {
        glow.style.removeProperty('--glow-x')
        glow.style.removeProperty('--glow-y')
        glow.style.background = ''
        glow.style.transition = ''
      }
    }, 400)
  }

  scheduleMotionFrame()
}
</script>

<style scoped lang="scss">
/* 与脚本「仅加载成功后才开始淡出」配套；勿单独缩短，否则与过渡脱节 */
$glass-reveal-duration: 3s;

@mixin card-chip-surface-dark {
  color: rgba(255, 255, 255, 0.92);
  background: rgba(0, 0, 0, 0.48);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

@mixin card-chip-surface-light {
  color: rgba(28, 32, 40, 0.88);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.card-wrapper {
  width: 100%;
  height: auto;
}

.card {
  width: 100%;
  height: auto;
  container-type: inline-size;
  // background: #0a0a12;
  box-shadow:
    0 25px 35px -12px rgba(0, 0, 0, 0.3),
    0 0 0 1px rgba(255, 255, 255, 0.05) inset;
  overflow: hidden;
  will-change: auto;
  cursor: pointer;
  transform-style: preserve-3d;
}

.card:not(.card--lite) {
  will-change: transform;
}

.image-container {
  position: relative;
  width: 100%;
  height: auto;
  transform-style: preserve-3d;
  border-radius: inherit;
}

/* 图片与毛玻璃共同的裁剪区域（与卡片圆角一致） */
.image-slot {
  position: relative;
  width: 100%;
  /* 未加载真实像素前瀑布流需要稳定占位高度，否则多项高度≈0 会叠在同一列顶 */
  aspect-ratio: 3 / 4;
  overflow: hidden;
  border-radius: inherit;
  isolation: isolate;
  background: #f5f5f7;
}

/* 解码完成后由图片真实宽高比撑开格子，恢复错落瀑布流（网格 4:3 固定裁切除外） */
.card:not(.card--fixed-landscape-43) .image-slot.has-media:not(.is-file-placeholder):not(.is-video-slot) {
  aspect-ratio: unset;
}

/* 视频为绝对定位层，比例由脚本按真实宽高写入 style.aspectRatio */
.image-slot.is-video-slot {
  /* background: #0a0a0c; */
}

.image-slot.is-video-slot .video-poster,
.image-slot.is-video-slot.has-media .video-poster {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* 非图片：固定横向 4:3，居中类型图标 */
.image-slot.is-file-placeholder {
  aspect-ratio: 4 / 3;
  background: transparent;
}

.image-slot.is-file-placeholder.has-media {
  aspect-ratio: 4 / 3;
}

.file-type-panel {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: inherit;
}

.file-type-icon-img {
  width: min(46%, 128px);
  max-height: min(40%, 104px);
  height: auto;
  object-fit: contain;
  display: block;
  pointer-events: none;
  -webkit-user-drag: none;
}

.image {
  position: absolute;
  inset: 0;
  z-index: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  -webkit-user-drag: none;
  opacity: 0;
  filter: blur(12px);
  animation: none;
}

.card:not(.card--fixed-landscape-43) .image-slot.has-media .image {
  position: relative;
  inset: auto;
  width: 100%;
  height: auto;
  object-fit: unset;
}

/* 固定横向 4:3：占位与出图后比例一致，图片始终 cover 填满 */
.card--fixed-landscape-43 .image-slot {
  aspect-ratio: 4 / 3;
}

.card--fixed-landscape-43 .image-slot.has-media {
  aspect-ratio: 4 / 3;
}

.card--fixed-landscape-43 .image-slot.is-video-slot,
.card--fixed-landscape-43 .image-slot.is-video-slot.has-media {
  aspect-ratio: 4 / 3;
}

.card--fixed-landscape-43 .image-slot .image {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-container:not(.is-image-ready) .image {
  opacity: 1;
  animation: none;
}

/*
 * 仅在脚本置 is-image-ready（加载并 decode 成功后）后触发：
 * 上层毛玻璃 $glass-reveal-duration 渐透明，下图同步从略糊收到清晰。
 */
.image-container.is-image-ready .image {
  opacity: 1;
  animation: pictureRevealFromUnderGlass $glass-reveal-duration cubic-bezier(0.22, 1, 0.36, 1)
    forwards;
}

.image-container.is-lite .image {
  filter: none;
}

.image-container.is-lite.is-image-ready .image {
  opacity: 1;
  animation: none;
  transition: opacity 220ms ease;
}

@keyframes pictureRevealFromUnderGlass {
  from {
    filter: blur(12px) saturate(1.06);
  }
  to {
    filter: blur(0px) saturate(1);
  }
}

/*
 * Apple 官网式磨砂：高斯模糊 + vibrancy（saturation）+ 极淡中性 tint；
 * 发丝顶边光 + 底侧略压暗，颗粒只做细微层次，不做明显网格。
 */
.glass-overlay {
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
  overflow: hidden;
  border-radius: inherit;
  opacity: 1;
  visibility: visible;
  transition: opacity $glass-reveal-duration cubic-bezier(0.22, 1, 0.36, 1);
  /* 材质底色：轻微冷暖过渡，避免一块死白 */
  background:
    radial-gradient(ellipse 95% 65% at 18% 8%, rgba(255, 255, 255, 0.42) 0%, transparent 58%),
    radial-gradient(ellipse 80% 55% at 92% 88%, rgba(120, 145, 175, 0.09) 0%, transparent 55%),
    linear-gradient(
      168deg,
      rgba(255, 255, 255, 0.38) 0%,
      rgba(250, 250, 252, 0.14) 45%,
      rgba(235, 237, 242, 0.22) 100%
    );
  backdrop-filter: blur(28px) saturate(1.72) brightness(1.03);
  -webkit-backdrop-filter: blur(28px) saturate(1.72) brightness(1.03);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.62),
    inset 0 -1px 0 rgba(0, 0, 0, 0.07),
    inset 0 0 0 0.5px rgba(255, 255, 255, 0.18);
}

/* 细腻胶片颗粒（比大块噪声更贴近官网磨砂膜） */
.glass-overlay__grain {
  position: absolute;
  inset: -20%;
  opacity: 0.22;
  mix-blend-mode: overlay;
  pointer-events: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='256' height='256'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.72' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='0.55'/%3E%3C/svg%3E");
  background-size: 128px 128px;
}

/* 顶部柔和镜面高光，增强玻璃厚度感 */
.glass-overlay__sheen {
  position: absolute;
  inset: 0;
  pointer-events: none;
  border-radius: inherit;
  background: linear-gradient(
    178deg,
    rgba(255, 255, 255, 0.38) 0%,
    rgba(255, 255, 255, 0.06) 28%,
    transparent 52%
  );
  opacity: 0.85;
}

.image-container.is-image-ready .glass-overlay {
  opacity: 0;
  visibility: hidden;
  transition:
    opacity $glass-reveal-duration cubic-bezier(0.22, 1, 0.36, 1),
    visibility 0s linear $glass-reveal-duration;
}

.image-container.is-load-failed .image {
  display: none;
}

.media-load-failed {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f0f2;
  color: #8e8e93;
  font-size: 13px;
  line-height: 1.35;
  text-align: center;
  padding: 10px;
  border-radius: inherit;
  user-select: none;
}

/* 底部操作区：与日期同一高度（bottom: 12px） */
.card-footer {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 8px;
  z-index: 5;
  pointer-events: none;

  :deep(.card-toolbar) {
    pointer-events: auto;
  }
}

@media (max-width: 640px) {
  .card-footer :deep(.card-toolbar) {
    display: none;
  }

  .date-tag--hidden-by-toolbar {
    opacity: 1;
    visibility: visible;
  }
}

@container (max-width: 140px) {
  :deep(.card-toolbar) {
    gap: 6px;
    padding: 4px 8px 4px 10px;
  }

  :deep(.card-meta-stack) {
    padding-right: 6px;
  }

  :deep(.card-meta-line) {
    font-size: 8px;
  }

  :deep(.card-toolbar-actions) {
    gap: 1px;
  }

  :deep(.card-action) {
    width: 22px;
    height: 22px;

    img {
      width: 11px;
      height: 11px;
    }
  }

  .date-tag {
    font-size: 10px;
    padding: 3px 6px;
  }
}

@container (max-width: 108px) {
  :deep(.card-toolbar) {
    gap: 5px;
    padding: 3px 6px 3px 8px;
    max-width: calc(100% - 4px);
  }

  :deep(.card-meta-stack) {
    padding-right: 5px;
  }

  :deep(.card-meta-line) {
    font-size: 7px;
    line-height: 1.15;
  }

  :deep(.card-toolbar-actions) {
    gap: 0;
  }

  :deep(.card-action) {
    width: 20px;
    height: 20px;

    img {
      width: 10px;
      height: 10px;
    }
  }
}

.card-actions-fade-enter-active,
.card-actions-fade-leave-active {
  transition:
    opacity 0.22s ease,
    transform 0.22s ease;
}

.card-actions-fade-enter-from,
.card-actions-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}

.card-live-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  z-index: 6;
  width: 22px;
  height: 22px;
  pointer-events: none;

  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
    filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.35));
  }
}

.live-video,
.card-video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 2;
  opacity: 0;
  transition: opacity 0.2s ease;
  pointer-events: none;

  &.is-playing {
    opacity: 1;
  }
}

.card-video.is-standalone {
  opacity: 1;

  &.is-playing {
    opacity: 1;
  }
}

.image.is-live-hidden,
.image.is-video-hidden {
  opacity: 0;
}

.card-video-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  z-index: 6;
  width: 22px;
  height: 22px;
  pointer-events: none;

  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
    filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.35));
  }
}

.image-container.is-video-kind {
  .date-tag {
    @include card-chip-surface-dark;
  }
}

.image-container.is-file-kind {
  .date-tag {
    @include card-chip-surface-light;
  }
}

/* 右下角日期小字 */
.date-tag {
  position: absolute;
  right: 12px;
  bottom: 0;
  font-size: 11px;
  font-weight: 400;
  padding: 4px 8px;
  border-radius: 999px;
  letter-spacing: 0.5px;
  font-family: monospace;
  pointer-events: none;
  z-index: 4;
  transform: translateZ(10px);
  flex-shrink: 0;
  transition:
    opacity 0.35s cubic-bezier(0.22, 1, 0.36, 1),
    visibility 0.35s cubic-bezier(0.22, 1, 0.36, 1),
    color 0.25s ease,
    background 0.25s ease;
  @include card-chip-surface-dark;

  &.date-tag--hidden-by-toolbar {
    opacity: 0;
    visibility: hidden;
  }
}

.image-container:not(.is-image-ready) .date-tag {
  @include card-chip-surface-light;
  color: rgba(90, 96, 110, 0.85);
}

.card--file-kind {
  box-shadow:
    0 12px 24px -10px rgba(0, 0, 0, 0.22),
    0 0 0 1px rgba(0, 0, 0, 0.06) inset;
}

.glow-layer {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  opacity: 0;
  z-index: 3;
  transition: opacity 0.18s ease;
  mix-blend-mode: overlay;
  --glow-x: 50%;
  --glow-y: 50%;
  background: radial-gradient(
    circle at var(--glow-x) var(--glow-y),
    rgba(255, 240, 200, 0.45) 0%,
    rgba(255, 210, 150, 0.25) 30%,
    rgba(255, 255, 255, 0) 80%
  );
  will-change: opacity;
}

.ambient-reflection {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 3;
  background: radial-gradient(
    ellipse at 20% 30%,
    rgba(255, 255, 255, 0.08) 0%,
    rgba(0, 0, 0, 0) 70%
  );
  mix-blend-mode: screen;
}

.card:hover .glow-layer {
  opacity: 0.9;
}

.card:hover .image,
.card:hover .video-poster,
.card:hover .card-video {
  filter: blur(0px) brightness(1.02) contrast(1.03);
}

.card:hover .file-type-panel {
  filter: brightness(1.04) contrast(1.02);
}

.card.card--lite:hover .image {
  filter: none;
}
</style>
