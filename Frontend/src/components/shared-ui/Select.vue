<template>
  <div ref="rootEl" class="ui-select" :class="{ 'is-open': isOpen, 'is-disabled': disabled }">
    <button
      type="button"
      class="ui-select__trigger"
      :disabled="disabled"
      aria-haspopup="listbox"
      :aria-expanded="isOpen"
      :aria-label="triggerLabel"
      @click="toggleOpen"
    >
      <span class="ui-select__value">{{ currentLabel }}</span>
      <span class="ui-select__chevron" aria-hidden="true" />
    </button>
    <Teleport to="body">
      <ul
        v-if="isOpen"
        ref="panelEl"
        class="ui-select__menu"
        :style="{ ...menuStyle, zIndex: String(menuZIndex) }"
        role="listbox"
        :aria-label="menuAriaLabel"
      >
        <li v-for="item in options" :key="item.value" class="ui-select__menu-item" role="none">
          <button
            type="button"
            class="ui-select__option"
            :class="{ 'is-disabled': item.disabled }"
            role="option"
            :aria-selected="modelValue === item.value"
            :aria-disabled="item.disabled || undefined"
            @click="onPick(item)"
          >
            {{ item.label }}
          </button>
        </li>
      </ul>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: string
    options: Array<{ label: string; value: string; disabled?: boolean }>
    disabled?: boolean
    open?: boolean
    triggerLabel?: string
    menuAriaLabel?: string
    menuZIndex?: number
  }>(),
  {
    disabled: false,
    open: undefined,
    triggerLabel: '',
    menuAriaLabel: '',
    menuZIndex: 6000,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  toggle: []
  select: [value: string]
  outside: []
}>()

const internalOpen = ref(false)
const rootEl = ref<HTMLElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const menuStyle = ref<Record<string, string>>({})
const isControlled = computed(() => typeof props.open === 'boolean')
const isOpen = computed(() => (isControlled.value ? !!props.open : internalOpen.value))
const currentLabel = computed(() => {
  const hit = props.options.find((item) => item.value === props.modelValue)
  return hit?.label ?? ''
})

function toggleOpen() {
  if (props.disabled) return
  emit('toggle')
  if (isControlled.value) return
  internalOpen.value = !internalOpen.value
}

function onPick(item: { value: string; disabled?: boolean }) {
  if (item.disabled) return
  emit('update:modelValue', item.value)
  emit('select', item.value)
  if (!isControlled.value) internalOpen.value = false
}

function onDocPointerdown(event: PointerEvent) {
  if (!isOpen.value) return
  const target = event.target as Node | null
  if (!target) return
  if (rootEl.value?.contains(target) || panelEl.value?.contains(target)) return
  emit('outside')
  if (!isControlled.value) internalOpen.value = false
}

function updateMenuPosition() {
  if (!isOpen.value) return
  const trigger = rootEl.value
  const panel = panelEl.value
  if (!trigger || !panel) return
  const rect = trigger.getBoundingClientRect()
  const panelRect = panel.getBoundingClientRect()
  const viewportW = window.innerWidth
  const viewportH = window.innerHeight
  const gap = 6
  const placeBelow = viewportH - rect.bottom >= panelRect.height + gap
  const top = placeBelow ? rect.bottom + gap : Math.max(gap, rect.top - panelRect.height - gap)
  const left = Math.min(
    Math.max(gap, rect.left),
    Math.max(gap, viewportW - Math.max(rect.width, panelRect.width) - gap),
  )
  menuStyle.value = {
    top: `${Math.round(top)}px`,
    left: `${Math.round(left)}px`,
    minWidth: `${Math.round(rect.width)}px`,
  }
}

function bindFloatingListeners() {
  window.addEventListener('scroll', updateMenuPosition, true)
  window.addEventListener('resize', updateMenuPosition)
}

function unbindFloatingListeners() {
  window.removeEventListener('scroll', updateMenuPosition, true)
  window.removeEventListener('resize', updateMenuPosition)
}

watch(isOpen, async (v) => {
  if (!v) {
    unbindFloatingListeners()
    return
  }
  await nextTick()
  updateMenuPosition()
  bindFloatingListeners()
})

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerdown)
})

onUnmounted(() => {
  unbindFloatingListeners()
  document.removeEventListener('pointerdown', onDocPointerdown)
})

defineExpose({
  rootEl,
  panelEl,
})
</script>

<style scoped lang="scss">
.ui-select {
  position: relative;
  width: 100%;
  box-sizing: border-box;

  &.is-disabled {
    opacity: 0.6;

    .ui-select__trigger {
      cursor: not-allowed;
    }
  }
}

.ui-select__trigger {
  width: 100%;
  box-sizing: border-box;
  padding: 0 12px;
  height: 36px;
  font-size: 12px;
  line-height: 36px;
  border: 1px solid #ccc;
  outline: none;
  color: #111111;
  background: #fff;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  cursor: pointer;

  &:disabled {
    cursor: not-allowed;
  }
}

.ui-select__value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-select__chevron {
  width: 10px;
  height: 6px;
  flex-shrink: 0;
  background: #9a9a9a;
  clip-path: polygon(0 0, 100% 0, 50% 100%);
  transition: transform 0.15s ease;
}

.ui-select.is-open .ui-select__chevron {
  transform: rotate(180deg);
}

.ui-select__menu {
  position: fixed;
  margin: 0;
  padding: 4px;
  list-style: none;
  border: 1px solid #e9e9e9;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.12);
  box-sizing: border-box;
}

.ui-select__menu-item {
  margin: 0;
  padding: 0;
}

.ui-select__option {
  display: block;
  width: 100%;
  margin: 0;
  padding: 7px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #555;
  font-size: 12px;
  text-align: left;
  cursor: pointer;

  &:hover {
    background: #f5f5f5;
    color: #222;
  }

  &[aria-selected='true'] {
    background: #efefee;
    color: #111;
    font-weight: 600;
  }

  &.is-disabled {
    color: #aaa;
    cursor: not-allowed;

    &:hover {
      background: #f5f5f5;
      color: #aaa;
    }

    &[aria-selected='true'],
    &[aria-selected='true']:hover {
      background: #efefee;
      color: #aaa;
    }
  }
}
</style>
