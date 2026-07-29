<template>
  <section
    ref="stageRef"
    class="card-switching"
    :class="{ 'is-dragging': gestureAxis === 'horizontal' }"
    aria-label="卡牌切换"
    tabindex="0"
    @keydown.left.prevent="previous"
    @keydown.right.prevent="next"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerCancel"
  >
    <button
      class="card-switching__arrow viewer-nav viewer-prev"
      type="button"
      :disabled="activeIndex === 0 || buttonAnimating"
      :class="{ 'is-hidden': activeIndex === 0 || buttonAnimating }"
      aria-label="上一张"
      @pointerdown.stop
      @pointerup.stop
      @click="previous"
    ><img src="@/assets/icon/left.svg" alt="" aria-hidden="true" /></button>
    <div ref="deckRef" class="card-switching__deck">
      <div
        v-for="card in visibleItems"
        :key="card.item.id"
        class="card-switching__card"
        :class="{
          'is-active': card.index === activeIndex,
          'is-prev': card.index < activeIndex,
          'is-next': card.index > activeIndex,
        }"
        :style="stackStyle(card.index)"
      >
        <GalleryCardItem
          :item="card.item"
          :draggable="false"
          :show-admin-actions="showAdminActions"
          :gallery-folder-key="galleryFolderKey"
          :item-style="card.index === activeIndex ? getItemStyle?.(card.item, activeIndex) : undefined"
          @click="card.index === activeIndex && handleCardClick($event, card.index)"
          @delete="$emit('delete', $event)"
          @transfer="$emit('transfer', $event)"
          @edit="$emit('edit', $event)"
          @view="$emit('view', $event)"
          @download="$emit('download', $event)"
          @copy-link="$emit('copy-link', $event)"
          @aspect-hw="(id, hw) => $emit('aspect-hw', id, hw)"
        />
      </div>
    </div>
    <button
      class="card-switching__arrow viewer-nav viewer-next"
      type="button"
      :disabled="activeIndex === items.length - 1 || buttonAnimating"
      :class="{ 'is-hidden': activeIndex === items.length - 1 || buttonAnimating }"
      aria-label="下一张"
      @pointerdown.stop
      @pointerup.stop
      @click="next"
    ><img src="@/assets/icon/left.svg" alt="" aria-hidden="true" /></button>
    <p class="card-switching__counter">{{ activeIndex + 1 }} / {{ items.length }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import GalleryCardItem from '@/components/gallery/GalleryCardItem.vue'
import type { GalleryDisplayItem, GalleryItemStyleFn } from '@/components/gallery/types'

const props = defineProps<{
  items: GalleryDisplayItem[]
  showAdminActions?: boolean
  galleryFolderKey?: string
  getItemStyle?: GalleryItemStyleFn
}>()

const emit = defineEmits<{
  (e: 'card-click', event: MouseEvent, index: number): void
  (e: 'delete', item: GalleryDisplayItem): void
  (e: 'transfer', item: GalleryDisplayItem): void
  (e: 'edit', item: GalleryDisplayItem): void
  (e: 'view', item: GalleryDisplayItem): void
  (e: 'download', item: GalleryDisplayItem): void
  (e: 'copy-link', item: GalleryDisplayItem): void
  (e: 'aspect-hw', itemId: string, hw: number): void
}>()

const activeIndex = ref(0)
const pointerStartX = ref<number | null>(null)
const pointerStartY = ref<number | null>(null)
const activePointerId = ref<number | null>(null)
const gestureAxis = ref<'pending' | 'horizontal' | 'vertical' | null>(null)
const stageRef = ref<HTMLElement | null>(null)
const deckRef = ref<HTMLElement | null>(null)
const dragOffset = ref(0)
const pointerCaptured = ref(false)
const buttonAnimating = ref(false)
const SWIPE_SENSITIVITY = 1.5
const POINTER_CAPTURE_THRESHOLD_PX = 6
const TOUCH_DIRECTION_LOCK_PX = 8
const DIRECTION_LOCK_RATIO = 1.15
const SWIPE_THRESHOLD_RATIO = 3 / 5
let suppressCardClickUntil = 0
const visibleItems = computed(() =>
  props.items
    .map((item, index) => ({ item, index }))
    .filter(({ index }) => Math.abs(index - activeIndex.value) <= 1),
)

watch(
  () => props.items.map((item) => item.id).join('|'),
  () => {
    activeIndex.value = 0
    dragOffset.value = 0
  },
)

function next() {
  animateButtonSwitch(1)
}
function previous() {
  animateButtonSwitch(-1)
}
function animateButtonSwitch(direction: 1 | -1) {
  if (buttonAnimating.value) return
  if (direction === 1 && activeIndex.value >= props.items.length - 1) return
  if (direction === -1 && activeIndex.value <= 0) return
  buttonAnimating.value = true
  const threshold = getSwipeThreshold()
  dragOffset.value = direction === 1 ? -threshold : threshold
  window.setTimeout(() => {
    activeIndex.value += direction
    dragOffset.value = 0
    buttonAnimating.value = false
  }, 280)
}
function handleCardClick(event: MouseEvent, index: number) {
  if (performance.now() < suppressCardClickUntil) return
  emit('card-click', event, index)
}
function onPointerDown(event: PointerEvent) {
  if (!event.isPrimary || (event.pointerType === 'mouse' && event.button !== 0)) return
  const target = event.target as Element | null
  if (target?.closest('.card-toolbar, a, button, [role="button"]')) return
  pointerStartX.value = event.clientX
  pointerStartY.value = event.clientY
  activePointerId.value = event.pointerId
  gestureAxis.value = 'pending'
  pointerCaptured.value = false
}
function onPointerMove(event: PointerEvent) {
  if (
    event.pointerId !== activePointerId.value ||
    pointerStartX.value === null ||
    pointerStartY.value === null ||
    gestureAxis.value === 'vertical'
  ) {
    return
  }
  const pointerDelta = event.clientX - pointerStartX.value
  const verticalDelta = event.clientY - pointerStartY.value
  if (gestureAxis.value === 'pending') {
    const absX = Math.abs(pointerDelta)
    const absY = Math.abs(verticalDelta)
    if (Math.max(absX, absY) < TOUCH_DIRECTION_LOCK_PX) return
    if (absY > absX * DIRECTION_LOCK_RATIO) {
      gestureAxis.value = 'vertical'
      dragOffset.value = 0
      return
    }
    if (absX <= absY * DIRECTION_LOCK_RATIO) return
    gestureAxis.value = 'horizontal'
    suppressCardClickUntil = performance.now() + 350
  }

  event.preventDefault()
  const threshold = getSwipeThreshold()
  const direction = pointerDelta < 0 ? 1 : -1
  const canSwitch =
    (direction === 1 && activeIndex.value < props.items.length - 1) ||
    (direction === -1 && activeIndex.value > 0)
  if (!pointerCaptured.value && Math.abs(pointerDelta) >= POINTER_CAPTURE_THRESHOLD_PX) {
    stageRef.value?.setPointerCapture?.(event.pointerId)
    pointerCaptured.value = true
  }
  const nextOffset = pointerDelta * SWIPE_SENSITIVITY
  dragOffset.value = canSwitch
    ? nextOffset
    : Math.sign(nextOffset) * Math.min(Math.abs(nextOffset), threshold)
}
function onPointerUp(event: PointerEvent) {
  if (event.pointerId !== activePointerId.value || pointerStartX.value === null) return
  if (gestureAxis.value !== 'horizontal') {
    clearPointerGesture()
    return
  }
  const delta = (event.clientX - pointerStartX.value) * SWIPE_SENSITIVITY
  suppressCardClickUntil = performance.now() + 350
  clearPointerGesture()
  const threshold = getSwipeThreshold()
  const direction = delta < 0 ? 1 : -1
  const canSwitch =
    (direction === 1 && activeIndex.value < props.items.length - 1) ||
    (direction === -1 && activeIndex.value > 0)
  const shouldSwitch = Math.abs(delta) >= threshold && canSwitch
  if (shouldSwitch) {
    activeIndex.value += direction
    dragOffset.value = 0
  } else {
    resetDragWithAnimation()
  }
}
function onPointerCancel(event: PointerEvent) {
  if (event.pointerId !== activePointerId.value) return
  const shouldAnimateBack = gestureAxis.value === 'horizontal' && dragOffset.value !== 0
  clearPointerGesture()
  if (shouldAnimateBack) resetDragWithAnimation()
  else dragOffset.value = 0
}
function clearPointerGesture() {
  pointerStartX.value = null
  pointerStartY.value = null
  activePointerId.value = null
  gestureAxis.value = null
  pointerCaptured.value = false
}
function getSwipeThreshold() {
  return (deckRef.value?.getBoundingClientRect().width ?? 360) * SWIPE_THRESHOLD_RATIO
}
function resetDragWithAnimation() {
  const currentOffset = dragOffset.value
  dragOffset.value = currentOffset
  requestAnimationFrame(() => {
    dragOffset.value = 0
  })
}
function stackStyle(index: number) {
  const relative = index - activeIndex.value
  const drag = dragOffset.value
  const deckWidth = deckRef.value?.getBoundingClientRect().width ?? 360
  const threshold = deckWidth * SWIPE_THRESHOLD_RATIO
  const dragProgress = Math.min(1, Math.abs(drag) / threshold)
  if (relative !== 0) {
    const isNextEntering = relative > 0 && drag < 0
    const isPrevEntering = relative < 0 && drag > 0
    const isPrevLeaving = relative < 0 && drag < 0
    const isNextLeaving = relative > 0 && drag > 0
    if (isPrevLeaving || isNextLeaving) {
      const direction = isPrevLeaving ? -1 : 1
      return {
        transform: `translate(${direction * (30 + 20 * dragProgress)}px, 0px) rotate(${direction * (2 + 2 * dragProgress)}deg) scale(${0.9 - 0.1 * dragProgress})`,
        transformOrigin: isPrevLeaving ? '0% 60%' : '100% 60%',
        opacity: 1 - dragProgress,
        zIndex: 3,
      }
    }
    if (isNextEntering || isPrevEntering) {
      const direction = isNextEntering ? 1 : -1
      // 后置卡牌比前景卡牌更快完成入场，让左右滑动时露出更及时。
      const enteringProgress = Math.min(1, dragProgress * 1.35)
      return {
        transform: `translate(${direction * 30 * (1 - enteringProgress)}px, 0px) rotate(${direction * 2 * (1 - enteringProgress)}deg) scale(${0.9 + 0.1 * enteringProgress})`,
        transformOrigin: isNextEntering ? '100% 60%' : '0% 60%',
        zIndex: enteringProgress >= 1 ? 6 : 5 - Math.abs(relative),
      }
    }
    return {
      zIndex: 5 - Math.abs(relative),
    }
  }
  const restingAngle = relative === 0 ? 0 : relative < 0 ? -2 : 2
  const dragDistance = Math.abs(drag)
  const dragDirection = drag < 0 ? -1 : 1
  // 以目标卡片提升到最顶层的阈值作为回到背后层的起点，
  // 不把 150px 的跟手平移距离当作切换判定。
  const overshootProgress = Math.min(1, Math.max(0, (dragDistance - threshold) / threshold))
  const cappedProgress = Math.min(1, dragDistance / threshold)
  // 0 → threshold 之间连续地走完 0 → 150px，避免在某个距离点跳变。
  const normalShift = dragDirection * 150 * cappedProgress
  const backShift = dragDirection * (150 - 120 * overshootProgress)
  const dragShift = overshootProgress > 0 ? backShift : normalShift
  const normalAngle = Math.max(-16, Math.min(16, drag * 0.035))
  const backAngle = dragDirection * 2
  const thresholdAngle = dragDirection * Math.min(16, threshold * 0.035)
  const dragAngle =
    overshootProgress <= 0
      ? normalAngle
      : thresholdAngle + (backAngle - thresholdAngle) * overshootProgress
  const stackShift = relative * 16
  const normalScale = 1 - cappedProgress * 0.1
  const activeScale =
    overshootProgress <= 0
      ? normalScale
      : 0.9 + (normalScale - 0.9) * (1 - overshootProgress)
  return {
    transform: `translate(${stackShift + dragShift}px, 0) rotate(${restingAngle + dragAngle}deg) scale(${relative === 0 ? activeScale : 0.9})`,
    transformOrigin:
      relative !== 0
        ? 'center bottom'
        : drag < 0
          ? 'left bottom'
          : drag > 0
            ? 'right bottom'
            : 'center bottom',
    zIndex: 5 - Math.abs(relative),
  }
}
</script>

<style scoped lang="scss">
.card-switching {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  justify-content: center;
  align-items: center;
  gap: clamp(4px, 2vw, 24px);
  padding: 18px 12px 6px;
  outline: none;
  user-select: none;
  touch-action: pan-y;
}

.card-switching__deck {
  position: relative;
  width: min(100%, 420px);
  aspect-ratio: 3 / 4;
  grid-column: 1;
  justify-self: center;
}

.card-switching__card {
  position: absolute;
  inset: 0;
  // overflow: hidden;
  border-radius: 14px;
  transition: transform 0.28s ease, opacity 0.28s ease;
  transform-origin: center bottom;
  // 卡牌区域内只处理横向切换，不让触摸手势带动页面上下滚动。
  // 仅限制该区域，不修改 body overflow，避免滚动条消失造成布局变宽。
  touch-action: none;
  pointer-events: none;
}

.card-switching.is-dragging .card-switching__card {
  transition: none;
}

.card-switching__card:has(.gallery-card-item.is-new) {
  transition: none;
}

.card-switching__card.is-active {
  pointer-events: auto;
}

.card-switching__card.is-next {
  transform: translate(30px, 0px) rotate(2deg) scale(0.9);
  transform-origin: 100% 60%;
}

.card-switching__card.is-prev {
  transform: translate(-30px, 0px) rotate(-2deg) scale(0.9);
  transform-origin: 0% 60%;
}

.card-switching__card :deep(.gallery-card-item) {
  height: 100%;
}

.card-switching__card :deep(.card-wrapper),
.card-switching__card :deep(.card),
.card-switching__card :deep(.image-container),
.card-switching__card :deep(.image-slot) {
  height: 100% !important;
}

.card-switching__card :deep(.image),
.card-switching__card :deep(.card-video),
.card-switching__card :deep(.video-poster) {
  position: absolute !important;
  inset: 0 !important;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
}

.card-switching__card :deep(.card .image-slot.has-media .image) {
  position: absolute !important;
  inset: 0 !important;
  width: 100% !important;
  height: 100% !important;
  object-fit: cover !important;
  object-position: center;
}

.card-switching__arrow {
  z-index: 5;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.72);
  color: #fff;
  font-size: 30px;
  line-height: 30px;
  cursor: pointer;
}

