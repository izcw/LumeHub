<template>
  <div class="gallery-file-preview" :class="{ 'is-compact': compact }">
    <div class="gallery-file-preview__icon-wrap" :style="{ backgroundColor: theme.bg }">
      <img class="gallery-file-preview__icon" :src="iconUrl" alt="" draggable="false" />
    </div>
    <div class="gallery-file-preview__body">
      <p v-if="displayName" class="gallery-file-preview__name" :title="displayName">{{ displayName }}</p>
      <p class="gallery-file-preview__line">{{ kindLabel }} · {{ formatLabel }}</p>
      <p v-if="sizeText !== '—'" class="gallery-file-preview__line">{{ sizeText }}</p>
      <p v-if="dateText" class="gallery-file-preview__line">{{ dateText }}</p>
      <template v-if="showDetails">
        <p v-if="uploadedText" class="gallery-file-preview__line gallery-file-preview__line--muted">
          上传 {{ uploadedText }}
        </p>
        <p v-if="updatedText" class="gallery-file-preview__line gallery-file-preview__line--muted">
          更新 {{ updatedText }}
        </p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  galleryExtFromUrl,
  galleryMediaKindFromUrl,
  galleryMediaKindLabel,
  type GalleryMediaKind,
} from '@/utils/galleryMedia'
import { formatFileSize } from '@/utils/formatFileSize'

import iconCompress from '@/assets/icon/Compress.svg?url'
import iconVideo from '@/assets/icon/video.svg?url'
import iconAudio from '@/assets/icon/Audio.svg?url'
import iconFile from '@/assets/icon/File.svg?url'
import iconMarkdown from '@/assets/icon/markdown.svg?url'
import iconOther from '@/assets/icon/other.svg?url'

const props = withDefaults(
  defineProps<{
    src: string
    fileName?: string
    format?: string
    mediaKind?: string
    fileSize?: number
    date?: string
    uploadedAt?: string
    updatedAt?: string
    compact?: boolean
    showDetails?: boolean
  }>(),
  {
    fileName: '',
    format: '',
    mediaKind: '',
    date: '',
    uploadedAt: '',
    updatedAt: '',
    compact: false,
    showDetails: false,
  },
)

const FILE_PALETTE: Record<Exclude<GalleryMediaKind, 'image'>, { bg: string }> = {
  archive: { bg: '#ffd766' },
  video: { bg: '#dde8ff' },
  audio: { bg: '#ede9fe' },
  document: { bg: '#fff2e8' },
  other: { bg: '#eef1f5' },
}

const resolvedKind = computed<GalleryMediaKind>(() => {
  const k = props.mediaKind?.trim()
  if (k && k !== 'image') return k as GalleryMediaKind
  return galleryMediaKindFromUrl(props.src)
})

const theme = computed(() => {
  const k = resolvedKind.value
  if (k === 'image') return FILE_PALETTE.other
  return FILE_PALETTE[k]
})

const iconUrl = computed(() => {
  const k = resolvedKind.value
  const ext = galleryExtFromUrl(props.src)
  if (k === 'document' && ext === 'md') return iconMarkdown
  if (k === 'archive') return iconCompress
  if (k === 'video') return iconVideo
  if (k === 'audio') return iconAudio
  if (k === 'document') return iconFile
  return iconOther
})

const displayName = computed(() => {
  const name = props.fileName?.trim()
  if (name) return name
  const seg = (props.src.split('?')[0] ?? '').split(/[/\\]/).pop() ?? ''
  if (!seg) return ''
  try {
    return decodeURIComponent(seg)
  } catch {
    return seg
  }
})

const kindLabel = computed(() => galleryMediaKindLabel(resolvedKind.value))
const formatLabel = computed(
  () => (props.format || galleryExtFromUrl(props.src)).toUpperCase() || '—',
)
const sizeText = computed(() => formatFileSize(props.fileSize))
const dateText = computed(() => props.date?.trim() ?? '')

function formatDateTime(iso?: string): string {
  const raw = iso?.trim()
  if (!raw) return ''
  const t = Date.parse(raw)
  if (Number.isNaN(t)) return raw
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

const uploadedText = computed(() => formatDateTime(props.uploadedAt))
const updatedText = computed(() => formatDateTime(props.updatedAt))
</script>

<style scoped lang="scss">
.gallery-file-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  width: 100%;
  height: 100%;
  padding: 20px 16px;
  text-align: center;
}

.gallery-file-preview.is-compact {
  gap: 8px;
  padding: 12px 10px;
}

.gallery-file-preview__icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 88px;
  height: 88px;
  border-radius: 18px;
}

.gallery-file-preview.is-compact .gallery-file-preview__icon-wrap {
  width: min(38%, 72px);
  height: min(38%, 72px);
  border-radius: 14px;
}

.gallery-file-preview__icon {
  width: 52%;
  height: auto;
  max-height: 52%;
  object-fit: contain;
  display: block;
  pointer-events: none;
  -webkit-user-drag: none;
}

.gallery-file-preview__body {
  min-width: 0;
  width: 100%;
}

.gallery-file-preview__name {
  margin: 0 0 4px;
  font-size: 14px;
  font-weight: 600;
  color: #1d1d1f;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gallery-file-preview.is-compact .gallery-file-preview__name {
  font-size: 11px;
  margin-bottom: 2px;
}

.gallery-file-preview__line {
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
  color: rgba(29, 29, 31, 0.72);
}

.gallery-file-preview.is-compact .gallery-file-preview__line {
  font-size: 10px;
  line-height: 1.4;
}

.gallery-file-preview__line--muted {
  font-size: 11px;
  color: rgba(29, 29, 31, 0.55);
}

.gallery-file-preview.is-compact .gallery-file-preview__line--muted {
  display: none;
}
</style>
