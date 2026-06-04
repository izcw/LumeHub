<template>
  <label class="ui-switch" :class="{ 'is-disabled': disabled }">
    <input
      type="checkbox"
      :checked="modelValue"
      :disabled="disabled"
      @change="onChange"
    />
    <span class="ui-switch__track"><span class="ui-switch__thumb" /></span>
    <span v-if="label" class="ui-switch__label">{{ label }}</span>
  </label>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: boolean
  disabled?: boolean
  label?: string
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
.ui-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  position: relative;

  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  input {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }
}

.ui-switch__track {
  position: relative;
  width: 36px;
  height: 20px;
  background: #d4d4d4;
  border-radius: 10px;
  transition: background 0.2s ease;
  flex-shrink: 0;

  input:checked + & {
    background: #333;
  }

  .is-disabled & {
    opacity: 0.6;
  }
}

.ui-switch__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  background: #fff;
  border-radius: 50%;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);

  input:checked + .ui-switch__track > & {
    transform: translateX(16px);
  }
}

.ui-switch__label {
  font-size: 12px;
  color: #666;
  line-height: 20px;
}
</style>
