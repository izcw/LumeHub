<template>
  <div ref="rootEl" class="ui-popconfirm">
    <div class="ui-popconfirm__trigger" @click.stop="openPanel" @pointerdown.stop>
      <slot name="trigger" />
    </div>
    <Teleport to="body">
      <div
        v-if="open"
        ref="panelEl"
        class="ui-popconfirm__panel"
        :style="{ ...panelStyle, zIndex: String(resolvedZIndex) }"
        @click.stop
        @pointerdown.stop
      >
        <p class="ui-popconfirm__title">{{ title }}</p>
        <div class="ui-popconfirm__actions">
          <Button class="ui-popconfirm__btn" type="info" native-type="button" @click="cancel">
            取消
          </Button>
          <Button class="ui-popconfirm__btn" native-type="button" @click="confirm">
            {{ confirmText }}
          </Button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import Button from '@/components/shared-ui/Button.vue'
import { FLOATING_UI_Z_INDEX_KEY } from '@/components/viewers/shared/viewerLayers'

const props = withDefaults(
  defineProps<{
    title: string
    confirmText?: string
    zIndex?: number
  }>(),
  {
    confirmText: '确定',
    zIndex: 9000,
  },
)

const injectedFloatingZ = inject<Ref<number | undefined>>(FLOATING_UI_Z_INDEX_KEY)
const resolvedZIndex = computed(() => injectedFloatingZ?.value ?? props.zIndex)

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const rootEl = ref<HTMLElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const panelStyle = ref<Record<string, string>>({})

function openPanel() {
  open.value = true
}

function confirm() {
  open.value = false
  emit('confirm')
}

function cancel() {
  open.value = false
  emit('cancel')
}

function onDocPointerDown(event: PointerEvent) {
  if (!open.value) return
  const target = event.target as Node | null
  if (!target) return
  if (rootEl.value?.contains(target) || panelEl.value?.contains(target)) return
  open.value = false
}

function updatePosition() {
  if (!open.value) return
  const trigger = rootEl.value
  const panel = panelEl.value
  if (!trigger || !panel) return
  const rect = trigger.getBoundingClientRect()
  const panelRect = panel.getBoundingClientRect()
  const viewportW = window.innerWidth
  const viewportH = window.innerHeight

  const spaceBelow = viewportH - rect.bottom
  const placeBelow = spaceBelow >= panelRect.height + 8
  const top = placeBelow ? rect.bottom + 8 : Math.max(8, rect.top - panelRect.height - 8)
  const left = Math.min(
    Math.max(8, rect.right - panelRect.width),
    Math.max(8, viewportW - panelRect.width - 8),
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

watch(open, async (v) => {
  if (!v) {
    unbindFloatingListeners()
    return
  }
  await nextTick()
  updatePosition()
  bindFloatingListeners()
})

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerDown)
})

onUnmounted(() => {
  unbindFloatingListeners()
  document.removeEventListener('pointerdown', onDocPointerDown)
})
</script>

<style scoped lang="scss">
.ui-popconfirm {
  position: relative;
  display: inline-flex;
}

.ui-popconfirm__trigger {
  display: inline-flex;
}

.ui-popconfirm__panel {
  position: fixed;
  min-width: 180px;
  border: 1px solid #efefef;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 12px 26px rgba(0, 0, 0, 0.14);
  padding: 10px;
  pointer-events: auto;
}

.ui-popconfirm__title {
  margin: 0;
  font-size: 12px;
  color: #333;
  line-height: 1.45;
}

.ui-popconfirm__actions {
  margin-top: 8px;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.ui-popconfirm__btn {
  width: auto;
  height: 30px;
  padding: 0 10px;
}
</style>
