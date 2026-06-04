<template>
  <Teleport to="body">
    <Transition name="viewer-mask">
      <div v-if="maskVisible" class="viewer-overlay" aria-hidden="true" />
    </Transition>

    <div
      v-if="internalVisible"
      class="viewer-root"
      @mousemove="onViewerMouseMove"
      @mouseleave="onViewerRootMouseLeave"
      @wheel.prevent="onViewerWheel"
      @touchstart.passive="onViewerTouchStart"
      @touchend="onViewerTouchEnd"
    >
      <div class="viewer-fly" @click="onViewerStageClick">
        <div ref="flyFrameRef" class="viewer-fly__frame" :style="flyFrameStyle">
          <video
            ref="videoRef"
            class="viewer-fly__video"
            :src="currentVideo"
            playsinline
            preload="metadata"
            controls
            @loadedmetadata="onVideoMetadata"
            @click.stop
          />
        </div>
      </div>

      <Transition name="viewer-ui">
        <ViewerChrome
          v-if="chromeVisible"
          :show-prev="showPrevBtn"
          :show-next="showNextBtn"
          :show-prev-highlight="showPrevNav"
          :show-next-highlight="showNextNav"
          :show-bottom-highlight="showBottomBar || toolbarPinned"
          :current="displayCurrent"
          :total="displayTotal"
          @close="close"
          badge-label="视频"
          @prev="prevVideo"
          @next="nextVideo"
        >
          <template #badge>
            <img :src="iconVideo" alt="" draggable="false" />
          </template>
          <template v-if="$slots.toolbar" #toolbar>
            <slot name="toolbar" />
          </template>
        </ViewerChrome>
      </Transition>

      <ViewerLoading :visible="loadingVisible" />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted, onMounted, nextTick } from 'vue'
import type { CSSProperties } from 'vue'
import iconVideo from '@/assets/icon/video.svg?url'
import ViewerChrome from './shared/ViewerChrome.vue'
import ViewerLoading from './shared/ViewerLoading.vue'
import { useViewerEdgeNav } from './shared/useViewerEdgeNav'
import { lockBodyScroll, unlockBodyScroll } from '@/utils/bodyScrollLock'

const props = defineProps<{
  visible: boolean
  videos: string[]
  itemIds?: string[]
  initialIndex?: number
  fromRect?: DOMRect | null
  /** 画廊统一排序下的 1-based 序号（跨图片/视频/文件） */
  navCurrent?: number
  /** 画廊统一排序下的总项数 */
  navTotal?: number
  /** 跨类型切换时跳过缩回缩略图的关闭动画 */
  skipExitAnimation?: boolean
  /** 工具栏交互中（如删除确认）时保持底部工具栏可见 */
  toolbarPinned?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'close'): void
  (e: 'edge-navigate', direction: 'prev' | 'next', anchorItemId?: string): void
}>()

const { showPrevNav, showNextNav, showBottomBar, onViewerMouseMove, onViewerMouseLeave } =
  useViewerEdgeNav()
const toolbarPinned = computed(() => props.toolbarPinned === true)
const internalVisible = ref(false)
const maskVisible = ref(false)
const chromeVisible = ref(false)
const currentIndex = ref(0)
const loadingVisible = ref(false)
const videoRef = ref<HTMLVideoElement | null>(null)
const flyFrameRef = ref<HTMLElement | null>(null)

let savedFromRect: DOMRect | null = null
let frameRect: { top: number; left: number; width: number; height: number } | null = null

let viewerGeneration = 0
let resizeTimer: ReturnType<typeof setTimeout> | null = null

let openWindowWidth = 0
let openWindowHeight = 0
let hasWindowSizeChanged = false

let scrollLockedByViewer = false
let touchStartX = 0
let touchStartY = 0
let wheelDeltaAccum = 0
let switching = false

const WHEEL_TRIGGER_DELTA = 48

const DURATION_MS = 320
const EASING = 'cubic-bezier(0.22, 1, 0.36, 1)'
const naturalSizeCache = new Map<string, { w: number; h: number }>()

function lockScroll() {
  if (scrollLockedByViewer) return
  lockBodyScroll()
  scrollLockedByViewer = true
}

function unlockScroll() {
  if (!scrollLockedByViewer) return
  unlockBodyScroll()
  scrollLockedByViewer = false
}

function resetWheelAccum() {
  wheelDeltaAccum = 0
}

function onViewerRootMouseLeave() {
  onViewerMouseLeave()
  resetWheelAccum()
}

