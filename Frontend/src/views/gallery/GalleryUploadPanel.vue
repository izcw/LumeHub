<template>
  <div
    v-if="siteFileDragActive && visible"
    class="site-upload-drop-mask"
    aria-hidden="true"
  >
    <div class="site-upload-drop-mask__inner">
      <img src="@/assets/icon/upload.svg" alt="" width="24" height="24" />
      <span>松开鼠标，上传文件</span>
    </div>
  </div>

  <div
    v-if="visible"
    class="upload-float-panel"
    :class="{ 'is-collapsed': collapsed, 'is-dragging': dragging }"
    :style="panelStyle"
  >
    <button
      v-if="collapsed"
      type="button"
      class="upload-float-panel__bubble"
      aria-label="展开上传面板"
      @pointerdown="$emit('panel-drag-start', $event)"
      @click="$emit('bubble-click', $event)"
    >
      <img src="@/assets/icon/upload.svg" alt="" width="20" height="20" aria-hidden="true" />
    </button>
    <template v-else>
      <div class="upload-float-panel__toolbar">
        <button
          type="button"
          class="upload-float-panel__drag-handle"
          aria-label="拖拽上传面板"
          @pointerdown="$emit('panel-drag-start', $event)"
        >
          <img src="@/assets/icon/drag.svg" alt="" width="14" height="14" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="upload-float-panel__toggle-btn"
          aria-label="缩小上传面板"
          @click="$emit('toggle-panel')"
        >
          <img :src="reduceIcon" alt="" width="14" height="14" aria-hidden="true" />
        </button>
      </div>
      <div class="upload-float-panel__body">
        <div class="upload-float-panel__hint">
          拖入或点击上传；同主名图片 + MOV 将识别为实况图
        </div>
        <div class="upload-float-panel__upload-btn" @click="$emit('open-picker')">
          <img src="@/assets/icon/upload.svg" alt="" width="18" height="18" aria-hidden="true" />
          <span>{{
            uploadInProgressCount > 0 ? `上传中 ${uploadInProgressCount}` : '上传文件'
          }}</span>
        </div>
        <div v-if="tasks.length > 0" class="upload-float-panel__tasks">
          <div
            v-for="task in tasks"
            :key="task.id"
            class="upload-float-panel__task"
            :class="{
              'is-success': task.status === 'success',
              'is-error': task.status === 'error',
              'is-canceled': task.status === 'canceled',
            }"
          >
            <div class="upload-float-panel__task-head">
              <p class="upload-float-panel__task-name" :title="task.name">{{ task.name }}</p>
              <span class="upload-float-panel__task-size">{{
                formatUploadBytes(task.total)
              }}</span>
              <button
                v-if="task.status === 'uploading'"
                type="button"
                class="upload-float-panel__task-cancel"
                @click="$emit('cancel-task', task.id)"
              >
                取消
              </button>
            </div>
            <div class="upload-float-panel__task-progress">
              <span
                class="upload-float-panel__task-progress-fill"
                :style="{ width: `${task.progress}%` }"
              />
            </div>
            <p class="upload-float-panel__task-meta">
              <template v-if="task.status === 'error'">
                <span class="upload-float-panel__task-meta-lead"
                  >{{ formatUploadBytes(task.total) }} ·
                </span>
                <span>{{ task.errorText || '上传失败' }}</span>
              </template>
              <span v-else-if="task.status === 'canceled'"
                >{{ formatUploadBytes(task.total) }} · 已取消</span
              >
              <span v-else-if="task.status === 'success'" class="upload-float-panel__task-status"
                >已完成</span
              >
              <span v-else
                >{{ formatUploadBytes(task.loaded) }} / {{ formatUploadBytes(task.total) }} ·
                {{ task.progress }}%</span
              >
            </p>
          </div>
        </div>
        <input
          ref="uploadPanelFileInputRef"
          class="upload-float-panel__file-input"
          type="file"
          multiple
          @change="onFileSelected"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { UploadTask } from './types'

defineProps<{
  siteFileDragActive: boolean
  visible: boolean
  collapsed: boolean
  dragging: boolean
  panelStyle: Record<string, string | number>
  tasks: UploadTask[]
  uploadInProgressCount: number
  reduceIcon: string
}>()

