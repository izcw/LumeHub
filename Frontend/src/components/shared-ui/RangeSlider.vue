<template>
  <div
    ref="rootRef"
    class="ui-range-slider"
    :class="{ 'is-disabled': disabled, 'is-dragging': dragging }"
    :style="rootStyle"
    role="slider"
    :aria-valuemin="min"
    :aria-valuemax="max"
    :aria-valuenow="modelValue"
    :aria-disabled="disabled || undefined"
    tabindex="0"
    @pointerdown="onPointerDown"
    @keydown="onKeyDown"
  >
    <div class="ui-range-slider__track" aria-hidden="true">
      <div class="ui-range-slider__fill" :style="{ width: fillPercent }" />
    </div>
    <div class="ui-range-slider__thumb" aria-hidden="true" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const THUMB_SIZE = 12
const TRACK_HEIGHT = 4

const props = withDefaults(
  defineProps<{
    modelValue: number
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    accentColor?: string
    trackColor?: string
  }>(),
  {
    min: 0,
    max: 100,
    step: 1,
    disabled: false,
    accentColor: undefined,
    trackColor: undefined,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
  pointerdown: [event: PointerEvent]
}>()

const rootRef = ref<HTMLDivElement | null>(null)
const dragging = ref(false)

const fillRatio = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 0
  return Math.max(0, Math.min(1, (props.modelValue - props.min) / span))
})

const fillPercent = computed(() => `${fillRatio.value * 100}%`)

const rootStyle = computed(() => ({
  '--ui-range-fill': props.accentColor ?? '#000',
  '--ui-range-track': props.trackColor ?? '#d8d8d8',
  '--ui-range-thumb-size': `${THUMB_SIZE}px`,
  '--ui-range-track-height': `${TRACK_HEIGHT}px`,
  '--ui-range-ratio': String(fillRatio.value),
}))

function snapValue(raw: number): number {
  const clamped = Math.max(props.min, Math.min(props.max, raw))
  if (props.step <= 0) return clamped
  const steps = Math.round((clamped - props.min) / props.step)
  return props.min + steps * props.step
}

function valueFromClientX(clientX: number): number {
  const el = rootRef.value
  if (!el) return props.modelValue
  const rect = el.getBoundingClientRect()
  const inset = THUMB_SIZE / 2
  const trackWidth = Math.max(1, rect.width - THUMB_SIZE)
  const x = clientX - rect.left - inset
  const ratio = Math.max(0, Math.min(1, x / trackWidth))
  return snapValue(props.min + ratio * (props.max - props.min))
}

function emitValue(next: number) {
  if (next === props.modelValue) return
  emit('update:modelValue', next)
}

function onPointerMove(event: PointerEvent) {
  emitValue(valueFromClientX(event.clientX))
}

function endDrag(event: PointerEvent) {
  const el = rootRef.value
  if (!el) return
  dragging.value = false
  if (el.hasPointerCapture(event.pointerId)) {
    el.releasePointerCapture(event.pointerId)
  }
  el.removeEventListener('pointermove', onPointerMove)
  el.removeEventListener('pointerup', endDrag)
  el.removeEventListener('pointercancel', endDrag)
}

function onPointerDown(event: PointerEvent) {
  if (props.disabled || event.button !== 0) return
  event.preventDefault()
  emit('pointerdown', event)
  const el = rootRef.value
  if (!el) return
  dragging.value = true
  el.setPointerCapture(event.pointerId)
  emitValue(valueFromClientX(event.clientX))
  el.addEventListener('pointermove', onPointerMove)
  el.addEventListener('pointerup', endDrag)
  el.addEventListener('pointercancel', endDrag)
}

function onKeyDown(event: KeyboardEvent) {
  if (props.disabled) return
  const step = props.step > 0 ? props.step : 1
  let next: number | null = null
  if (event.key === 'ArrowRight' || event.key === 'ArrowUp') next = props.modelValue + step
  else if (event.key === 'ArrowLeft' || event.key === 'ArrowDown') next = props.modelValue - step
  else if (event.key === 'Home') next = props.min
  else if (event.key === 'End') next = props.max
  if (next == null) return
  event.preventDefault()
  emitValue(snapValue(next))
}
</script>

<style scoped lang="scss">
.ui-range-slider {
  --thumb: var(--ui-range-thumb-size, 12px);
  --track-h: var(--ui-range-track-height, 4px);

  position: relative;
  width: 100%;
  height: var(--thumb);
  touch-action: none;
  user-select: none;
  cursor: pointer;
  outline: none;

  &:focus-visible .ui-range-slider__thumb {
    outline: 2px solid rgba(0, 0, 0, 0.35);
    outline-offset: 2px;
  }

  &.is-dragging,
  &.is-dragging .ui-range-slider__thumb {
    cursor: grabbing;
  }

  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.ui-range-slider__track {
  position: absolute;
  left: calc(var(--thumb) / 2);
  right: calc(var(--thumb) / 2);
  top: 50%;
  height: var(--track-h);
  margin-top: calc(var(--track-h) / -2);
  border-radius: 2px;
  background: var(--ui-range-track, #d8d8d8);
  overflow: hidden;
  pointer-events: none;
}

.ui-range-slider__fill {
  height: 100%;
  border-radius: inherit;
  background: var(--ui-range-fill, #000);
}

.ui-range-slider__thumb {
  position: absolute;
  top: 50%;
  left: calc(var(--thumb) / 2 + (100% - var(--thumb)) * var(--ui-range-ratio, 0));
  width: var(--thumb);
  height: var(--thumb);
  border-radius: 50%;
  background: var(--ui-range-fill, #000);
  transform: translate(-50%, -50%);
  cursor: grab;
  pointer-events: none;

  .ui-range-slider.is-dragging & {
    cursor: grabbing;
  }

  .ui-range-slider.is-disabled & {
    cursor: not-allowed;
  }
}
</style>
