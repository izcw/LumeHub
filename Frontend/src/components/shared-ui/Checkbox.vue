<template>
  <div class="ui-checkbox">
    <input
      :id="inputId"
      class="ui-checkbox__input"
      :checked="modelValue"
      :disabled="disabled"
      type="checkbox"
      @change="onChange"
    />
    <span class="ui-checkbox__text" :class="{ 'is-disabled': disabled }">
      <slot />
    </span>
  </div>
</template>

<script setup lang="ts">
const inputId = `ui-checkbox-${Math.random().toString(36).slice(2, 10)}`

defineProps<{
  modelValue: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function onChange(event: Event) {
  const target = event.target as HTMLInputElement | null
  emit('update:modelValue', !!target?.checked)
}
</script>

<style scoped lang="scss">
.ui-checkbox {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  line-height: 1.4;
  color: #333;
}

.ui-checkbox__input {
  width: 18px;
  height: 18px;
  margin: 0;
  border-radius: 4px;
  accent-color: #111;
  flex-shrink: 0;
  cursor: pointer;

  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
}

.ui-checkbox__text {
  display: inline-block;
  cursor: default;

  &.is-disabled {
    color: #999;
  }
}
</style>