const emit = defineEmits<{
  'panel-drag-start': [event: PointerEvent]
  'bubble-click': [event: MouseEvent]
  'toggle-panel': []
  'open-picker': []
  'cancel-task': [taskId: string]
  'file-selected': [files: FileList | null]
}>()

const uploadPanelFileInputRef = ref<HTMLInputElement | null>(null)

function formatUploadBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB'] as const
  let v = bytes
  let u = 0
  while (v >= 1024 && u < units.length - 1) {
    v /= 1024
    u++
  }
  const d = u === 0 ? 0 : v >= 100 ? 0 : v >= 10 ? 1 : 2
  return `${v.toFixed(d)} ${units[u]}`
}

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  emit('file-selected', input.files)
}

defineExpose({
  openFilePicker() {
    uploadPanelFileInputRef.value?.click()
  },
})
</script>

<style scoped lang="scss">
.site-upload-drop-mask {
  position: fixed;
  inset: 0;
  z-index: 2190;
  background: rgba(43, 62, 117, 0.16);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.site-upload-drop-mask__inner {
  min-width: 210px;
  height: 84px;
  border-radius: 14px;
  border: 1px dashed rgba(76, 112, 255, 0.6);
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.18);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #2b3d76;
  font-size: 14px;
  font-weight: 600;

  img {
    width: 22px;
    height: 22px;
    display: block;
  }
}

.upload-float-panel {
  position: fixed;
  z-index: 2200;
  transform-origin: bottom right;
  border-radius: 14px;
  background: linear-gradient(152deg, rgba(255, 255, 255, 0.9) 0%, rgba(249, 249, 255, 0.93) 100%);
  border: 1px solid rgba(255, 255, 255, 0.65);
  box-shadow:
    0 16px 32px rgba(15, 20, 40, 0.18),
    0 2px 6px rgba(15, 20, 40, 0.12);
  backdrop-filter: blur(8px);
  overflow: hidden;
  user-select: none;
  transition:
    box-shadow 0.25s ease,
    transform 0.2s ease;

  &::before {
    content: '';
    position: absolute;
    width: 86px;
    height: 86px;
    top: -30px;
    right: -26px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(90, 130, 255, 0.24) 0%, rgba(90, 130, 255, 0) 72%);
    pointer-events: none;
  }

  &.is-dragging {
    transform: scale(1.015);
    box-shadow:
      0 22px 36px rgba(15, 20, 40, 0.24),
      0 4px 8px rgba(15, 20, 40, 0.16);
  }

  &.is-collapsed {
    border-radius: 999px;
    border-color: rgba(90, 130, 255, 0.42);
    background: linear-gradient(
      160deg,
      rgba(255, 255, 255, 0.95) 0%,
      rgba(236, 242, 255, 0.95) 100%
    );
    box-shadow:
      0 14px 30px rgba(76, 112, 255, 0.28),
      0 2px 8px rgba(15, 20, 40, 0.15);

    &::before {
      display: none;
    }
  }
}

.upload-float-panel__bubble {
  width: 100%;
  height: 100%;
  border: none;
  border-radius: 999px;
  background: transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: grab;
  padding: 0;
  transition:
    transform 0.18s ease,
    filter 0.18s ease;

  img {
    width: 20px;
    height: 20px;
    display: block;
    object-fit: contain;
  }

  &:hover {
    transform: scale(1.04);
    filter: brightness(1.05);
  }
}

.upload-float-panel.is-dragging .upload-float-panel__bubble {
  cursor: grabbing;
  transform: scale(1.03);
}

.upload-float-panel__toolbar {
  height: 32px;
  padding: 0 7px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(255, 255, 255, 0.6);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.76) 0%, rgba(244, 247, 255, 0.62) 100%);
}

.upload-float-panel__drag-handle,
.upload-float-panel__toggle-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.35);
  cursor: pointer;
  padding: 0;
  transition:
    background 0.2s ease,
    transform 0.2s ease;

  img {
    width: 14px;
    height: 14px;
    display: block;
    object-fit: contain;
    -webkit-user-drag: none;
  }

  &:hover {
    background: rgba(85, 120, 255, 0.13);
    transform: translateY(-1px);
  }
}

.upload-float-panel__drag-handle {
  cursor: grab;
}

.upload-float-panel.is-dragging .upload-float-panel__drag-handle {
  cursor: grabbing;
}

.upload-float-panel__body {
  height: calc(100% - 32px);
  padding: 8px 10px 10px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 7px;
  overflow-y: auto;
}

