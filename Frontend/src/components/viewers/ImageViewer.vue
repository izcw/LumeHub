<template>
  <Teleport to="body">
    <!-- 蒙层：纯半透明，不做 backdrop-filter（与动画叠加大幅掉帧） -->
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
    >
      <!-- 左右切换：单轨道双视窗，只动画 translateX -->
      <div v-show="slideMode || slideActive" class="viewer-slide">
        <div ref="slideTrackRef" class="viewer-slide__track" :style="slideTrackStyle">
          <div class="viewer-slide__cell">
            <img
              v-if="slideLeftSrc"
              class="viewer-slide__img"
              :src="slideLeftSrc"
              decoding="sync"
              loading="eager"
              draggable="false"
              alt=""
              @load="onSlideImgLoad"
              @error="onSlideImgError"
            />
          </div>
          <div class="viewer-slide__cell">
            <img
              v-if="slideRightSrc"
              class="viewer-slide__img"
              :src="slideRightSrc"
              decoding="sync"
              loading="eager"
              draggable="false"
              alt=""
              @load="onSlideImgLoad"
              @error="onSlideImgError"
            />
          </div>
        </div>
      </div>

      <!-- 仅用于 FLIP 开/关与实况图；日常浏览与切换走 slide 层，避免 handoff 闪屏 -->
      <div
        v-show="!slideMode"
        class="viewer-fly"
        @click="onViewerStageClick"
      >
        <div ref="flyFrameRef" class="viewer-fly__frame" :style="flyFrameStyle">
          <video
            v-if="currentLiveVideoUrl"
            v-show="livePlaying"
            ref="liveVideoRef"
            class="viewer-fly__video"
            :src="currentLiveVideoUrl"
            muted
            playsinline
            loop
            preload="metadata"
            @click.stop
          />
          <img
            ref="imgRef"
            class="viewer-fly__img"
            :class="{ 'is-live-hidden': livePlaying }"
            :src="flyDisplaySrc"
            decoding="async"
            fetchpriority="high"
            loading="eager"
            draggable="false"
            alt=""
            @load="onImageLoad"
            @error="onImageError"
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
          :badge-label="currentLiveVideoUrl ? '实况图' : undefined"
          @prev="prevImage"
          @next="nextImage"
        >
          <template v-if="currentLiveVideoUrl" #badge>
            <img :src="iconLive" alt="" draggable="false" />
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
import iconLive from '@/assets/icon/live.svg?url'
import ViewerChrome from './shared/ViewerChrome.vue'
import ViewerLoading from './shared/ViewerLoading.vue'
import { useViewerEdgeNav } from './shared/useViewerEdgeNav'
import { lockBodyScroll, unlockBodyScroll } from '@/utils/bodyScrollLock'
import { warmAdjacentViewerImages, warmViewerImage, getWarmViewerPromise, getViewerNaturalSize } from '@/utils/viewerImageWarm'