function onViewerWheel(event: WheelEvent) {
  if (!internalVisible.value || switching) return
  if (!navEligible.value) return

  wheelDeltaAccum += event.deltaY
  if (Math.abs(wheelDeltaAccum) < WHEEL_TRIGGER_DELTA) return

  const goingNext = wheelDeltaAccum > 0
  resetWheelAccum()

  if (goingNext) nextVideo()
  else prevVideo()
}

function onViewerTouchStart(event: TouchEvent) {
  const touch = event.touches[0]
  if (!touch) return
  touchStartX = touch.clientX
  touchStartY = touch.clientY
}

function onViewerTouchEnd(event: TouchEvent) {
  if (!internalVisible.value || switching) return
  if (!navEligible.value) return
  const touch = event.changedTouches[0]
  if (!touch) return
  const dx = touch.clientX - touchStartX
  const dy = touch.clientY - touchStartY
  if (Math.abs(dx) < 48 || Math.abs(dx) < Math.abs(dy)) return
  if (dx < 0) nextVideo()
  else prevVideo()
}

function computeContainRect(
  naturalW: number,
  naturalH: number,
  vw: number,
  vh: number,
): { top: number; left: number; width: number; height: number } {
  if (naturalW <= 0 || naturalH <= 0) {
    const w = Math.min(vw, vh * 1.78)
    const h = w / 1.78
    return {
      left: (vw - w) / 2,
      top: (vh - h) / 2,
      width: w,
      height: h,
    }
  }
  const ratio = naturalW / naturalH
  let w = vw
  let h = w / ratio
  if (h > vh) {
    h = vh
    w = h * ratio
  }
  return {
    left: (vw - w) / 2,
    top: (vh - h) / 2,
    width: w,
    height: h,
  }
}

function rectFromVideoEl(
  el: HTMLVideoElement | null,
  src: string,
): { top: number; left: number; width: number; height: number } {
  const vw = window.innerWidth
  const vh = window.innerHeight
  let nw = 0
  let nh = 0
  if (el && el.videoWidth > 0) {
    nw = el.videoWidth
    nh = el.videoHeight
  } else if (src && naturalSizeCache.has(src)) {
    const cached = naturalSizeCache.get(src)!
    nw = cached.w
    nh = cached.h
  }
  return computeContainRect(nw, nh, vw, vh)
}

async function waitForVideoSize(el: HTMLVideoElement | null, maxMs = 400) {
  const t0 = performance.now()
  while (el && el.videoWidth === 0 && performance.now() - t0 < maxMs) {
    await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
  }
}

const flyFrameStyle = ref<CSSProperties>({
  position: 'fixed',
  left: '0px',
  top: '0px',
  width: '0px',
  height: '0px',
  transform: 'translate3d(0,0,0)',
  transformOrigin: 'top left',
  transition: 'none',
  willChange: 'auto',
})

const nextFrame = () => new Promise<void>((r) => requestAnimationFrame(() => r()))

async function runEnterFlip(gen: number) {
  if (!savedFromRect) return

  const fx = savedFromRect.left
  const fy = savedFromRect.top
  const fw = savedFromRect.width
  const fh = savedFromRect.height

  const videoEl = videoRef.value
  const src = props.videos[currentIndex.value] ?? ''
  await waitForVideoSize(videoEl)
  if (gen !== viewerGeneration) return

  frameRect = rectFromVideoEl(videoEl, src)
  const { left: tx, top: ty, width: tw, height: th } = frameRect
  const twSafe = Math.max(tw, 1)
  const thSafe = Math.max(th, 1)
  const sx = fw / twSafe
  const sy = fh / thSafe

  flyFrameStyle.value = {
    position: 'fixed',
    left: `${tx}px`,
    top: `${ty}px`,
    width: `${tw}px`,
    height: `${th}px`,
    transform: `translate(${fx - tx}px, ${fy - ty}px) scale(${sx}, ${sy})`,
    transformOrigin: 'top left',
    transition: 'none',
    willChange: 'transform',
  }

  await nextTick()
  await nextFrame()
  await nextFrame()
  if (gen !== viewerGeneration) return

  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    transform: 'translate3d(0,0,0) scale(1, 1)',
    transition: `transform ${DURATION_MS}ms ${EASING}`,
  }

  await new Promise<void>((r) => setTimeout(r, DURATION_MS + 40))
  if (gen !== viewerGeneration) return

  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    transition: 'none',
    willChange: 'auto',
  }
}