.upload-float-panel__hint {
  font-size: 11px;
  line-height: 1.2;
  letter-spacing: 0.02em;
  color: #65708f;
  text-align: center;
}

.upload-float-panel__upload-btn {
  width: 100%;
  height: 34px;
  min-height: 34px;
  border: 1px solid rgba(76, 112, 255, 0.34);
  border-radius: 10px;
  background: linear-gradient(160deg, #ffffff 0%, #eef2ff 100%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  user-select: none;
  -webkit-user-drag: none;
  font-size: 12.5px;
  font-weight: 600;
  color: #2c3d75;
  cursor: pointer;
  padding: 0;
  transition:
    transform 0.2s ease,
    box-shadow 0.25s ease,
    border-color 0.25s ease;

  img {
    width: 16px;
    height: 16px;
    filter: saturate(1.08);
    -webkit-user-drag: none;
  }

  &:hover {
    border-color: rgba(76, 112, 255, 0.5);
    box-shadow: 0 8px 15px rgba(76, 112, 255, 0.2);
    transform: translateY(-1px);
  }

  &:active {
    transform: translateY(0);
  }
}

.upload-float-panel__tasks {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.upload-float-panel__task {
  padding: 8px 10px;
  border-radius: 12px;
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.55) 0%, rgba(241, 244, 255, 0.9) 100%);
  border: 1px solid rgba(90, 110, 180, 0.2);
  box-shadow: 0 2px 8px rgba(28, 40, 90, 0.06);
}

.upload-float-panel__task-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.upload-float-panel__task-name {
  margin: 0;
  font-size: 11.5px;
  font-weight: 500;
  letter-spacing: 0.01em;
  color: #2a3358;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1 1 auto;
  min-width: 0;
}

.upload-float-panel__task-size {
  flex: 0 0 auto;
  font-size: 10px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: #6b7394;
  white-space: nowrap;
}

.upload-float-panel__task-cancel {
  flex: 0 0 auto;
  border: 1px solid rgba(208, 79, 79, 0.35);
  background: rgba(255, 255, 255, 0.85);
  color: #b24444;
  font-size: 10px;
  line-height: 1;
  border-radius: 999px;
  padding: 4px 9px;
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease;

  &:hover {
    background: #fff;
    border-color: rgba(208, 79, 79, 0.55);
  }
}

.upload-float-panel__task-progress {
  margin-top: 7px;
  height: 5px;
  border-radius: 999px;
  background: rgba(66, 87, 141, 0.12);
  overflow: hidden;
  box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.6);
}

.upload-float-panel__task-progress-fill {
  height: 100%;
  display: block;
  border-radius: inherit;
  background: linear-gradient(90deg, #5b7eff 0%, #94aeff 100%);
  box-shadow: 0 0 8px rgba(91, 126, 255, 0.45);
  transition: width 0.18s linear;
}

.upload-float-panel__task-meta {
  margin: 6px 0 0;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  color: #5c6488;
  line-height: 1.35;
}

.upload-float-panel__task-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 7px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  color: #1f7a55;
  background: rgba(49, 178, 127, 0.12);
  border: 1px solid rgba(49, 178, 127, 0.22);
}

.upload-float-panel__task.is-success .upload-float-panel__task-progress-fill {
  background: linear-gradient(90deg, #2dad7a 0%, #5ed4a4 100%);
  box-shadow: 0 0 6px rgba(45, 173, 122, 0.35);
}

.upload-float-panel__task.is-error .upload-float-panel__task-progress-fill {
  background: linear-gradient(90deg, #d85b5b 0%, #f08989 100%);
  box-shadow: 0 0 6px rgba(216, 91, 91, 0.35);
}

.upload-float-panel__task.is-error .upload-float-panel__task-meta {
  color: #ae3d3d;
}

.upload-float-panel__task.is-canceled .upload-float-panel__task-progress-fill {
  background: linear-gradient(90deg, #a3acbe 0%, #c3cad8 100%);
  box-shadow: none;
}

.upload-float-panel__task-meta-lead {
  color: #6b7394;
}

.upload-float-panel__file-input {
  display: none;
}

@media (max-width: 640px) {
  .upload-float-panel {
    box-shadow:
      0 12px 26px rgba(15, 20, 40, 0.2),
      0 2px 6px rgba(15, 20, 40, 0.12);
  }
}
</style>
