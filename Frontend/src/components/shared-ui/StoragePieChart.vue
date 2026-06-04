<template>
  <div class="storage-pie-chart" role="img" :aria-label="ariaLabel">
    <div class="storage-pie-chart__visual">
      <div class="storage-pie-chart__ring" :style="ringStyle">
        <div class="storage-pie-chart__hole">
          <span class="storage-pie-chart__percent">{{ percentLabel }}</span>
          <span class="storage-pie-chart__caption">已使用</span>
        </div>
      </div>
    </div>
    <ul class="storage-pie-chart__legend">
      <li class="storage-pie-chart__legend-item">
        <span class="storage-pie-chart__legend-head">
          <span class="storage-pie-chart__dot is-used" :style="{ background: usedColor }" />
          <span class="storage-pie-chart__legend-label">已用</span>
        </span>
        <span class="storage-pie-chart__legend-value">{{ formatFileSize(usedBytes) }}</span>
      </li>
      <li class="storage-pie-chart__legend-item">
        <span class="storage-pie-chart__legend-head">
          <span class="storage-pie-chart__dot is-free" />
          <span class="storage-pie-chart__legend-label">可用</span>
        </span>
        <span class="storage-pie-chart__legend-value">{{ formatFileSize(availableBytes) }}</span>
      </li>
      <li class="storage-pie-chart__legend-item">
        <span class="storage-pie-chart__legend-head">
          <span class="storage-pie-chart__dot is-total" />
          <span class="storage-pie-chart__legend-label">总</span>
        </span>
        <span class="storage-pie-chart__legend-value">{{ formatFileSize(quotaBytes) }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatFileSize } from '@/utils/formatFileSize'

const props = defineProps<{
  usedBytes: number
  quotaBytes: number
  availableBytes: number
  usedPercent: number
}>()

const clampedPercent = computed(() => {
  const p = props.usedPercent
  if (!Number.isFinite(p)) return 0
  return Math.min(100, Math.max(0, p))
})

const usedColor = computed(() => {
  const p = clampedPercent.value
  if (p >= 100) return '#e74c3c'
  if (p >= 85) return '#f5a623'
  return '#5a82ff'
})

const ringStyle = computed(() => {
  const deg = clampedPercent.value * 3.6
  return {
    background: `conic-gradient(from -90deg, ${usedColor.value} 0deg ${deg}deg, #eef1f6 ${deg}deg 360deg)`,
  }
})

const percentLabel = computed(() => {
  const p = clampedPercent.value
  return p < 10 ? `${p.toFixed(1)}%` : `${Math.round(p)}%`
})

const ariaLabel = computed(() => {
  return `存储已用 ${percentLabel.value}，已用 ${formatFileSize(props.usedBytes)}，可用 ${formatFileSize(props.availableBytes)}，总配额 ${formatFileSize(props.quotaBytes)}`
})
</script>

<style scoped lang="scss">
.storage-pie-chart {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  margin-top: 8px;
  width: 100%;
}

.storage-pie-chart__visual {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
}

.storage-pie-chart__ring {
  width: 156px;
  height: 156px;
  border-radius: 50%;
  position: relative;
  transition: background 0.4s ease;
  box-shadow: 0 8px 24px rgba(90, 130, 255, 0.12);
}

.storage-pie-chart__hole {
  position: absolute;
  inset: 26px;
  border-radius: 50%;
  background: #fff;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.04);
}

.storage-pie-chart__percent {
  font-size: 22px;
  font-weight: 700;
  color: #222;
  line-height: 1.1;
}

.storage-pie-chart__caption {
  font-size: 11px;
  color: #9ca3af;
  letter-spacing: 0.02em;
}

.storage-pie-chart__legend {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  justify-content: center;
  gap: 28px;
  width: 100%;
  flex-wrap: wrap;
}

.storage-pie-chart__legend-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #444;
  min-width: 72px;
}

.storage-pie-chart__legend-head {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.storage-pie-chart__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;

  &.is-free {
    background: #eef1f6;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.06);
  }

  &.is-total {
    background: transparent;
    box-shadow: inset 0 0 0 1.5px #d1d5db;
  }
}

.storage-pie-chart__legend-label {
  color: #666;
}

.storage-pie-chart__legend-value {
  font-weight: 600;
  color: #222;
  white-space: nowrap;
}
</style>
