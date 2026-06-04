<template>
  <Teleport to="body">
    <Transition :name="props.transitionName">
      <div v-if="props.open" class="dialog-root" :style="rootStyle">
        <div
          class="dialog-panel"
          :class="{
            'is-mobile-fullscreen': props.fullscreenOnMobile,
            'is-fullscreen': props.fullscreen,
          }"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="resolvedAriaLabelledby"
          :style="panelStyle"
          @click.stop
        >
          <button
            v-if="props.showClose"
            type="button"
            class="close-btn"
            aria-label="关闭"
            @click="emitClose"
          />
          <div v-if="hasTitle" class="dialog-header">
            <div :id="resolvedTitleId" class="dialog-title">
              <slot name="title">
                {{ props.title }}
              </slot>
            </div>
          </div>
          <div class="dialog-body" :class="{ 'is-body-padded': props.bodyPadded }">
            <slot />
          </div>
          <div v-if="props.showActions" class="dialog-actions">
            <Button
              class="dialog-action-btn"
              type="info"
              native-type="button"
              :disabled="props.cancelDisabled"
              @click="onCancel"
            >
              {{ props.cancelText }}
            </Button>
            <Button
              class="dialog-action-btn"
              native-type="button"
              :disabled="props.confirmDisabled"
              @click="onConfirm"
            >
              {{ props.confirmText }}
            </Button>
          </div>
        </div>
        <div
          v-if="props.showMask"
          class="dialog-mask"
          aria-hidden="true"
          @click.self="onMaskClick"
        />
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, useSlots, watch } from 'vue'
import Button from '@/components/shared-ui/Button.vue'
import { lockBodyScroll, unlockBodyScroll } from '@/utils/bodyScrollLock'

type DialogSize = string | number

const props = withDefaults(
  defineProps<{
    open: boolean
    width?: DialogSize
    height?: DialogSize
    zIndex?: number
    showClose?: boolean
    showMask?: boolean
    closeOnMask?: boolean
    fullscreenOnMobile?: boolean
    /** 桌面端也占满视口 */
    fullscreen?: boolean
    transitionName?: string
    ariaLabelledby?: string
    title?: string
    titleId?: string
    bodyPadded?: boolean
    showActions?: boolean
    cancelText?: string
    confirmText?: string
    cancelDisabled?: boolean
    confirmDisabled?: boolean
    lockBodyScroll?: boolean
  }>(),
  {
    width: '340px',
    height: '400px',
    zIndex: 2450,
    showClose: true,
    showMask: true,
    closeOnMask: true,
    fullscreenOnMobile: true,
    fullscreen: false,
    transitionName: 'dialog-mount',
    ariaLabelledby: undefined,
    title: '',
    titleId: undefined,
    bodyPadded: true,
    showActions: true,
    cancelText: '取消',
    confirmText: '确认',
    cancelDisabled: false,
    confirmDisabled: false,
    lockBodyScroll: true,
  },
)

let bodyScrollLockedByThis = false

function syncBodyScrollLock(open: boolean) {
  if (!props.lockBodyScroll) {
    if (bodyScrollLockedByThis) {
      unlockBodyScroll()
      bodyScrollLockedByThis = false
    }
    return
  }
  if (open && !bodyScrollLockedByThis) {
    lockBodyScroll()
    bodyScrollLockedByThis = true
    return
  }
  if (!open && bodyScrollLockedByThis) {
    unlockBodyScroll()
    bodyScrollLockedByThis = false
  }
}

watch(
  () => props.open,
  (open) => {
    syncBodyScrollLock(open)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  syncBodyScrollLock(false)
})

const slots = useSlots()
const autoTitleId = `dialog-title-${Math.random().toString(36).slice(2, 10)}`

const hasTitle = computed(() => !!props.title || !!slots.title)
const resolvedTitleId = computed(() => props.titleId ?? autoTitleId)
const resolvedAriaLabelledby = computed(() => {
  if (props.ariaLabelledby) return props.ariaLabelledby
  return hasTitle.value ? resolvedTitleId.value : undefined
})

const panelStyle = computed(() => {
  const width = normalizeSize(props.width)
  const height = normalizeSize(props.height)
  return {
    '--dialog-width': width,
    '--dialog-height': height,
  }
})

const rootStyle = computed(() => ({
  '--dialog-z-index': String(props.zIndex),
}))

