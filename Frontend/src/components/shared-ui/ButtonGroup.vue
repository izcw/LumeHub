<template>
  <div class="ui-button-group" :class="`is-${variant}`" role="group" :aria-label="ariaLabel">
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'plain' | 'toolbar'
    ariaLabel?: string
  }>(),
  {
    variant: 'plain',
    ariaLabel: undefined,
  },
)
</script>

<style scoped lang="scss">
.ui-button-group {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;

  &.is-toolbar {
    height: 36px;
    min-height: 36px;
    padding: 3px;
    border-radius: 8px;
    background: #111111;
    box-sizing: border-box;
    box-shadow: 0 10px 28px rgba(0, 0, 0, 0.22);
  }
}
</style>
