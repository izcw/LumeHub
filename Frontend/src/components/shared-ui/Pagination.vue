<template>
  <div class="ui-pagination" :class="{ 'is-compact': compact }" role="navigation" :aria-label="ariaLabel">
    <button
      type="button"
      class="ui-pagination__btn"
      :disabled="!canPrev"
      aria-label="上一页"
      @click="goPrev"
    >
      <img :src="iconLeft" alt="" draggable="false" />
    </button>
    <span class="ui-pagination__info">
      <span class="ui-pagination__current">{{ currentPage }}</span>
      <span class="ui-pagination__sep">/</span>
      <span class="ui-pagination__total">{{ safeTotalPages }}</span>
      <span v-if="showCount && typeof totalCount === 'number'" class="ui-pagination__dot" />
      <span v-if="showCount && typeof totalCount === 'number'" class="ui-pagination__count">
        {{ totalCount }} {{ countUnit }}
      </span>
    </span>
    <button
      type="button"
      class="ui-pagination__btn ui-pagination__btn--next"
      :disabled="!canNext"
      aria-label="下一页"
      @click="goNext"
    >
      <img :src="iconLeft" alt="" draggable="false" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import iconLeft from '@/assets/icon/left.svg?url'

const props = withDefaults(
  defineProps<{
    currentPage: number
    totalPages: number
    totalCount?: number
    showCount?: boolean
    countUnit?: string
    compact?: boolean
    ariaLabel?: string
  }>(),
  {
    totalCount: undefined,
    showCount: true,
    countUnit: '项',
    compact: false,
    ariaLabel: '分页',
  },
)

const emit = defineEmits<{
  'update:currentPage': [value: number]
  prev: []
  next: []
}>()

const safeTotalPages = computed(() => Math.max(1, props.totalPages))
const canPrev = computed(() => props.currentPage > 1 && safeTotalPages.value > 0)
const canNext = computed(() => props.currentPage < safeTotalPages.value)

function goPrev() {
  if (!canPrev.value) return
  emit('update:currentPage', props.currentPage - 1)
  emit('prev')
}

function goNext() {
  if (!canNext.value) return
  emit('update:currentPage', props.currentPage + 1)
  emit('next')
}
</script>

<style scoped lang="scss">
.ui-pagination {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: clamp(4px, 1.2vw, 10px);
  min-width: 0;
}

.ui-pagination__info {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: clamp(4px, 1.2vw, 8px);
  padding: 0 10px;
  min-width: 148px;
  height: 32px;
  line-height: 1;
  color: #2f2f2f;
}

.ui-pagination__current {
  color: #111;
  font-size: 12px;
  // font-weight: 700;
}

.ui-pagination__sep {
  color: #8f8f8f;
  font-size: 12px;
  margin: 0 1px;
}

.ui-pagination__total {
  color: #595959;
  font-size: 12px;
  // font-weight: 500;
}

.ui-pagination__dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #c4c4c4;
}

.ui-pagination__count {
  color: #8b8b8b;
  font-size: 12px;
  // font-weight: 400;
  white-space: nowrap;
}

.ui-pagination__btn {
  width: 32px;
  height: 32px;
  padding: 0;
  line-height: 1;
  color: #111;
  background: #fff;
  border: 1px solid #1f1f1f;
  border-radius: 50%;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease,
    color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;

  img {
    width: 14px;
    height: 14px;
    display: block;
  }

  &--next img {
    transform: scaleX(-1);
  }

  &:hover:not(:disabled) {
    background-color: #111;
    border-color: #111;
    color: #fff;
    box-shadow: 0 6px 14px rgba(0, 0, 0, 0.18);
    transform: translateY(-1px);

    img {
      filter: invert(1);
    }
  }

  &:disabled {
    background: #f3f3f3;
    border-color: #ddd;
    color: #bdbdbd;
    box-shadow: none;
    cursor: not-allowed;
    transform: none;

    img {
      opacity: 0.4;
    }
  }
}

.ui-pagination.is-compact {
  .ui-pagination__info {
    min-width: 0;
    padding: 0 4px;
    gap: 6px;
  }
}
</style>