const emit = defineEmits<{
  close: []
  cancel: []
  confirm: []
}>()

function normalizeSize(value: DialogSize): string {
  return typeof value === 'number' ? `${value}px` : value
}

function emitClose() {
  emit('close')
}

function onMaskClick() {
  if (!props.closeOnMask) return
  emitClose()
}

function onCancel() {
  emit('cancel')
}

function onConfirm() {
  emit('confirm')
}
</script>

<style scoped lang="scss">
$ease-brand: cubic-bezier(0.22, 1, 0.36, 1);

.dialog-root {
  width: 100%;
  height: 100%;
  position: fixed;
  inset: 0;
  z-index: var(--dialog-z-index, 2450);
  display: flex;
  justify-content: center;
  align-items: center;
  box-sizing: border-box;
  pointer-events: none;
  overflow: hidden;

  .dialog-panel {
    width: min(var(--dialog-width), calc(100vw - 24px));
    height: min(var(--dialog-height), calc(100vh - 24px));
    margin: 0 auto;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    background: #ffffff;
    border-radius: 6px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
    pointer-events: auto;
    position: relative;
    z-index: 1;
  }

  .dialog-header {
    padding: 20px 24px 0;
    margin-bottom: 12px;
  }

  .dialog-title {
    margin: 0;
    font-size: 18px;
    font-weight: bold;
    color: #000000;
    line-height: 34px;
  }

  .dialog-body {
    width: 100%;
    flex: 1;
    min-height: 0;
    box-sizing: border-box;
    overflow-x: hidden;
    overflow-y: auto;
  }

  .dialog-body.is-body-padded {
    padding: 0 24px;
  }

  .dialog-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    padding: 0 30px 20px;
    margin-top: 12px;
    box-sizing: border-box;
  }

  .dialog-action-btn {
    width: 100%;
  }

  .close-btn {
    position: absolute;
    top: 16px;
    right: 16px;
    width: 18px;
    height: 18px;
    padding: 0;
    border: none;
    background: #ffffff;
    color: #7a7a7a;
    cursor: pointer;
    transition:
      border-color 0.18s ease,
      color 0.18s ease;
    z-index: 2;

    &::before,
    &::after {
      content: '';
      position: absolute;
      left: 50%;
      top: 50%;
      width: 18px;
      height: 1.6px;
      border-radius: 1px;
      background: currentColor;
    }

    &::before {
      transform: translate(-50%, -50%) rotate(45deg);
    }

    &::after {
      transform: translate(-50%, -50%) rotate(-45deg);
    }

    &:hover {
      border-color: #9ca3af;
      color: #111111;
    }

    &:focus-visible {
      outline: 2px solid rgba(0, 113, 227, 0.5);
      outline-offset: 2px;
    }
  }

  .dialog-mask {
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, 0.2);
    pointer-events: auto;
    backdrop-filter: blur(4px) saturate(1);
  }
}

.dialog-mount-enter-active .dialog-mask,
.dialog-mount-leave-active .dialog-mask {
  transition: opacity 0.3s $ease-brand;
}

.dialog-mount-enter-active .dialog-panel,
.dialog-mount-leave-active .dialog-panel {
  transition:
    transform 0.42s $ease-brand,
    opacity 0.32s $ease-brand;
}

.dialog-mount-enter-from .dialog-mask,
.dialog-mount-leave-to .dialog-mask {
  opacity: 0;
}

.dialog-mount-enter-from .dialog-panel,
.dialog-mount-leave-to .dialog-panel {
  transform: translateY(10px);
  opacity: 0;
}

.dialog-mount-enter-to .dialog-mask,
.dialog-mount-leave-from .dialog-mask {
  opacity: 1;
}

.dialog-mount-enter-to .dialog-panel,
.dialog-mount-leave-from .dialog-panel {
  transform: translateY(0);
  opacity: 1;
}

@media (max-width: 719px) {
  .dialog-root {
    .dialog-panel.is-mobile-fullscreen {
      width: 100%;
      height: 100%;
      border-radius: 0;
    }

    /* .dialog-header {
      padding: 22px 24px 0;
    } */
  }
}

.dialog-root {
  .dialog-panel.is-fullscreen {
    width: 100vw;
    height: 100vh;
    max-width: 100vw;
    max-height: 100vh;
    border-radius: 0;
  }
}
</style>