async function runExitFlip(): Promise<void> {
  if (!savedFromRect || !frameRect) return

  const fx = savedFromRect.left
  const fy = savedFromRect.top
  const fw = savedFromRect.width
  const fh = savedFromRect.height
  const { left: tx, top: ty, width: tw, height: th } = frameRect
  const twSafe = Math.max(tw, 1)
  const thSafe = Math.max(th, 1)
  const sx = fw / twSafe
  const sy = fh / thSafe

  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    left: `${tx}px`,
    top: `${ty}px`,
    width: `${tw}px`,
    height: `${th}px`,
    transform: 'translate3d(0,0,0) scale(1,1)',
    opacity: 1,
    transition: 'none',
    willChange: 'transform, opacity',
  }

  await nextFrame()

  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    transform: `translate(${fx - tx}px, ${fy - ty}px) scale(${sx}, ${sy})`,
    opacity: 0,
    transition: `transform ${DURATION_MS}ms ${EASING}, opacity ${DURATION_MS}ms ${EASING}`,
  }

  await new Promise<void>((r) => setTimeout(r, DURATION_MS + 40))
}

async function runExitFade() {
  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    transition: 'none',
    opacity: 1,
  }
  await nextFrame()
  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    opacity: 0,
    transition: `opacity ${220}ms ease-out`,
  }
  await new Promise<void>((r) => setTimeout(r, 260))
}

