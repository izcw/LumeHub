<template>
  <div
    class="card-toolbar"
    :class="[`card-toolbar--${layout}`, `card-toolbar--${surface}`]"
    @click.stop
    @mouseenter.stop
  >
    <div v-if="hasMetaStack" class="card-meta-stack">
      <span v-if="metaSizeText" class="card-meta-line">{{ metaSizeText }}</span>
      <span v-if="metaFormatText" class="card-meta-line">{{ metaFormatText }}</span>
    </div>
    <div class="card-toolbar-actions">
      <Tooltip v-if="showRestore" text="恢复">
        <div
          class="card-action"
          role="button"
          tabindex="0"
          aria-label="恢复"
          @click.stop="emit('restore')"
        >
          <img :src="iconRecover" alt="" draggable="false" />
        </div>
      </Tooltip>
      <Tooltip v-if="showAdminActions" text="删除">
        <Popconfirm
          v-model:open="deleteConfirmOpen"
          :title="deleteConfirmTitle"
          confirm-text="删除"
          @confirm="emit('delete')"
        >
          <template #trigger>
            <div class="card-action" role="button" tabindex="0" aria-label="删除">
              <img :src="iconDelete" alt="" draggable="false" />
            </div>
          </template>
        </Popconfirm>
      </Tooltip>
      <Tooltip v-if="!deleteOnly && showAdminActions" text="编辑">
        <div
          class="card-action"
          role="button"
          tabindex="0"
          aria-label="编辑"
          @click="emit('edit')"
        >
          <img :src="iconEdit" alt="" draggable="false" />
        </div>
      </Tooltip>
      <Tooltip v-if="!deleteOnly && viewHref" text="查看">
        <a
          class="card-action"
          :href="viewHref"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="查看"
          @click.stop
        >
          <img :src="iconEye" alt="" draggable="false" />
        </a>
      </Tooltip>
      <Tooltip v-if="!deleteOnly" text="下载">
        <div
          class="card-action"
          role="button"
          tabindex="0"
          aria-label="下载"
          @click="emit('download')"
        >
          <img :src="iconDownload" alt="" draggable="false" />
        </div>
      </Tooltip>
      <Tooltip v-if="!deleteOnly" text="复制链接">
        <div
          class="card-action"
          role="button"
          tabindex="0"
          aria-label="复制链接"
          @click="emit('copy-link')"
        >
          <img :src="iconCopy" alt="" draggable="false" />
        </div>
      </Tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, provide, ref, toRef, watch } from 'vue'
import { galleryExtFromUrl } from '@/utils/galleryMedia'
import { formatFileSize } from '@/utils/formatFileSize'
import { FLOATING_UI_Z_INDEX_KEY } from '@/components/viewers/shared/viewerLayers'
import iconDelete from '@/assets/icon/delete.svg?url'
import iconRecover from '@/assets/icon/recover.svg?url'
import iconEdit from '@/assets/icon/edit.svg?url'
import iconEye from '@/assets/icon/eye.svg?url'
import iconDownload from '@/assets/icon/download.svg?url'
import iconCopy from '@/assets/icon/copy.svg?url'
import Popconfirm from '@/components/shared-ui/Popconfirm.vue'
import Tooltip from '@/components/shared-ui/Tooltip.vue'

const props = withDefaults(
  defineProps<{
    src: string
    viewerFullSrc?: string
    fileSize?: number
    format?: string
    showAdminActions?: boolean
    deleteOnly?: boolean
    showRestore?: boolean
    deleteConfirmTitle?: string
    viewHref?: string
    layout?: 'card' | 'inline'
    surface?: 'dark' | 'light'
    /** 查看器内使用时，提升 tooltip / popconfirm 层级 */
    floatingZIndex?: number
  }>(),
  {
    layout: 'card',
    surface: 'dark',
    deleteConfirmTitle: '确定删除该文件吗？',
  },
)

const emit = defineEmits<{
  restore: []
  delete: []
  edit: []
  download: []
  'copy-link': []
  'pin-change': [pinned: boolean]
}>()

provide(FLOATING_UI_Z_INDEX_KEY, toRef(props, 'floatingZIndex'))

const deleteConfirmOpen = ref(false)

watch(deleteConfirmOpen, (open) => {
  emit('pin-change', open)
})

const deleteOnly = computed(() => props.deleteOnly === true)
const showRestore = computed(() => props.showRestore === true)
const showAdminActions = computed(() => props.showAdminActions === true)
const viewHref = computed(() => (props.viewHref ?? '').trim())

const metaFormatText = computed(
  () => (props.format || galleryExtFromUrl(props.viewerFullSrc || props.src)).toUpperCase(),
)
const metaSizeText = computed(() => {
  const size = formatFileSize(props.fileSize)
  return size === '—' ? '' : size
})
const hasMetaStack = computed(() => Boolean(metaSizeText.value || metaFormatText.value))
</script>

<style scoped lang="scss">
@mixin card-chip-surface-dark {
  color: rgba(255, 255, 255, 0.92);
  background: rgba(0, 0, 0, 0.48);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

@mixin card-chip-surface-light {
  color: rgba(28, 32, 40, 0.88);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.card-toolbar {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 10px;
  max-width: calc(100% - 8px);
  padding: 5px 10px 5px 12px;
  border-radius: 999px;
  box-sizing: border-box;
  z-index: 2;

  &--card {
    position: absolute;
    left: 50%;
    bottom: 0;
    transform: translateX(-50%);
  }

  &--inline {
    position: static;
    transform: none;
    max-width: min(100%, calc(100vw - 120px));
  }

  &--dark {
    @include card-chip-surface-dark;
  }

  &--light {
    @include card-chip-surface-light;

    .card-meta-stack {
      border-right-color: rgba(0, 0, 0, 0.08);
    }

    .card-action img {
      filter: none;
      opacity: 0.82;
    }

    .card-action:hover {
      background: rgba(0, 0, 0, 0.06);
    }
  }
}

.card-meta-stack {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  padding-right: 8px;
  margin-right: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.14);
  flex-shrink: 0;
  text-align: center;
}

.card-meta-line {
  display: block;
  width: 100%;
  font-size: 9px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: 0.2px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: nowrap;
  text-align: center;
}

.card-toolbar-actions {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;

  :deep(.ui-popconfirm) {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  :deep(.ui-tooltip) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
}

.card-action {
  width: 26px;
  height: 26px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.92);
  flex: 0 0 auto;
  transition:
    background 0.18s ease,
    transform 0.18s ease;

  img {
    width: 13px;
    height: 13px;
    display: block;
    filter: brightness(0) invert(1);
    opacity: 0.92;
  }

  &:hover {
    background: rgba(255, 255, 255, 0.16);
    transform: translateY(-1px);
  }
}

a.card-action {
  text-decoration: none;
}
</style>
