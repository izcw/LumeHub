<template>
  <div
    ref="rootEl"
    class="ui-tooltip"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
    @focusin="onEnter"
    @focusout="onLeave"
  >
    <slot />
    <Teleport to="body">
      <div
        v-if="visible"
        ref="panelEl"
        class="ui-tooltip__panel"
        :style="{ ...panelStyle, zIndex: String(resolvedZIndex) }"
        role="tooltip"
      >
        {{ text }}
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, onUnmounted, ref, watch, type Ref } from 'vue'
import { FLOATING_UI_Z_INDEX_KEY } from '@/components/viewers/shared/viewerLayers'

const props = withDefaults(
  defineProps<{
    text: string
    placement?: 'top' | 'bottom'
    disabled?: boolean
    showDelay?: number
    zIndex?: number
  }>(),
  {
    placement: 'top',
    disabled: false,
    showDelay: 180,
    zIndex: 9500,
  },
)

const injectedFloatingZ = inject<Ref<number | undefined>>(FLOATING_UI_Z_INDEX_KEY)
const resolvedZIndex = computed(() => injectedFloatingZ?.value ?? props.zIndex)

const rootEl = ref<HTMLElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const visible = ref(false)
const panelStyle = ref<Record<string, string>>({})

let showTimer: ReturnType<typeof setTimeout> | null = null

function clearShowTimer() {
  if (showTimer !== null) {
    clearTimeout(showTimer)
    showTimer = null
  }
}

function onEnter() {
  if (props.disabled || !props.text.trim()) return
  clearShowTimer()
  showTimer = setTimeout(() => {
    showTimer = null
    visible.value = true
  }, props.showDelay)
}

function onLeave() {
  clearShowTimer()
  visible.value = false
}

function updatePosition() {
  if (!visible.value) return
  const trigger = rootEl.value
  const panel = panelEl.value
  if (!trigger || !panel) return

  const rect = trigger.getBoundingClientRect()
  const panelRect = panel.getBoundingClientRect()
  const viewportW = window.innerWidth
  const viewportH = window.innerHeight
  const gap = 6
  const placeBelow = props.placement === 'bottom'
  const top = placeBelow
    ? rect.bottom + gap
    : Math.max(gap, rect.top - panelRect.height - gap)
  const left = Math.min(
    Math.max(gap, rect.left + rect.width / 2 - panelRect.width / 2),
    Math.max(gap, viewportW - panelRect.width - gap),
  )

  panelStyle.value = {
    top: `${Math.round(top)}px`,
    left: `${Math.round(left)}px`,
  }
}

function bindFloatingListeners() {
  window.addEventListener('scroll', updatePosition, true)
  window.addEventListener('resize', updatePosition)
}

function unbindFloatingListeners() {
  window.removeEventListener('scroll', updatePosition, true)
  window.removeEventListener('resize', updatePosition)
}

watch(visible, async (v) => {
  if (!v) {
    unbindFloatingListeners()
    return
  }
  await nextTick()
  updatePosition()
  bindFloatingListeners()
})

onUnmounted(() => {
  clearShowTimer()
  unbindFloatingListeners()
})
</script>

<style scoped lang="scss">
.ui-tooltip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.ui-tooltip__panel {
  position: fixed;
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(18, 18, 18, 0.92);
  color: rgba(255, 255, 255, 0.96);
  font-size: 11px;
  line-height: 1.35;
  font-weight: 500;
  white-space: nowrap;
  pointer-events: none;
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.24);
}
</style>
