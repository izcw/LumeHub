<template>
  <div class="markdown-editor" :class="{ 'is-editing': editing }">
    <div v-if="editing" ref="editorHost" class="markdown-editor__editor" />
    <div v-else ref="viewerHost" class="markdown-editor__viewer" />
  </div>
</template>

<script setup lang="ts">
import { Editor } from '@toast-ui/editor'
import Viewer from '@toast-ui/editor/viewer'
import '@toast-ui/editor/dist/toastui-editor.css'
import '@toast-ui/editor/dist/toastui-editor-viewer.css'
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: string
    editing?: boolean
    height?: string
  }>(),
  { editing: false, height: '100%' },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const editorHost = ref<HTMLElement | null>(null)
const viewerHost = ref<HTMLElement | null>(null)
let editor: Editor | null = null
let viewer: Viewer | null = null

function destroyEditor() {
  editor?.destroy()
  editor = null
}

function destroyViewer() {
  viewer?.destroy()
  viewer = null
}

function createEditor() {
  if (!editorHost.value) return
  destroyEditor()
  editor = new Editor({
    el: editorHost.value,
    height: props.height,
    initialEditType: 'wysiwyg',
    previewStyle: 'vertical',
    initialValue: props.modelValue,
    usageStatistics: false,
    hideModeSwitch: true,
    toolbarItems: [
      ['heading', 'bold', 'italic', 'strike'],
      ['hr', 'quote'],
      ['ul', 'ol', 'task', 'indent', 'outdent'],
      ['table', 'link', 'image'],
      ['code', 'codeblock'],
    ],
  })
  editor.on('change', () => {
    emit('update:modelValue', editor?.getMarkdown() ?? '')
  })
}

function createViewer() {
  if (!viewerHost.value) return
  destroyViewer()
  viewer = new Viewer({
    el: viewerHost.value,
    initialValue: props.modelValue,
    usageStatistics: false,
  })
}

async function syncInstance(editing: boolean) {
  await nextTick()
  if (editing) {
    destroyViewer()
    createEditor()
  } else {
    destroyEditor()
    createViewer()
  }
}

watch(() => props.editing, (editing) => void syncInstance(editing), { immediate: true })
watch(() => props.modelValue, (value) => {
  if (props.editing && editor && editor.getMarkdown() !== value) {
    editor.setMarkdown(value, false)
  }
  if (!props.editing && viewer) {
    viewer.setMarkdown(value)
  }
})

onBeforeUnmount(() => {
  destroyEditor()
  destroyViewer()
})
</script>

<style scoped lang="scss">
.markdown-editor {
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: #fff;
}

.markdown-editor__editor,
.markdown-editor__viewer {
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: auto;
  padding: 32px 42px 56px;
  background: #fff;
}

:deep(.toastui-editor-defaultUI) {
  height: 100%;
  border: 0;
}

:deep(.toastui-editor-main) {
  min-height: 0;
}

:deep(.toastui-editor-ww-container) {
  background: #fff;
}

:deep(.toastui-editor-ww-container .toastui-editor-contents) {
  max-width: 920px;
  margin: 0 auto;
  padding: 30px 42px 56px;
}

:deep(.toastui-editor-contents) {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', sans-serif;
  max-width: 920px;
  margin: 0 auto;
  color: #252525;
  font-size: 15px;
  line-height: 1.8;
}

:deep(.toastui-editor-contents h1),
:deep(.toastui-editor-contents h2),
:deep(.toastui-editor-contents h3),
:deep(.toastui-editor-contents h4) {
  color: #171717;
  letter-spacing: -0.02em;
  line-height: 1.3;
}

:deep(.toastui-editor-contents h1) {
  padding-bottom: 0.35em;
  border-bottom: 1px solid #e5e7eb;
  font-size: 2em;
}

:deep(.toastui-editor-contents h2) {
  padding-bottom: 0.25em;
  border-bottom: 1px solid #ececec;
  font-size: 1.55em;
}

:deep(.toastui-editor-contents blockquote) {
  margin: 1.2em 0;
  padding: 0.7em 1em;
  border-left: 4px solid #c7cbd1;
  background: #f7f8fa;
  color: #6b7280;
}

:deep(.toastui-editor-contents table) {
  display: table;
  width: 100%;
  margin: 1.4em 0;
  border-color: #dfe3e8;
  border-collapse: collapse;
}

:deep(.toastui-editor-contents th) {
  background: #f4f5f7;
  font-weight: 650;
}

:deep(.toastui-editor-contents th),
:deep(.toastui-editor-contents td) {
  padding: 0.65em 0.8em;
  border: 1px solid #dfe3e8;
}

:deep(.toastui-editor-contents code) {
  padding: 0.15em 0.35em;
  border-radius: 4px;
  background: #f1f3f5;
  color: #b42318;
  font-size: 0.9em;
}

:deep(.toastui-editor-contents pre) {
  margin: 1.2em 0;
  border-radius: 8px;
  background: #1f2937;
}

:deep(.toastui-editor-contents pre code) {
  padding: 0;
  background: transparent;
  color: #f3f4f6;
}

:deep(.toastui-editor-contents a) {
  color: #2563eb;
}

:deep(.toastui-editor-contents ul),
:deep(.toastui-editor-contents ol) {
  padding-left: 1.8em;
}

:deep(.toastui-editor-contents img) {
  max-width: 100%;
}
</style>
