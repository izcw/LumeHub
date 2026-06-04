<template>
  <div class="toolbar-select" :class="{ 'is-menu-open': open }">
    <img v-if="iconSrc" class="toolbar-select__icon" :src="iconSrc" alt="" aria-hidden="true" />
    <span
      v-else-if="iconText"
      class="toolbar-select__icon toolbar-select__icon--glyph"
      aria-hidden="true"
    >
      {{ iconText }}
    </span>
    <Select
      ref="selectRef"
      class="toolbar-select__control"
      :open="open"
      :model-value="modelValue"
      :options="options"
      :trigger-label="triggerLabel"
      :menu-aria-label="menuAriaLabel"
      @toggle="$emit('toggle')"
      @select="$emit('select', $event)"
      @outside="$emit('toggle')"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Select from '@/components/shared-ui/Select.vue'

defineProps<{
  open: boolean
  modelValue: string
  displayLabel?: string
  triggerLabel: string
  menuAriaLabel: string
  iconSrc?: string
  iconText?: string
  options: Array<{ value: string; label: string }>
}>()

defineEmits<{
  toggle: []
  select: [value: string]
}>()

const selectRef = ref<{
  rootEl: HTMLElement | null
  panelEl: HTMLElement | null
} | null>(null)

defineExpose({
  get rootEl() {
    return selectRef.value?.rootEl ?? null
  },
  get panelEl() {
    return selectRef.value?.panelEl ?? null
  },
})
</script>

<style scoped lang="scss">
.toolbar-select {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  height: 36px;
  min-height: 36px;
  padding: 0 8px 0 6px;
  border-radius: 8px;
  background: #111111;
  box-sizing: border-box;

  &.is-menu-open {
    z-index: 1200;
  }
}

.toolbar-select__icon {
  width: 18px;
  height: 18px;
  display: block;
  object-fit: contain;
  flex-shrink: 0;
  opacity: 0.55;
  filter: invert(1);
}

.toolbar-select__icon--glyph {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1;
  color: #d4d4d4;
  opacity: 0.85;
  filter: none;
}

.toolbar-select__control {
  min-width: 0;
}

:deep(.toolbar-select__control .ui-select) {
  width: auto;
}

:deep(.toolbar-select__control .ui-select__trigger) {
  min-width: 4.75rem;
  height: 30px;
  min-height: 30px;
  padding: 0 6px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #d4d4d4;
  font-size: 14px;
  line-height: 30px;

  &:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #fff;
  }

  &:focus-visible {
    outline: none;
    background: rgba(255, 255, 255, 0.1);
  }
}

:deep(.toolbar-select__control .ui-select__chevron) {
  background: #b5b5b5;
}

:deep(.ui-select__menu) {
  min-width: 7.5rem;
  background: #0d0d0d;
  border: none;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.45);
}

:deep(.ui-select__option) {
  padding: 8px 10px;
  font-size: 14px;
  line-height: 1.25;
  color: #c6c6c6;

  &:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #fff;
  }

  &:focus-visible {
    outline: none;
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
  }

  &[aria-selected='true'] {
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
    font-weight: 500;
  }
}
</style>