.card-switching__arrow.viewer-nav {
  position: absolute;
  top: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transform: translateY(-50%);
  transition:
    background 0.2s ease,
    opacity 0.22s ease;
}

.card-switching__arrow.viewer-prev { left: 24px; }
.card-switching__arrow.viewer-next { right: 24px; }
.card-switching__arrow.viewer-nav.is-hidden { opacity: 0; pointer-events: none; }

.card-switching__arrow.viewer-nav img {
  display: block;
  width: 16px;
  height: 16px;
  object-fit: contain;
  filter: brightness(0) invert(1);
  opacity: 0.92;
}

.card-switching__arrow.viewer-next img { transform: scaleX(-1); }

.card-switching__arrow:disabled { opacity: 0.22; cursor: default; }
.card-switching__counter { grid-column: 1; margin: 8px 0 0; text-align: center; color: #555; font-size: 13px; }

@media (max-width: 640px) {
  .card-switching {
    grid-template-columns: minmax(0, 1fr);
    gap: 2px;
    width: 100%;
    box-sizing: border-box;
    padding: 40px 40px 6px;
  }
  .card-switching__arrow.viewer-nav { display: none; }
  .card-switching__deck {
    grid-column: 1;
    width: 100%;
    max-width: 360px;
    margin-inline: auto;
  }
  .card-switching__counter {
    grid-column: 1;
    justify-self: center;
  }
}
</style>
