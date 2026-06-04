<template>
  <details class="ui-details" :open="open" @toggle="onToggle">
    <summary class="ui-details__summary">
      <slot name="summary">{{ summary }}</slot>
    </summary>
    <div class="ui-details__body">
      <slot />
    </div>
  </details>
</template>

<script setup lang="ts">
defineProps<{
  /** 折叠标题；也可用 #summary 插槽自定义 */
  summary?: string
  /** 是否展开；配合 v-model:open 使用 */
  open?: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

function onToggle(event: Event) {
  emit('update:open', (event.target as HTMLDetailsElement).open)
}
</script>

<style scoped lang="scss">
.ui-details {
  margin-bottom: 14px;
  border: 1px solid #ececec;
  border-radius: 6px;
  background: #fafafa;
  overflow: hidden;

  &[open] {
    background: #fff;
  }
}

.ui-details__summary {
  padding: 10px 12px;
  font-size: 12px;
  font-weight: 600;
  color: #111;
  cursor: pointer;
  list-style: none;
  user-select: none;

  &::-webkit-details-marker {
    display: none;
  }

  &::after {
    content: '+';
    float: right;
    color: #888;
    font-weight: 400;
  }
}

.ui-details[open] .ui-details__summary::after {
  content: '−';
}

.ui-details__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0 12px 12px;
}
</style>
