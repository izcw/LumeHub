<template>
  <button
    class="ui-button"
    :class="[`is-${type}`, `is-${size}`, { 'has-icon': !!iconSrc, 'is-icon-only': iconOnly }]"
    :style="buttonStyle"
    :type="nativeType"
    :disabled="disabled"
  >
    <img
      v-if="iconSrc && iconPosition === 'left'"
      class="ui-button__icon"
      :class="iconClass"
      :src="iconSrc"
      :alt="iconAlt"
      aria-hidden="true"
    />
    <span v-if="!iconOnly" class="ui-button__label">
      <slot />
    </span>
    <img
      v-if="iconSrc && iconPosition === 'right'"
      class="ui-button__icon"
      :class="iconClass"
      :src="iconSrc"
      :alt="iconAlt"
      aria-hidden="true"
    />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type ButtonSize = string | number

const props = withDefaults(
  defineProps<{
    type?: 'primary' | 'info'
    size?: 'default' | 'small'
    nativeType?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    iconSrc?: string
    iconAlt?: string
    iconClass?: string
    iconPosition?: 'left' | 'right'
    iconOnly?: boolean
    width?: ButtonSize
  }>(),
  {
    type: 'primary',
    size: 'default',
    nativeType: 'button',
    disabled: false,
    iconSrc: '',
    iconAlt: '',
    iconClass: '',
    iconPosition: 'left',
    iconOnly: false,
    width: '',
  },
)

const buttonStyle = computed(() => {
  if (!props.width) return undefined
  const normalized = normalizeSize(props.width)
  return {
    width: normalized,
    minWidth: normalized,
  }
})

function normalizeSize(value: ButtonSize): string {
  return typeof value === 'number' ? `${value}px` : value
}
</script>

<style scoped lang="scss">
.ui-button {
  flex-shrink: 0;
  width: auto;
  min-width: 60px;
  padding: 10px 12px;
  height: 36px;
  margin: 0;
  border: none;
  border-radius: 6px;
  background: #000000;
  color: #ffffff;
  font-size: 12px;
  cursor: pointer;
  transition:
    background 0.2s ease,
    opacity 0.2s ease,
    transform 0.14s ease;

  &:hover:not(:disabled) {
    background: #1d1d1d;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &:active:not(:disabled) {
    background: #2a2a2a;
    transform: scale(0.995);
  }

  &:focus-visible {
    outline: 2px solid rgba(0, 0, 0, 0.35);
    outline-offset: 1px;
  }

  &.is-info {
    background: #f5f5f7;
    color: #1d1d1f;

    &:hover:not(:disabled) {
      background: #ececf0;
    }

    &:active:not(:disabled) {
      background: #e3e3e8;
      transform: scale(0.995);
    }
  }
}

.ui-button.is-small {
  width: auto;
  min-width: 50px;
  height: 30px;
  padding: 6px 10px;
}

.ui-button__label {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.ui-button__icon {
  width: 18px;
  height: 18px;
  display: block;
  object-fit: contain;
  flex-shrink: 0;
}

.ui-button.has-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.ui-button.is-icon-only {
  padding: 0;
}
</style>