const props = defineProps<{
  visible: boolean
  images: string[]
  liveVideoUrls?: (string | undefined)[]
  itemIds?: string[]
  initialIndex?: number
  fromRect?: DOMRect | null
  /** 画廊统一排序下的 1-based 序号（跨图片/视频/文件） */
  navCurrent?: number
  /** 画廊统一排序下的总项数 */
  navTotal?: number
  /** 画廊内切换上一项/下一项时的滑入方向 */
  slideDirection?: 'left' | 'right'
  /** 跨类型切换时跳过缩回缩略图的关闭动画 */
  skipExitAnimation?: boolean
  /** 工具栏交互中（如删除确认）时保持底部工具栏可见 */
  toolbarPinned?: boolean
  /** 画廊排序中的上一张预览（单图数组时用于滑动 peek） */
  adjacentPrevSrc?: string
  /** 画廊排序中的下一张预览（单图数组时用于滑动 peek） */
  adjacentNextSrc?: string
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
const imgRef = ref<HTMLImageElement | null>(null)
const liveVideoRef = ref<HTMLVideoElement | null>(null)
const livePlaying = ref(false)
const flyFrameRef = ref<HTMLElement | null>(null)
const slideTrackRef = ref<HTMLElement | null>(null)

/** 幻灯：轨道上有两张图；slideMode 为 true 时常驻 slide 层显示，不再切回 fly */
const slideActive = ref(false)
const slideMode = ref(false)
const slideLeftSrc = ref('')
const slideRightSrc = ref('')
const slideTrackOffset = ref(0)
const slideTrackTransition = ref('none')
/** slide 动画期间锁定 fly 层显示的旧图，避免 props 更新抢先闪新图 */
const flyDisplaySrcOverride = ref<string | null>(null)
let savedFromRect: DOMRect | null = null
/** 当前「居中 contain」几何，用于关闭 FLIP 与 resize */
let frameRect: { top: number; left: number; width: number; height: number } | null = null

let viewerGeneration = 0
let slideTransitionGeneration = 0
let resizeTimer: ReturnType<typeof setTimeout> | null = null

let openWindowWidth = 0
let openWindowHeight = 0
let hasWindowSizeChanged = false

let scrollLockedByViewer = false
let touchStartX = 0
let touchStartY = 0
let wheelDeltaAccum = 0
let touchDragging = false
let touchDragMode: 'next' | 'prev' | null = null
let touchAxisLocked: 'x' | 'y' | null = null
let activeTouchId: number | null = null
let touchMoveRaf = 0
let pendingTouchDx = 0
let pendingTouchEdgeNav: 'next' | 'prev' | null = null

const TOUCH_AXIS_THRESHOLD = 10
const TOUCH_COMMIT_RATIO = 0.22
const TOUCH_COMMIT_MIN_PX = 72
const TOUCH_RUBBER_BAND = 0.35

const WHEEL_TRIGGER_DELTA = 48

const DURATION_MS = 320
const EASING = 'cubic-bezier(0.22, 1, 0.36, 1)'
const SLIDE_MS = 340
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
  if (slideActive.value && !slideMode.value) return
  if (!navEligible.value) return

  wheelDeltaAccum += event.deltaY
  if (Math.abs(wheelDeltaAccum) < WHEEL_TRIGGER_DELTA) return

  const goingNext = wheelDeltaAccum > 0
  resetWheelAccum()

  if (goingNext) nextImage()
  else prevImage()
}