async function enterFallbackStatic(gen: number) {
  const src = props.videos[currentIndex.value] ?? ''
  await waitForVideoSize(videoRef.value)
  if (gen !== viewerGeneration) return
  frameRect = rectFromVideoEl(videoRef.value, src)
  const { left, top, width, height } = frameRect
  flyFrameStyle.value = {
    position: 'fixed',
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`,
    transform: 'scale(0.96)',
    transformOrigin: 'center center',
    opacity: 0,
    transition: 'none',
    willChange: 'transform, opacity',
  }
  await nextTick()
  await nextFrame()
  if (gen !== viewerGeneration) return
  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    transform: 'scale(1)',
    opacity: 1,
    transition: `transform ${DURATION_MS}ms ${EASING}, opacity ${220}ms ease-out`,
    willChange: 'transform, opacity',
  }
  await new Promise<void>((r) => setTimeout(r, DURATION_MS + 40))
  if (gen !== viewerGeneration) return
  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    transition: 'none',
    willChange: 'auto',
    opacity: 1,
  }
}

function resetFlyFrameForOpen() {
  flyFrameStyle.value = {
    position: 'fixed',
    left: '0px',
    top: '0px',
    width: '0px',
    height: '0px',
    transform: 'translate3d(0,0,0)',
    transformOrigin: 'top left',
    transition: 'none',
    opacity: 1,
    willChange: 'auto',
  }
}

function pauseCurrentVideo() {
  const video = videoRef.value
  if (!video) return
  video.pause()
  video.currentTime = 0
}

async function playCurrentVideo() {
  const video = videoRef.value
  if (!video) return
  try {
    await video.play()
  } catch {
    /* 用户手势或策略限制 */
  }
}

function onViewerStageClick(event: MouseEvent) {
  if (event.button !== 0) return
  const video = videoRef.value
  if (!video || !video.paused) return
  void playCurrentVideo()
}

function applyFrameLayoutForCurrent() {
  const src = props.videos[currentIndex.value] ?? ''
  frameRect = rectFromVideoEl(videoRef.value, src)
  const r = frameRect
  flyFrameStyle.value = {
    ...flyFrameStyle.value,
    left: `${r.left}px`,
    top: `${r.top}px`,
    width: `${r.width}px`,
    height: `${r.height}px`,
    transition: 'none',
  }
}

async function switchTo(newIndex: number) {
  if (switching || currentIndex.value === newIndex) return
  switching = true
  pauseCurrentVideo()
  loadingVisible.value = true
  currentIndex.value = newIndex
  await nextTick()
  await waitForVideoSize(videoRef.value)
  applyFrameLayoutForCurrent()
  loadingVisible.value = false
  switching = false
}

function emitEdgeNavigate(direction: 'prev' | 'next') {
  emit('edge-navigate', direction, props.itemIds?.[currentIndex.value])
}

function prevVideo() {
  if (currentIndex.value <= 0) {
    emitEdgeNavigate('prev')
    return
  }
  void switchTo(currentIndex.value - 1)
}

function nextVideo() {
  if (currentIndex.value >= props.videos.length - 1) {
    emitEdgeNavigate('next')
    return
  }
  void switchTo(currentIndex.value + 1)
}

function close() {
  emit('update:visible', false)
}

function onVideoMetadata() {
  const el = videoRef.value
  const src = currentVideo.value
  if (el && src && el.videoWidth > 0) {
    naturalSizeCache.set(src, { w: el.videoWidth, h: el.videoHeight })
  }
  loadingVisible.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (!internalVisible.value || switching) return
  if (e.key === 'ArrowLeft') prevVideo()
  else if (e.key === 'ArrowRight') nextVideo()
  else if (e.key === 'Escape') close()
}

watch(internalVisible, (open) => {
  if (open) window.addEventListener('keydown', onKeydown)
  else window.removeEventListener('keydown', onKeydown)
})

function handleResize() {
  if (!internalVisible.value || !props.visible) return
  if (window.innerWidth !== openWindowWidth || window.innerHeight !== openWindowHeight) {
    hasWindowSizeChanged = true
  }
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    applyFrameLayoutForCurrent()
  }, 120)
}

onMounted(() => window.addEventListener('resize', handleResize))
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('keydown', onKeydown)
  if (resizeTimer) clearTimeout(resizeTimer)
  unlockScroll()
})

watch(
  () => props.visible,
  async (vis) => {
    if (vis && !internalVisible.value) {
      viewerGeneration++
      const gen = viewerGeneration

      frameRect = null
      hasWindowSizeChanged = false
      openWindowWidth = window.innerWidth
      openWindowHeight = window.innerHeight

      savedFromRect = props.fromRect
        ? new DOMRect(
            props.fromRect.left,
            props.fromRect.top,
            props.fromRect.width,
            props.fromRect.height,
          )
        : null

      const idx = Math.max(0, Math.min(props.initialIndex ?? 0, props.videos.length - 1))
      currentIndex.value = idx

      loadingVisible.value = true
      chromeVisible.value = false
      maskVisible.value = false
      switching = false
      resetFlyFrameForOpen()

      internalVisible.value = true
      lockScroll()

      await nextTick()
      pauseCurrentVideo()
      maskVisible.value = true
      chromeVisible.value = true

      const validFrom =
        savedFromRect &&
        savedFromRect.width > 0 &&
        savedFromRect.height > 0 &&
        savedFromRect.top >= -window.innerHeight &&
        savedFromRect.top <= window.innerHeight * 2 &&
        savedFromRect.left >= -window.innerWidth &&
        savedFromRect.left <= window.innerWidth * 2

      if (validFrom && savedFromRect) {
        void runEnterFlip(gen)
      } else {
        void enterFallbackStatic(gen)
      }
    } else if (!vis && internalVisible.value) {
      viewerGeneration++
      chromeVisible.value = false
      maskVisible.value = false
      loadingVisible.value = false
      switching = true

      if (props.skipExitAnimation) {
        internalVisible.value = false
        switching = false
        pauseCurrentVideo()
        unlockScroll()
        emit('update:visible', false)
        emit('close')
        return
      }

      await nextTick()
      await nextFrame()

      pauseCurrentVideo()

      const curSrc = props.videos[currentIndex.value] ?? ''
      frameRect = rectFromVideoEl(videoRef.value, curSrc)
      const r = frameRect
      flyFrameStyle.value = {
        position: 'fixed',
        left: `${r.left}px`,
        top: `${r.top}px`,
        width: `${r.width}px`,
        height: `${r.height}px`,
        transform: 'translate3d(0,0,0)',
        transformOrigin: 'top left',
        opacity: 1,
        transition: 'none',
        willChange: 'auto',
      }

      const canFlyExit =
        savedFromRect &&
        savedFromRect.width > 0 &&
        savedFromRect.height > 0 &&
        !hasWindowSizeChanged &&
        frameRect

      if (canFlyExit) {
        await runExitFlip()
      } else {
        await runExitFade()
      }

      internalVisible.value = false
      switching = false
      unlockScroll()
      emit('update:visible', false)
      emit('close')
    }
  },
  { immediate: true },
)

watch(currentIndex, () => {
  pauseCurrentVideo()
})

watch(
  () => props.videos[0] ?? '',
  async (newSrc, oldSrc) => {
    if (!internalVisible.value || !props.visible) return
    if (!newSrc || !oldSrc || newSrc === oldSrc) return
    switching = true
    pauseCurrentVideo()
    loadingVisible.value = true
    await nextTick()
    await waitForVideoSize(videoRef.value)
    applyFrameLayoutForCurrent()
    loadingVisible.value = false
    switching = false
  },
)

const displayCurrent = computed(() => props.navCurrent ?? currentIndex.value + 1)
const displayTotal = computed(() => props.navTotal ?? props.videos.length)
const navEligible = computed(() => displayTotal.value > 1)
const showPrevBtn = computed(() => navEligible.value && displayCurrent.value > 1)
const showNextBtn = computed(() => navEligible.value && displayCurrent.value < displayTotal.value)
const currentVideo = computed(() => props.videos[currentIndex.value] || '')
</script>

<style scoped lang="scss">
@use './shared/viewer-shell.scss';

.viewer-fly {
  pointer-events: auto;
  cursor: default;
}

.viewer-fly__frame {
  position: relative;
  overflow: hidden;
}

.viewer-fly__video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}

</style>
