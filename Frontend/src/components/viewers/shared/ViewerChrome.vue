<template>
  <div class="viewer-chrome">
    <div
      v-if="showClose"
      class="viewer-close"
      role="button"
      tabindex="0"
      aria-label="关闭"
      @click="emit('close')"
    >
      <img :src="iconClose" alt="" draggable="false" />
    </div>
    <div
      v-if="showPrev"
      class="viewer-nav viewer-prev"
      :class="{ 'is-hidden': !showPrevHighlight }"
      role="button"
      tabindex="0"
      aria-label="上一项"
      @click="emit('prev')"
    >
      <img :src="iconLeft" alt="" draggable="false" />
    </div>
    <div
      v-if="showNext"
      class="viewer-nav viewer-next"
      :class="{ 'is-hidden': !showNextHighlight }"
      role="button"
      tabindex="0"
      aria-label="下一项"
      @click="emit('next')"
    >
      <img :src="iconLeft" alt="" draggable="false" />
    </div>
    <div v-if="total > 1 || $slots.toolbar" class="viewer-bottom-bar">
      <div
        v-if="$slots.toolbar"
        class="viewer-toolbar"
        :class="{ 'is-hidden': !showBottomHighlight }"
      >
        <slot name="toolbar" />
      </div>
      <div v-if="total > 1" class="viewer-counter">
        {{ current }} / {{ total }}
      </div>
    </div>
    <div v-if="$slots.badge" class="viewer-type-badge" :aria-label="badgeLabel">
      <slot name="badge" />
    </div>
  </div>
</template>

<script setup lang="ts">
import iconClose from '@/assets/icon/close.svg?url'
import iconLeft from '@/assets/icon/left.svg?url'

withDefaults(
  defineProps<{
    showPrev: boolean
    showNext: boolean
    showPrevHighlight: boolean
    showNextHighlight: boolean
    /** 鼠标移入底部区域时显示工具栏 */
    showBottomHighlight?: boolean
    /** 1-based index for display */
    current: number
    total: number
    /** 左上角类型角标（如实况 / 视频） */
    badgeLabel?: string
    /** 是否显示右上角关闭按钮 */
    showClose?: boolean
  }>(),
  {
    showClose: true,
    showBottomHighlight: false,
  },
)

const emit = defineEmits<{
  close: []
  prev: []
  next: []
}>()
</script>

<style scoped lang="scss">
@use './viewer-chrome.scss';
</style>