function computeContainRect(
  naturalW: number,
  naturalH: number,
  vw: number,
  vh: number,
): { top: number; left: number; width: number; height: number } {
  if (naturalW <= 0 || naturalH <= 0) {
    return { left: 0, top: 0, width: vw, height: vh }
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

function rememberNaturalSize(src: string, w: number, h: number) {
  if (!src || w <= 0 || h <= 0) return
  naturalSizeCache.set(src, { w, h })
}

function viewerSrcMatches(candidate: string, src: string) {
  if (!candidate || !src) return false
  if (candidate === src) return true
  try {
    return candidate.endsWith(src) || src.endsWith(candidate)
  } catch {
    return false
  }
}

function imageElMatchesSrc(el: HTMLImageElement, src: string) {
  const current = el.currentSrc || el.src
  return viewerSrcMatches(current, src)
}

function rectForSrc(src: string, el: HTMLImageElement | null = null) {
  const vw = window.innerWidth
  const vh = window.innerHeight

  const shared = src ? getViewerNaturalSize(src) : undefined
  if (shared) {
    return computeContainRect(shared.w, shared.h, vw, vh)
  }

  if (src && naturalSizeCache.has(src)) {
    const cached = naturalSizeCache.get(src)!
    return computeContainRect(cached.w, cached.h, vw, vh)
  }

  if (el && el.naturalWidth > 0 && imageElMatchesSrc(el, src)) {
    rememberNaturalSize(src, el.naturalWidth, el.naturalHeight)
    return computeContainRect(el.naturalWidth, el.naturalHeight, vw, vh)
  }

  if (src) {
    const probe = new Image()
    probe.src = src
    if (probe.complete && probe.naturalWidth > 0) {
      rememberNaturalSize(src, probe.naturalWidth, probe.naturalHeight)
      return computeContainRect(probe.naturalWidth, probe.naturalHeight, vw, vh)
    }
  }

  return { left: 0, top: 0, width: vw, height: vh }
}

function rectFromImageEl(
  el: HTMLImageElement | null,
  src: string,
): { top: number; left: number; width: number; height: number } {
  return rectForSrc(src, el)
}

function findSlideImgForSrc(targetSrc: string): HTMLImageElement | null {
  const imgs = slideTrackRef.value?.querySelectorAll('.viewer-slide__img')
  if (!imgs) return null
  for (let i = 0; i < imgs.length; i++) {
    const img = imgs.item(i) as HTMLImageElement | null
    if (!img) continue
    if (img.naturalWidth > 0 && viewerSrcMatches(img.currentSrc || img.src, targetSrc)) {
      return img
    }
  }
  return null
}

function applyFlyFrameFromSlideImg(img: HTMLImageElement) {
  const rect = img.getBoundingClientRect()
  const renderedSrc = img.currentSrc || img.src
  frameRect = { left: rect.left, top: rect.top, width: rect.width, height: rect.height }
  if (img.naturalWidth > 0) {
    for (const s of [slideLeftSrc.value, slideRightSrc.value, currentImage.value]) {
      if (s && viewerSrcMatches(renderedSrc, s)) {
        rememberNaturalSize(s, img.naturalWidth, img.naturalHeight)
        break
      }
    }
  }
  flyFrameStyle.value = {
    position: 'fixed',
    left: `${rect.left}px`,
    top: `${rect.top}px`,
    width: `${rect.width}px`,
    height: `${rect.height}px`,
    transform: 'translate3d(0,0,0)',
    transformOrigin: 'top left',
    transition: 'none',
    opacity: 1,
    willChange: 'auto',
  }
}

async function activateSlideDisplayMode() {
  const src = props.images[currentIndex.value] ?? ''
  slideLeftSrc.value = src
  slideRightSrc.value = src
  slideTrackOffset.value = 0
  slideTrackTransition.value = 'none'
  slideActive.value = true
  flyDisplaySrcOverride.value = null
  await nextTick()
  await waitSlideImgPaintReady(src, 'left')
  slideMode.value = true
}

async function syncFlyFromSlideForExit() {
  const src = currentImage.value
  const slideImg = findSlideImgForSrc(src)
  slideMode.value = false
  slideActive.value = false
  flyDisplaySrcOverride.value = null
  await nextTick()
  if (slideImg) {
    applyFlyFrameFromSlideImg(slideImg)
  } else {
    restoreFlyFrameForSrc(src)
  }
  await nextFrame()
}

function finalizeSlideNav(newSrc: string, onDone?: () => void) {
  slideTrackTransition.value = 'none'
  slideLeftSrc.value = newSrc
  slideRightSrc.value = newSrc
  slideTrackOffset.value = 0
  slideActive.value = true
  slideMode.value = true
  loadingVisible.value = false
  switching = false
  flyDisplaySrcOverride.value = null
  onDone?.()
}

function slideCellSelector(cell: 'left' | 'right') {
  return cell === 'left'
    ? '.viewer-slide__cell:first-child img'
    : '.viewer-slide__cell:last-child img'
}

function getSlideCellImg(cell: 'left' | 'right'): HTMLImageElement | null {
  return slideTrackRef.value?.querySelector(slideCellSelector(cell)) as HTMLImageElement | null
}

async function waitSlideImgPaintReady(src: string, cell: 'left' | 'right', maxMs = 420) {
  await nextTick()
  const img = getSlideCellImg(cell)
  if (!img || !viewerSrcMatches(img.currentSrc || img.src, src)) return

  await waitForNaturalSize(img, maxMs)
  if (typeof img.decode === 'function') {
    try {
      await img.decode()
    } catch {
      /* 解码失败仍继续，避免卡住切换 */
    }
  }
  await nextFrame()
}

function waitSlideTrackTransitionEnd(fallbackMs: number) {
  const track = slideTrackRef.value
  if (!track) return Promise.resolve()

  return new Promise<void>((resolve) => {
    let settled = false
    const finish = () => {
      if (settled) return
      settled = true
      track.removeEventListener('transitionend', onEnd)
      resolve()
    }
    const onEnd = (event: TransitionEvent) => {
      if (event.target !== track || event.propertyName !== 'transform') return
      finish()
    }
    track.addEventListener('transitionend', onEnd)
    window.setTimeout(finish, fallbackMs)
  })
}

async function settleSlideAfterNext(newSrc: string) {
  slideTrackTransition.value = 'none'
  slideLeftSrc.value = newSrc
  await waitSlideImgPaintReady(newSrc, 'left')
  slideTrackOffset.value = 0
  slideRightSrc.value = newSrc
  await nextFrame()
}

async function finalizeSlideNavAfterTransition(newSrc: string, direction: 'left' | 'right', onDone?: () => void) {
  const vw = window.innerWidth
  const endedOnNext = direction === 'right' && Math.abs(slideTrackOffset.value + vw) < 2

  if (endedOnNext) {
    await settleSlideAfterNext(newSrc)
  } else {
    slideTrackTransition.value = 'none'
    slideLeftSrc.value = newSrc
    slideRightSrc.value = newSrc
    slideTrackOffset.value = 0
  }

  slideActive.value = true
  slideMode.value = true
  loadingVisible.value = false
  switching = false
  flyDisplaySrcOverride.value = null
  onDone?.()
}

/** 等首帧尺寸就绪，避免 FLIP 用错比例；短超时避免永远卡住 */
async function waitForNaturalSize(el: HTMLImageElement | null, maxMs = 260) {
  const t0 = performance.now()
  while (el && el.naturalWidth === 0 && performance.now() - t0 < maxMs) {
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

const slideTrackStyle = computed<CSSProperties>(() => ({
  transform: `translate3d(${slideTrackOffset.value}px,0,0)`,
  transition: slideTrackTransition.value,
}))

const nextFrame = () => new Promise<void>((r) => requestAnimationFrame(() => r()))

/** FLIP 入场：仅改变 flyFrame 的 transform */
async function runEnterFlip(gen: number) {
  if (!savedFromRect) return

  const fx = savedFromRect.left
  const fy = savedFromRect.top
  const fw = savedFromRect.width
  const fh = savedFromRect.height

  const imgEl = imgRef.value
  const src = props.images[currentIndex.value] ?? ''
  await waitForNaturalSize(imgEl)
  if (gen !== viewerGeneration) return

  frameRect = rectFromImageEl(imgEl, src)
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
  await activateSlideDisplayMode()
}

/** 关闭 FLIP：反向 transform + 淡出 */
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

/** 无有效 fromRect：直接居中，轻微缩放入场（仍只用 transform） */
async function enterFallbackStatic(gen: number) {
  const src = props.images[currentIndex.value] ?? ''
  await waitForNaturalSize(imgRef.value)
  if (gen !== viewerGeneration) return
  frameRect = rectFromImageEl(imgRef.value, src)
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
  await activateSlideDisplayMode()
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

let switching = false

function restoreFlyFrameForSrc(src: string) {
  frameRect = rectForSrc(src, imgRef.value)
  const r = frameRect
  flyFrameStyle.value = {
    position: 'fixed',
    left: `${r.left}px`,
    top: `${r.top}px`,
    width: `${r.width}px`,
    height: `${r.height}px`,
    transform: 'translate3d(0,0,0)',
    transformOrigin: 'top left',
    transition: 'none',
    opacity: 1,
    willChange: 'auto',
  }
}

async function playSlideTransition(oldSrc: string, newSrc: string, direction: 'left' | 'right') {
  if (switching || !oldSrc || !newSrc || oldSrc === newSrc) return
  const transitionGen = ++slideTransitionGeneration
  switching = true
  resetWheelAccum()

  const goingNext = direction === 'right'

  if (goingNext) {
    slideLeftSrc.value = oldSrc
    slideRightSrc.value = newSrc
  } else {
    slideLeftSrc.value = newSrc
    slideRightSrc.value = oldSrc
  }

  slideTrackTransition.value = 'none'
  const vw = window.innerWidth
  slideTrackOffset.value = goingNext ? 0 : -vw

  warmViewerImage(oldSrc)
  warmViewerImage(newSrc)

  slideLoadPending = 2
  const imgsReady =
    getWarmViewerPromise(oldSrc) !== undefined && getWarmViewerPromise(newSrc) !== undefined
  loadingVisible.value = !imgsReady

  await nextTick()
  if (!slideMode.value) slideActive.value = true
  await nextFrame()

  const imgs = slideTrackRef.value?.querySelectorAll('img')
  if (imgs) {
    imgs.forEach((img) => {
      if (img.naturalWidth > 0) {
        for (const src of [slideLeftSrc.value, slideRightSrc.value]) {
          if (src && viewerSrcMatches(img.currentSrc || img.src, src)) {
            rememberNaturalSize(src, img.naturalWidth, img.naturalHeight)
          }
        }
      }
      if (img.complete && img.naturalWidth > 0) slideLoadPending--
    })
    if (slideLoadPending <= 0) {
      slideLoadPending = 0
      loadingVisible.value = false
    }
  }

  void slideTrackRef.value?.offsetWidth

  const incomingCell = goingNext ? 'right' : 'left'
  await waitSlideImgPaintReady(newSrc, incomingCell)

  slideTrackTransition.value = `transform ${SLIDE_MS}ms ${EASING}`
  await nextFrame()
  slideTrackOffset.value = goingNext ? -vw : 0

  void waitSlideTrackTransitionEnd(SLIDE_MS + 80).then(async () => {
    if (transitionGen !== slideTransitionGeneration) return
    await finalizeSlideNavAfterTransition(newSrc, direction)
  })
}

async function switchTo(newIndex: number, direction: 'left' | 'right') {
  if (currentIndex.value === newIndex) return
  const oldIndex = currentIndex.value
  const oldSrc = props.images[oldIndex] ?? ''
  const newSrc = props.images[newIndex] ?? ''
  warmAdjacentViewerImages(props.images, newIndex)
  currentIndex.value = newIndex
  await playSlideTransition(oldSrc, newSrc, direction)
}

function emitEdgeNavigate(direction: 'prev' | 'next') {
  emit('edge-navigate', direction, props.itemIds?.[currentIndex.value])
}

function prevImage() {
  if (currentIndex.value <= 0) {
    emitEdgeNavigate('prev')
    return
  }
  switchTo(currentIndex.value - 1, 'left')
}

function nextImage() {
  if (currentIndex.value >= props.images.length - 1) {
    emitEdgeNavigate('next')
    return
  }
  switchTo(currentIndex.value + 1, 'right')
}

function close() {
  emit('update:visible', false)
}

function onImageLoad() {
  const el = imgRef.value
  const src = flyDisplaySrc.value
  if (el && src && el.naturalWidth > 0) {
    rememberNaturalSize(src, el.naturalWidth, el.naturalHeight)
    if (!slideActive.value && !switching) {
      restoreFlyFrameForSrc(src)
    }
  }
  if (!slideActive.value) loadingVisible.value = false
}

function onImageError() {
  if (!slideActive.value) loadingVisible.value = false
}

let slideLoadPending = 0
function onSlideImgLoad(event: Event) {
  const img = event.target as HTMLImageElement
  if (img.naturalWidth > 0) {
    for (const src of [slideLeftSrc.value, slideRightSrc.value]) {
      if (src && viewerSrcMatches(img.currentSrc || img.src, src)) {
        rememberNaturalSize(src, img.naturalWidth, img.naturalHeight)
      }
    }
  }
  slideLoadPending = Math.max(0, slideLoadPending - 1)
  if (slideLoadPending <= 0) loadingVisible.value = false
}
function onSlideImgError() {
  slideLoadPending = Math.max(0, slideLoadPending - 1)
  if (slideLoadPending <= 0) loadingVisible.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (!internalVisible.value || switching) return
  if (e.key === 'ArrowLeft') prevImage()
  else if (e.key === 'ArrowRight') nextImage()
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
    if (!slideActive.value || slideMode.value) return
    const src = props.images[currentIndex.value] ?? ''
    frameRect = rectForSrc(src, imgRef.value)
    const r = frameRect
    flyFrameStyle.value = {
      ...flyFrameStyle.value,
      left: `${r.left}px`,
      top: `${r.top}px`,
      width: `${r.width}px`,
      height: `${r.height}px`,
      transition: 'none',
    }
  }, 120)
}

onMounted(() => window.addEventListener('resize', handleResize))
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('keydown', onKeydown)
  cleanupTouchDragListeners()
  if (resizeTimer) clearTimeout(resizeTimer)
  slideTransitionGeneration++
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

      const idx = Math.max(0, Math.min(props.initialIndex ?? 0, props.images.length - 1))
      currentIndex.value = idx
      warmAdjacentViewerImages(props.images, idx)

      loadingVisible.value = true
      chromeVisible.value = false
      maskVisible.value = false
      slideActive.value = false
      slideMode.value = false
      switching = false
      resetFlyFrameForOpen()

      pendingTouchEdgeNav = null
      touchDragging = false
      touchDragMode = null
      flyDisplaySrcOverride.value = null

      internalVisible.value = true
      lockScroll()

      await nextTick()
      maskVisible.value = true
      chromeVisible.value = true
      warmTouchPeekImages()

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
      slideTransitionGeneration++

      chromeVisible.value = false
      maskVisible.value = false
      loadingVisible.value = false
      switching = true
      const wasSlideMode = slideMode.value
      pendingTouchEdgeNav = null
      touchDragging = false
      touchDragMode = null
      flyDisplaySrcOverride.value = null

      if (props.skipExitAnimation) {
        slideActive.value = false
        slideMode.value = false
        internalVisible.value = false
        switching = false
        stopLivePlayback()
        unlockScroll()
        emit('update:visible', false)
        emit('close')
        return
      }

      await nextTick()
      await nextFrame()

      if (wasSlideMode) {
        await syncFlyFromSlideForExit()
      } else {
        slideActive.value = false
        slideMode.value = false
        const curSrc = props.images[currentIndex.value] ?? ''
        frameRect = rectFromImageEl(imgRef.value, curSrc)
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
      }

      const curSrc = props.images[currentIndex.value] ?? ''
      if (!frameRect) {
        frameRect = rectFromImageEl(imgRef.value, curSrc)
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
      stopLivePlayback()
      unlockScroll()
      emit('update:visible', false)
      emit('close')
    }
  },
  { immediate: true },
)

const displayCurrent = computed(() => props.navCurrent ?? currentIndex.value + 1)
const displayTotal = computed(() => props.navTotal ?? props.images.length)
const navEligible = computed(() => displayTotal.value > 1)
const showPrevBtn = computed(() => navEligible.value && displayCurrent.value > 1)
const showNextBtn = computed(() => navEligible.value && displayCurrent.value < displayTotal.value)
const currentImage = computed(() => props.images[currentIndex.value] || '')
const flyDisplaySrc = computed(() => flyDisplaySrcOverride.value ?? currentImage.value)
const currentLiveVideoUrl = computed(() => {
  const urls = props.liveVideoUrls ?? []
  return urls[currentIndex.value]?.trim() ?? ''
})

function cleanupTouchDragListeners() {
  document.removeEventListener('touchmove', onDocumentTouchMove)
  document.removeEventListener('touchend', onDocumentTouchEnd)
  document.removeEventListener('touchcancel', onDocumentTouchEnd)
  if (touchMoveRaf) {
    cancelAnimationFrame(touchMoveRaf)
    touchMoveRaf = 0
  }
}

function resolvePeekSrc(mode: 'next' | 'prev'): string {
  if (mode === 'next') {
    const inArray = props.images[currentIndex.value + 1]?.trim()
    if (inArray) return inArray
    return props.adjacentNextSrc?.trim() ?? ''
  }
  const inArray = props.images[currentIndex.value - 1]?.trim()
  if (inArray) return inArray
  return props.adjacentPrevSrc?.trim() ?? ''
}

function warmTouchPeekImages() {
  const cur = props.images[currentIndex.value]?.trim()
  if (cur) warmViewerImage(cur)
  const prev = resolvePeekSrc('prev')
  const next = resolvePeekSrc('next')
  if (prev) warmViewerImage(prev)
  if (next) warmViewerImage(next)
}

function getTrackedTouch(event: TouchEvent) {
  const list = event.touches.length > 0 ? event.touches : event.changedTouches
  if (activeTouchId == null) return list[0] ?? null
  for (let i = 0; i < list.length; i++) {
    const touch = list.item(i)
    if (touch && touch.identifier === activeTouchId) return touch
  }
  return null
}

function touchHasAdjacentImage(mode: 'next' | 'prev') {
  return Boolean(resolvePeekSrc(mode))
}

function hasLocalAdjacentImage(mode: 'next' | 'prev') {
  if (mode === 'next') return currentIndex.value < props.images.length - 1
  return currentIndex.value > 0
}

function finishTouchDragEdgeNavigate(mode: 'next' | 'prev') {
  touchDragging = false
  touchDragMode = null
  switching = true
  pendingTouchEdgeNav = mode
  slideTrackTransition.value = 'none'
  if (mode === 'next') nextImage()
  else prevImage()
}

function handoffTouchEdgeNav(newSrc: string) {
  pendingTouchEdgeNav = null
  finalizeTouchDragSwitch(newSrc)
}

function touchCanCommit(mode: 'next' | 'prev') {
  if (mode === 'next') return showNextBtn.value
  return showPrevBtn.value
}

function beginTouchDrag(mode: 'next' | 'prev') {
  const vw = window.innerWidth
  const currentSrc = props.images[currentIndex.value] ?? ''
  const peek = resolvePeekSrc(mode)
  touchDragMode = mode
  stopLivePlayback()
  flyDisplaySrcOverride.value = null

  if (mode === 'next') {
    slideLeftSrc.value = currentSrc
    slideRightSrc.value = peek
    slideTrackOffset.value = 0
  } else {
    slideLeftSrc.value = peek
    slideRightSrc.value = currentSrc
    slideTrackOffset.value = -vw
  }

  if (peek) warmViewerImage(peek)

  slideTrackTransition.value = 'none'
  slideActive.value = true
  loadingVisible.value = false
  void slideTrackRef.value?.offsetWidth
}

function applyTouchDragOffset(dx: number) {
  if (!touchDragMode) return
  const vw = window.innerWidth
  const hasAdjacent = touchHasAdjacentImage(touchDragMode)
  const canCommit = touchCanCommit(touchDragMode)

  if (touchDragMode === 'next') {
    let offset = dx
    if (hasAdjacent || canCommit) {
      offset = Math.max(-vw, Math.min(0, dx))
    } else {
      offset = Math.max(-vw * 0.28, Math.min(0, dx * TOUCH_RUBBER_BAND))
    }
    slideTrackOffset.value = offset
    return
  }

  let offset = -vw + dx
  if (hasAdjacent || canCommit) {
    offset = Math.max(-vw, Math.min(0, offset))
  } else {
    offset = Math.min(0, Math.max(-vw, -vw + dx * TOUCH_RUBBER_BAND))
  }
  slideTrackOffset.value = offset
}

function snapBackTouchDrag(targetOffset: number) {
  switching = true
  const src = props.images[currentIndex.value] ?? ''
  slideTrackTransition.value = `transform ${SLIDE_MS}ms ${EASING}`
  slideTrackOffset.value = targetOffset
  void waitSlideTrackTransitionEnd(SLIDE_MS + 80).then(() => {
    finalizeSlideNav(src, () => {
      touchDragging = false
      touchDragMode = null
    })
  })
}

function finalizeTouchDragSwitch(newSrc: string) {
  touchDragging = false
  touchDragMode = null
  const vw = window.innerWidth
  if (Math.abs(slideTrackOffset.value + vw) < 2) {
    void settleSlideAfterNext(newSrc).then(() => {
      slideActive.value = true
      slideMode.value = true
      loadingVisible.value = false
      switching = false
      flyDisplaySrcOverride.value = null
    })
    return
  }
  finalizeSlideNav(newSrc)
}

function completeTouchDrag(dx: number) {
  if (!touchDragging || !touchDragMode) return

  const vw = window.innerWidth
  const threshold = Math.min(vw * TOUCH_COMMIT_RATIO, TOUCH_COMMIT_MIN_PX)
  const mode = touchDragMode

  if (mode === 'next') {
    if (dx < -threshold && touchCanCommit('next')) {
      switching = true
      slideTrackTransition.value = `transform ${SLIDE_MS}ms ${EASING}`
      slideTrackOffset.value = -vw
      void waitSlideTrackTransitionEnd(SLIDE_MS + 80).then(() => {
        if (hasLocalAdjacentImage('next')) {
          const newIndex = currentIndex.value + 1
          warmAdjacentViewerImages(props.images, newIndex)
          currentIndex.value = newIndex
          finalizeTouchDragSwitch(props.images[newIndex] ?? '')
        } else {
          finishTouchDragEdgeNavigate('next')
        }
      })
      return
    }
    snapBackTouchDrag(0)
    return
  }

  if (dx > threshold && touchCanCommit('prev')) {
    switching = true
    slideTrackTransition.value = `transform ${SLIDE_MS}ms ${EASING}`
    slideTrackOffset.value = 0
    void waitSlideTrackTransitionEnd(SLIDE_MS + 80).then(() => {
      if (hasLocalAdjacentImage('prev')) {
        const newIndex = currentIndex.value - 1
        warmAdjacentViewerImages(props.images, newIndex)
        currentIndex.value = newIndex
        finalizeTouchDragSwitch(props.images[newIndex] ?? '')
      } else {
        finishTouchDragEdgeNavigate('prev')
      }
    })
    return
  }
  snapBackTouchDrag(-vw)
}

function onViewerTouchStart(event: TouchEvent) {
  if (!internalVisible.value || switching) return
  if (slideActive.value && !slideMode.value && !touchDragging) return
  if (!navEligible.value) return

  const touch = event.changedTouches[0] ?? event.touches[0]
  if (!touch) return

  cleanupTouchDragListeners()
  activeTouchId = touch.identifier
  touchStartX = touch.clientX
  touchStartY = touch.clientY
  touchAxisLocked = null
  touchDragging = false
  touchDragMode = null

  document.addEventListener('touchmove', onDocumentTouchMove, { passive: false })
  document.addEventListener('touchend', onDocumentTouchEnd)
  document.addEventListener('touchcancel', onDocumentTouchEnd)
}

function scheduleTouchDragOffset(dx: number) {
  pendingTouchDx = dx
  if (touchMoveRaf) return
  touchMoveRaf = requestAnimationFrame(() => {
    touchMoveRaf = 0
    applyTouchDragOffset(pendingTouchDx)
  })
}

function onDocumentTouchMove(event: TouchEvent) {
  if (activeTouchId == null || !internalVisible.value || switching) return
  const touch = getTrackedTouch(event)
  if (!touch) return

  const dx = touch.clientX - touchStartX
  const dy = touch.clientY - touchStartY

  if (!touchAxisLocked) {
    if (Math.abs(dx) < TOUCH_AXIS_THRESHOLD && Math.abs(dy) < TOUCH_AXIS_THRESHOLD) return
    if (Math.abs(dx) > Math.abs(dy) * 1.15) touchAxisLocked = 'x'
    else if (Math.abs(dy) > Math.abs(dx) * 1.15) touchAxisLocked = 'y'
    else return
  }

  if (touchAxisLocked !== 'x') return
  event.preventDefault()

  if (!touchDragging) {
    if (dx < 0) beginTouchDrag('next')
    else if (dx > 0) beginTouchDrag('prev')
    else return
    touchDragging = true
  }

  scheduleTouchDragOffset(dx)
}

function onDocumentTouchEnd(event: TouchEvent) {
  if (touchMoveRaf) {
    cancelAnimationFrame(touchMoveRaf)
    touchMoveRaf = 0
    applyTouchDragOffset(pendingTouchDx)
  }

  const touch = getTrackedTouch(event)
  cleanupTouchDragListeners()

  if (!touch || !internalVisible.value) {
    activeTouchId = null
    touchAxisLocked = null
    return
  }

  const dx = touch.clientX - touchStartX
  const dy = touch.clientY - touchStartY
  activeTouchId = null
  touchAxisLocked = null

  if (touchDragging) {
    completeTouchDrag(dx)
    return
  }

  if (Math.abs(dx) < TOUCH_AXIS_THRESHOLD || Math.abs(dx) < Math.abs(dy)) return
  if (dx < 0) nextImage()
  else prevImage()
}

function stopLivePlayback() {
  livePlaying.value = false
  const video = liveVideoRef.value
  if (video) {
    video.pause()
    video.currentTime = 0
  }
  if (internalVisible.value && props.visible) {
    void activateSlideDisplayMode()
  }
}

async function playLivePlayback() {
  const url = currentLiveVideoUrl.value
  if (!url) return
  if (slideMode.value) {
    const src = currentImage.value
    const slideImg = findSlideImgForSrc(src)
    slideMode.value = false
    slideActive.value = false
    flyDisplaySrcOverride.value = null
    await nextTick()
    if (slideImg) applyFlyFrameFromSlideImg(slideImg)
    else restoreFlyFrameForSrc(src)
  }
  livePlaying.value = true
  await nextTick()
  const video = liveVideoRef.value
  if (!video) return
  video.currentTime = 0
  try {
    await video.play()
  } catch {
    livePlaying.value = false
  }
}

function onViewerStageClick(event: MouseEvent) {
  if (event.button !== 0) return
  if (touchDragging) return
  if (!currentLiveVideoUrl.value) return
  if (livePlaying.value) return
  void playLivePlayback()
}

watch(currentIndex, () => {
  stopLivePlayback()
  warmTouchPeekImages()
})

watch(
  () => [props.adjacentPrevSrc, props.adjacentNextSrc, internalVisible.value] as const,
  ([, , open]) => {
    if (open) warmTouchPeekImages()
  },
)

watch(
  () => props.images[0] ?? '',
  (newSrc, oldSrc) => {
    if (!internalVisible.value || !props.visible) return
    if (!newSrc || !oldSrc || newSrc === oldSrc) return
    if (pendingTouchEdgeNav) {
      handoffTouchEdgeNav(newSrc)
      return
    }
    void playSlideTransition(oldSrc, newSrc, props.slideDirection ?? 'right')
  },
)
</script>

<style scoped lang="scss">
@use './shared/viewer-shell.scss';

.viewer-fly {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: auto;
  cursor: default;
}

.viewer-fly__frame {
  position: relative;
  overflow: hidden;
  border-radius: 2px;
}

.viewer-fly__img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
  -webkit-user-drag: none;

  &.is-live-hidden {
    opacity: 0;
  }
}

.viewer-fly__video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: transparent;
  pointer-events: none;
}

.viewer-slide {
  position: fixed;
  inset: 0;
  z-index: 1;
  overflow: hidden;
  pointer-events: none;
}

.viewer-slide__track {
  display: flex;
  flex-direction: row;
  width: 200vw;
  height: 100%;
  will-change: transform;
  backface-visibility: hidden;
}

.viewer-slide__cell {
  flex: 0 0 100vw;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.viewer-slide__img {
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
  -webkit-user-drag: none;
}

</style>
