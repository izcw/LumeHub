import { computed, nextTick, ref, watch } from 'vue'
import axios from 'axios'
import { uploadCategoryFile } from '@/api/gallery'
import { buildLivePhotoUploadBatches } from '@/utils/livePhoto'
import type { UploadTask } from '../types'

const UPLOAD_PANEL_WIDTH = 220
const UPLOAD_PANEL_HEIGHT = 140
const UPLOAD_PANEL_TASK_WIDTH = 280
const UPLOAD_PANEL_TASK_HEIGHT = 320
const UPLOAD_PANEL_COLLAPSED_SIZE = 44
const UPLOAD_PANEL_GAP = 20
const MAX_UPLOAD_TASK_COUNT = 8

export function useGalleryUploadPanel(options: {
  folderKey: () => string
  enabled: () => boolean
  onReloadCategory: () => Promise<void>
}) {
  const uploadPanelCollapsed = ref(true)
  const uploadPanelDragging = ref(false)
  const uploadPanelFileInputRef = ref<HTMLInputElement | null>(null)
  const uploadTasks = ref<UploadTask[]>([])
  const uploadPanelEdge = ref({ right: UPLOAD_PANEL_GAP, bottom: UPLOAD_PANEL_GAP })
  const uploadPanelPositionInitialized = ref(false)
  const uploadPanelDragOffset = ref({ x: 0, y: 0 })
  const uploadPanelDragStartPoint = ref({ x: 0, y: 0 })
  const uploadPanelDragMoved = ref(false)
  const siteFileDragActive = ref(false)
  let uploadTaskCounter = 0
  let siteFileDragDepth = 0
  let suppressUploadBubbleClickUntil = 0
  let uploadPanelDragPointerId: number | null = null
  const uploadTaskAbortControllers = new Map<string, AbortController>()
  let stopEnabledWatch: (() => void) | null = null

  const uploadInProgressCount = computed(
    () => uploadTasks.value.filter((task) => task.status === 'uploading').length,
  )
  const uploadPanelHasTasks = computed(() => uploadTasks.value.length > 0)

  function uploadPanelWidthFor(collapsed: boolean, hasTasks: boolean): number {
    if (collapsed) return UPLOAD_PANEL_COLLAPSED_SIZE
    return hasTasks ? UPLOAD_PANEL_TASK_WIDTH : UPLOAD_PANEL_WIDTH
  }

  function uploadPanelHeightFor(collapsed: boolean, hasTasks: boolean): number {
    if (collapsed) return UPLOAD_PANEL_COLLAPSED_SIZE
    return hasTasks ? UPLOAD_PANEL_TASK_HEIGHT : UPLOAD_PANEL_HEIGHT
  }

  const uploadPanelStyle = computed(() => ({
    width: `${uploadPanelCurrentWidth()}px`,
    height: `${uploadPanelCurrentHeight()}px`,
    right: `${uploadPanelEdge.value.right}px`,
    bottom: `${uploadPanelEdge.value.bottom}px`,
    left: 'auto',
    top: 'auto',
  }))

  function uploadPanelCurrentWidth() {
    return uploadPanelWidthFor(uploadPanelCollapsed.value, uploadPanelHasTasks.value)
  }

  function uploadPanelCurrentHeight() {
    return uploadPanelHeightFor(uploadPanelCollapsed.value, uploadPanelHasTasks.value)
  }

  function uploadPanelViewportSize() {
    const vv = window.visualViewport
    if (vv) {
      return { w: vv.width, h: vv.height }
    }
    return { w: window.innerWidth, h: window.innerHeight }
  }

  function clampUploadPanelEdge(rawRight: number, rawBottom: number) {
    const { w: vw, h: vh } = uploadPanelViewportSize()
    const width = uploadPanelCurrentWidth()
    const height = uploadPanelCurrentHeight()
    const minRight = UPLOAD_PANEL_GAP
    const minBottom = UPLOAD_PANEL_GAP
    const maxRight = Math.max(minRight, vw - width - UPLOAD_PANEL_GAP)
    const maxBottom = Math.max(minBottom, vh - height - UPLOAD_PANEL_GAP)
    return {
      right: Math.min(Math.max(minRight, rawRight), maxRight),
      bottom: Math.min(Math.max(minBottom, rawBottom), maxBottom),
    }
  }

  function resetUploadPanelToBottomRight() {
    uploadPanelEdge.value = clampUploadPanelEdge(UPLOAD_PANEL_GAP, UPLOAD_PANEL_GAP)
    uploadPanelPositionInitialized.value = true
  }

  function syncUploadPanelInViewport() {
    if (!uploadPanelPositionInitialized.value) {
      resetUploadPanelToBottomRight()
      return
    }
    uploadPanelEdge.value = clampUploadPanelEdge(
      uploadPanelEdge.value.right,
      uploadPanelEdge.value.bottom,
    )
  }

  function openUploadFilePicker() {
    uploadPanelFileInputRef.value?.click()
  }

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

  function normalizeUploadProgress(loaded: number, total: number): number {
    if (total <= 0) return 0
    const ratio = Math.round((loaded / total) * 100)
    if (!Number.isFinite(ratio)) return 0
    return Math.min(100, Math.max(0, ratio))
  }

  function updateUploadTask(id: string, patch: Partial<UploadTask>) {
    uploadTasks.value = uploadTasks.value.map((task) =>
      task.id === id ? { ...task, ...patch } : task,
    )
  }

  function trimUploadTasks() {
    const uploading = uploadTasks.value.filter((task) => task.status === 'uploading')
    const finished = uploadTasks.value
      .filter((task) => task.status !== 'uploading')
      .slice(0, Math.max(0, MAX_UPLOAD_TASK_COUNT - uploading.length))
    uploadTasks.value = [...uploading, ...finished]
  }

  function isCanceledUploadError(err: unknown): boolean {
    if (axios.isAxiosError(err) && err.code === 'ERR_CANCELED') return true
    if (err instanceof DOMException && err.name === 'AbortError') return true
    return false
  }

  function uploadErrorText(err: unknown): string {
    if (isCanceledUploadError(err)) return '已取消'
    if (axios.isAxiosError(err)) {
      const status = err.response?.status
      const body =
        typeof err.response?.data === 'string' ? err.response.data.toLowerCase() : ''
      if (status === 401) return '未登录或登录已过期'
      if (status === 404) return '目标画廊不存在'
      if (status === 507 || body.includes('storage quota')) return '存储空间不足'
      if (status && status >= 500) return '服务器上传失败'
    }
    return '上传失败'
  }

  function setUploadPanelCollapsedWithBottomRightAnchor(nextCollapsed: boolean) {
    if (uploadPanelCollapsed.value === nextCollapsed) return
    if (!uploadPanelPositionInitialized.value) {
      uploadPanelCollapsed.value = nextCollapsed
      syncUploadPanelInViewport()
      return
    }
    uploadPanelCollapsed.value = nextCollapsed
    uploadPanelEdge.value = clampUploadPanelEdge(
      uploadPanelEdge.value.right,
      uploadPanelEdge.value.bottom,
    )
  }

  async function uploadFilesToCurrentGallery(filesInput: readonly File[]) {
    const batches = buildLivePhotoUploadBatches(filesInput.filter((file) => file.size > 0))
    if (batches.length === 0) return
    if (!options.enabled()) return
    setUploadPanelCollapsedWithBottomRightAnchor(false)
    const targetFolderKey = options.folderKey()
    const tasks = batches.map((batch) => {
      uploadTaskCounter += 1
      const total = batch.main.size + (batch.motion?.size ?? 0)
      const task: UploadTask = {
        id: `upload-${Date.now()}-${uploadTaskCounter}`,
        name: batch.motion ? `${batch.label} + 实况` : batch.label,
        loaded: 0,
        total: Math.max(1, total),
        progress: 0,
        status: 'uploading',
        errorText: '',
      }
      return { task, batch }
    })
    uploadTasks.value = [...tasks.map((entry) => entry.task), ...uploadTasks.value].slice(
      0,
      MAX_UPLOAD_TASK_COUNT,
    )
    await Promise.all(
      tasks.map(async ({ task, batch }) => {
        const controller = new AbortController()
        uploadTaskAbortControllers.set(task.id, controller)
        try {
          await uploadCategoryFile(
            targetFolderKey,
            batch.main,
            ({ loaded, total }) => {
              updateUploadTask(task.id, {
                loaded,
                total,
                progress: normalizeUploadProgress(loaded, total),
                status: 'uploading',
                errorText: '',
              })
            },
            controller.signal,
            batch.motion,
          )
          updateUploadTask(task.id, {
            loaded: task.total,
            total: task.total,
            progress: 100,
            status: 'success',
            errorText: '',
          })
        } catch (error) {
          if (isCanceledUploadError(error)) {
            updateUploadTask(task.id, {
              status: 'canceled',
              errorText: '',
            })
            return
          }
          updateUploadTask(task.id, {
            status: 'error',
            errorText: uploadErrorText(error),
          })
        } finally {
          uploadTaskAbortControllers.delete(task.id)
        }
      }),
    )
    trimUploadTasks()
    if (targetFolderKey === options.folderKey()) {
      await options.onReloadCategory()
    }
  }

  function cancelUploadTask(taskId: string) {
    const hit = uploadTaskAbortControllers.get(taskId)
    if (!hit) return
    hit.abort()
  }

  function onUploadFileSelected(event: Event) {
    const target = event.target
    if (!(target instanceof HTMLInputElement)) return
    const files = target.files ? Array.from(target.files) : []
    void uploadFilesToCurrentGallery(files)
    target.value = ''
  }

  function onUploadFilesPicked(files: FileList | null) {
    if (!files) return
    void uploadFilesToCurrentGallery(Array.from(files))
  }

  function dragEventHasFiles(event: DragEvent): boolean {
    const types = event.dataTransfer?.types
    if (!types) return false
    for (const t of types) {
      if (t === 'Files') return true
    }
    return false
  }

  function clearSiteFileDragState() {
    siteFileDragDepth = 0
    siteFileDragActive.value = false
  }

  function onWindowDragEnter(event: DragEvent) {
    if (!dragEventHasFiles(event)) return
    event.preventDefault()
    siteFileDragDepth += 1
    if (!options.enabled()) return
    siteFileDragActive.value = true
  }

  function onWindowDragOver(event: DragEvent) {
    if (!dragEventHasFiles(event)) return
    event.preventDefault()
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
    if (!options.enabled()) return
    siteFileDragActive.value = true
  }

  function onWindowDragLeave(event: DragEvent) {
    if (!dragEventHasFiles(event)) return
    event.preventDefault()
    siteFileDragDepth = Math.max(0, siteFileDragDepth - 1)
    if (siteFileDragDepth === 0) {
      siteFileDragActive.value = false
    }
  }

  function onWindowDrop(event: DragEvent) {
    if (!dragEventHasFiles(event)) return
    event.preventDefault()
    const files = event.dataTransfer?.files ? Array.from(event.dataTransfer.files) : []
    clearSiteFileDragState()
    if (!options.enabled() || files.length === 0) return
    void uploadFilesToCurrentGallery(files)
  }

  function toggleUploadPanel() {
    setUploadPanelCollapsedWithBottomRightAnchor(!uploadPanelCollapsed.value)
  }

  function onUploadPanelDragMove(event: PointerEvent) {
    if (!uploadPanelDragging.value) return
    if (uploadPanelDragPointerId !== null && event.pointerId !== uploadPanelDragPointerId) return
    if (
      !uploadPanelDragMoved.value &&
      (Math.abs(event.clientX - uploadPanelDragStartPoint.value.x) > 3 ||
        Math.abs(event.clientY - uploadPanelDragStartPoint.value.y) > 3)
    ) {
      uploadPanelDragMoved.value = true
    }
    const { w: vw, h: vh } = uploadPanelViewportSize()
    const width = uploadPanelCurrentWidth()
    const height = uploadPanelCurrentHeight()
    const left = event.clientX - uploadPanelDragOffset.value.x
    const top = event.clientY - uploadPanelDragOffset.value.y
    const right = vw - left - width
    const bottom = vh - top - height
    uploadPanelEdge.value = clampUploadPanelEdge(right, bottom)
  }

  function endUploadPanelDrag() {
    if (uploadPanelDragMoved.value) {
      suppressUploadBubbleClickUntil = Date.now() + 180
    }
    uploadPanelDragging.value = false
    uploadPanelDragMoved.value = false
    uploadPanelDragPointerId = null
    window.removeEventListener('pointermove', onUploadPanelDragMove)
    window.removeEventListener('pointerup', endUploadPanelDrag)
    window.removeEventListener('pointercancel', endUploadPanelDrag)
  }

  function onUploadPanelDragStart(event: PointerEvent) {
    if (!options.enabled()) return
    const target = event.currentTarget
    if (!(target instanceof HTMLElement)) return
    const panel = target.closest('.upload-float-panel')
    if (!(panel instanceof HTMLElement)) return
    event.preventDefault()
    const rect = panel.getBoundingClientRect()
    uploadPanelDragStartPoint.value = { x: event.clientX, y: event.clientY }
    uploadPanelDragOffset.value = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    }
    uploadPanelDragMoved.value = false
    uploadPanelDragging.value = true
    uploadPanelDragPointerId = event.pointerId
    window.addEventListener('pointermove', onUploadPanelDragMove)
    window.addEventListener('pointerup', endUploadPanelDrag)
    window.addEventListener('pointercancel', endUploadPanelDrag)
  }

  function onUploadBubbleClick() {
    if (Date.now() < suppressUploadBubbleClickUntil) return
    setUploadPanelCollapsedWithBottomRightAnchor(false)
  }

  function setup() {
    stopEnabledWatch = watch(
      () => options.enabled(),
      (visible) => {
        if (!visible) {
          endUploadPanelDrag()
          clearSiteFileDragState()
          return
        }
        nextTick(() => {
          syncUploadPanelInViewport()
        })
      },
    )

    window.addEventListener('resize', syncUploadPanelInViewport)
    window.visualViewport?.addEventListener('resize', syncUploadPanelInViewport)
    window.visualViewport?.addEventListener('scroll', syncUploadPanelInViewport)
    window.addEventListener('dragenter', onWindowDragEnter)
    window.addEventListener('dragover', onWindowDragOver)
    window.addEventListener('dragleave', onWindowDragLeave)
    window.addEventListener('drop', onWindowDrop)

    if (options.enabled()) {
      resetUploadPanelToBottomRight()
    }
  }

  function teardown() {
    stopEnabledWatch?.()
    stopEnabledWatch = null

    window.removeEventListener('resize', syncUploadPanelInViewport)
    window.visualViewport?.removeEventListener('resize', syncUploadPanelInViewport)
    window.visualViewport?.removeEventListener('scroll', syncUploadPanelInViewport)
    window.removeEventListener('dragenter', onWindowDragEnter)
    window.removeEventListener('dragover', onWindowDragOver)
    window.removeEventListener('dragleave', onWindowDragLeave)
    window.removeEventListener('drop', onWindowDrop)

    endUploadPanelDrag()
    for (const ctrl of uploadTaskAbortControllers.values()) ctrl.abort()
    uploadTaskAbortControllers.clear()
    clearSiteFileDragState()
  }

  return {
    siteFileDragActive,
    uploadPanelCollapsed,
    uploadPanelDragging,
    uploadPanelStyle,
    uploadTasks,
    uploadInProgressCount,
    formatUploadBytes,
    openUploadFilePicker,
    cancelUploadTask,
    onUploadFileSelected,
    onUploadFilesPicked,
    onUploadPanelDragStart,
    onUploadBubbleClick,
    toggleUploadPanel,
    uploadPanelFileInputRef,
    setup,
    teardown,
  }
}
