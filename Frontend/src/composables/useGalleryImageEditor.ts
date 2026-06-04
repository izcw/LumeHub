import { computed, ref } from 'vue'

export type CropRect = { x: number; y: number; w: number; h: number }

export type WatermarkPosition =
  | 'bottom-right'
  | 'bottom-left'
  | 'top-right'
  | 'top-left'
  | 'center'
  | 'custom'
  | 'tile'

export type SizePreset = {
  id: string
  label: string
  width: number
  height: number
  desc: string
  custom?: boolean
}

export const IMAGE_PRESETS: SizePreset[] = [
  { id: '1inch', label: '一寸证件照', width: 295, height: 413, desc: '25×35mm' },
  { id: 'small1inch', label: '小一寸', width: 260, height: 378, desc: '22×32mm' },
  { id: '2inch', label: '二寸证件照', width: 413, height: 579, desc: '35×49mm' },
  { id: 'passport', label: '护照照片', width: 390, height: 567, desc: '33×48mm' },
  { id: '4x6', label: '四寸横版', width: 1200, height: 900, desc: '102×76mm' },
  { id: '3x4', label: '3:4 竖版', width: 900, height: 1200, desc: '3:4' },
  { id: '1080p', label: '1080P', width: 1920, height: 1080, desc: '16:9' },
  { id: 'square', label: '正方形', width: 1080, height: 1080, desc: '1:1' },
]

export type WatermarkPreset = {
  id: string
  label: string
  text: string
  opacity?: number
  rotation?: number
  position?: WatermarkPosition
  fontSize?: number
  /** 紧凑度百分比，100 为标准，越高越紧凑 */
  compactness?: number
  custom?: boolean
  customX?: number
  customY?: number
}

export const WATERMARK_PRESETS: WatermarkPreset[] = [
  { id: 'copyright', label: '版权', text: '© LumeHub', opacity: 45, position: 'bottom-right', fontSize: 22 },
  { id: 'sample', label: '样张', text: 'SAMPLE', opacity: 35, position: 'tile', rotation: -30, fontSize: 28 },
  { id: 'draft', label: '草稿', text: 'DRAFT', opacity: 50, position: 'center', fontSize: 48 },
  { id: 'confidential', label: '机密', text: 'CONFIDENTIAL', opacity: 40, position: 'tile', rotation: -24, fontSize: 24 },
]

export type ExportEncoding = 'png' | 'jpeg'

export type ExportQualityPreset = {
  id: string
  label: string
  percent: number
  /** 100% 时区分无损 PNG 与原图 JPEG；其余预设默认 JPEG */
  encoding?: ExportEncoding
  desc?: string
  custom?: boolean
}

/** 原图 JPEG（100%）所用质量，与历史默认一致 */
export const DEFAULT_JPEG_QUALITY = 0.92

export const EXPORT_QUALITY_PRESETS: ExportQualityPreset[] = [
  { id: 'lossless', label: '无损', percent: 100, encoding: 'png', desc: 'PNG' },
  { id: 'original', label: '原图', percent: 100, encoding: 'jpeg', desc: 'JPEG' },
  { id: 'high', label: '高质量', percent: 90 },
  { id: 'balanced', label: '均衡', percent: 75 },
  { id: 'web', label: '网页分享', percent: 60 },
  { id: 'compact', label: '体积优先', percent: 45 },
]

export function usePngLosslessExport(encoding: ExportEncoding): boolean {
  return encoding === 'png'
}

export function exportMimeFromEncoding(encoding: ExportEncoding): 'image/png' | 'image/jpeg' {
  return usePngLosslessExport(encoding) ? 'image/png' : 'image/jpeg'
}

export function exportQualitySummary(percent: number, encoding: ExportEncoding): string {
  if (Math.round(percent) >= 100 && encoding === 'png') return '100% · 无损 · PNG'
  if (Math.round(percent) >= 100 && encoding === 'jpeg') return '100% · 原图 · JPEG'
  return `${Math.round(percent)}% · JPEG`
}

export function jpegQualityFromPercent(percent: number): number {
  const p = Math.max(10, Math.min(100, Math.round(percent)))
  if (p >= 100) return DEFAULT_JPEG_QUALITY
  return Math.max(0.1, Math.min(DEFAULT_JPEG_QUALITY, p / 100))
}

export function resolveEditedExportFilename(filenameHint: string, encoding: ExportEncoding): string {
  const base = filenameHint.trim() || 'edited'
  const lower = base.toLowerCase()
  if (usePngLosslessExport(encoding)) {
    if (lower.endsWith('.png')) return base
    return `${base.replace(/\.[^.]+$/, '')}.png`
  }
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return base
  return `${base.replace(/\.[^.]+$/, '')}.jpg`
}

export function isExportPresetActive(
  preset: ExportQualityPreset,
  percent: number,
  encoding: ExportEncoding,
): boolean {
  if (Math.round(percent) !== Math.round(preset.percent)) return false
  if (preset.encoding) return encoding === preset.encoding
  return encoding === 'jpeg'
}

export function useGalleryImageEditor() {
  const sourceImg = ref<HTMLImageElement | null>(null)
  const cropReady = ref(false)
  const crop = ref<CropRect>({ x: 0, y: 0, w: 1, h: 1 })
  const baseDisplay = ref({ w: 0, h: 0, offsetX: 0, offsetY: 0 })
  const lockAspect = ref(true)
  const outWidthText = ref('')
  const outHeightText = ref('')
  const flipH = ref(false)
  const flipV = ref(false)
  const rotationQuarter = ref(0)
  const watermarkText = ref('')
  const watermarkOpacity = ref(55)
  const watermarkRotation = ref(-24)
  const watermarkPosition = ref<WatermarkPosition>('bottom-right')
  const watermarkCustomX = ref(50)
  const watermarkCustomY = ref(50)
  const watermarkFontSize = ref(0)
  /** 100 = 标准间距；50 更松散，150 更紧凑 */
  const watermarkCompactness = ref(100)
  const exportQualityPercent = ref(100)
  const exportEncoding = ref<ExportEncoding>('jpeg')
  const zoom = ref(1)
  const panX = ref(0)
  const panY = ref(0)
  const panning = ref(false)
  const layerInnerTransform = ref('')
  const transformAnimating = ref(false)

  const LAYER_TRANSFORM_MS = 280

  type CropDragMode = 'move' | 'nw' | 'ne' | 'sw' | 'se' | 'n' | 's' | 'e' | 'w'

  let cropDrag:
    | {
        mode: CropDragMode
        startX: number
        startY: number
        base: CropRect
      }
    | null = null

  let panDrag: { startX: number; startY: number; basePanX: number; basePanY: number } | null = null

  const stagePointers = new Map<number, { x: number; y: number }>()
  let pinchSession: {
    startDist: number
    startZoom: number
    anchorX: number
    anchorY: number
  } | null = null

  let pendingCropTouch: {
    mode: CropDragMode
    startX: number
    startY: number
    base: CropRect
    pointerId: number
  } | null = null

  const CROP_TOUCH_MOVE_THRESHOLD = 8
  const PAN_SNAP_THRESHOLD_RELEASE = 40
  let sizeCropSyncLock = false
  let pairingOutputLock = false
  /** 标记即将由联动写入的字段，避免 watch 再次反算 */
  let autoFillingOutput: 'w' | 'h' | null = null

  type EditorSnapshot = {
    crop: CropRect
    lockAspect: boolean
    outWidthText: string
    outHeightText: string
    flipH: boolean
    flipV: boolean
    rotationQuarter: number
    watermarkText: string
    watermarkOpacity: number
    watermarkRotation: number
    watermarkPosition: WatermarkPosition
    watermarkCustomX: number
    watermarkCustomY: number
    watermarkFontSize: number
    watermarkCompactness: number
  }

  const EDIT_HISTORY_LIMIT = 40
  const historyPast = ref<EditorSnapshot[]>([])
  const historyFuture = ref<EditorSnapshot[]>([])
  let applyingHistory = false

  function captureSnapshot(): EditorSnapshot {
    return {
      crop: { ...crop.value },
      lockAspect: lockAspect.value,
      outWidthText: outWidthText.value,
      outHeightText: outHeightText.value,
      flipH: flipH.value,
      flipV: flipV.value,
      rotationQuarter: rotationQuarter.value,
      watermarkText: watermarkText.value,
      watermarkOpacity: watermarkOpacity.value,
      watermarkRotation: watermarkRotation.value,
      watermarkPosition: watermarkPosition.value,
      watermarkCustomX: watermarkCustomX.value,
      watermarkCustomY: watermarkCustomY.value,
      watermarkFontSize: watermarkFontSize.value,
      watermarkCompactness: watermarkCompactness.value,
    }
  }

  function applySnapshot(snapshot: EditorSnapshot) {
    applyingHistory = true
    crop.value = { ...snapshot.crop }
    lockAspect.value = snapshot.lockAspect
    outWidthText.value = snapshot.outWidthText
    outHeightText.value = snapshot.outHeightText
    flipH.value = snapshot.flipH
    flipV.value = snapshot.flipV
    rotationQuarter.value = snapshot.rotationQuarter
    watermarkText.value = snapshot.watermarkText
    watermarkOpacity.value = snapshot.watermarkOpacity
    watermarkRotation.value = snapshot.watermarkRotation
    watermarkPosition.value = snapshot.watermarkPosition
    watermarkCustomX.value = snapshot.watermarkCustomX
    watermarkCustomY.value = snapshot.watermarkCustomY
    watermarkFontSize.value = snapshot.watermarkFontSize
    watermarkCompactness.value = snapshot.watermarkCompactness ?? 100
    layerInnerTransform.value = ''
    transformAnimating.value = false
    applyingHistory = false
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  function clearEditHistory() {
    historyPast.value = []
    historyFuture.value = []
  }

  function recordEditHistory() {
    if (applyingHistory) return
    historyPast.value.push(captureSnapshot())
    if (historyPast.value.length > EDIT_HISTORY_LIMIT) {
      historyPast.value.shift()
    }
    historyFuture.value = []
  }

  const canUndo = computed(() => historyPast.value.length > 0)
  const canRedo = computed(() => historyFuture.value.length > 0)

  function undoEdit() {
    if (historyPast.value.length === 0) return false
    historyFuture.value.push(captureSnapshot())
    const prev = historyPast.value.pop()
    if (!prev) return false
    applySnapshot(prev)
    return true
  }

  function redoEdit() {
    if (historyFuture.value.length === 0) return false
    historyPast.value.push(captureSnapshot())
    const next = historyFuture.value.pop()
    if (!next) return false
    applySnapshot(next)
    return true
  }

  /** 选框在图层内定位，与 canvas 同坐标系，随缩放/平移一致 */
  const cropBoxStyle = computed(() => {
    const d = baseDisplay.value
    const c = crop.value
    return {
      left: `${c.x * d.w}px`,
      top: `${c.y * d.h}px`,
      width: `${c.w * d.w}px`,
      height: `${c.h * d.h}px`,
    }
  })

  const layerStyle = computed(() => {
    const d = baseDisplay.value
    return {
      left: `${d.offsetX + panX.value}px`,
      top: `${d.offsetY + panY.value}px`,
      width: `${d.w}px`,
      height: `${d.h}px`,
      transform: `scale(${zoom.value})`,
      transformOrigin: '0 0',
    }
  })

  const layerInnerStyle = computed(() => ({
    transform: layerInnerTransform.value || 'none',
  }))

  function prefersReducedMotion(): boolean {
    return (
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    )
  }

  function delay(ms: number) {
    return new Promise<void>((resolve) => {
      window.setTimeout(resolve, ms)
    })
  }

  async function runLayerTransformAnimation(fromTf: string, toTf: string, commit: () => void) {
    if (prefersReducedMotion() || fromTf === toTf) {
      commit()
      return
    }
    transformAnimating.value = true
    layerInnerTransform.value = fromTf
    await delay(20)
    layerInnerTransform.value = toTf
    await delay(LAYER_TRANSFORM_MS)
    transformAnimating.value = false
    layerInnerTransform.value = ''
    commit()
  }

  /** 在画布保持当前镜像状态时，叠加 CSS 以达到目标镜像效果 */
  function overlayCssToReachTarget(
    curH: boolean,
    curV: boolean,
    tgtH: boolean,
    tgtV: boolean,
  ): string {
    const parts: string[] = []
    if (curH !== tgtH) parts.push('scaleX(-1)')
    if (curV !== tgtV) parts.push('scaleY(-1)')
    return parts.join(' ')
  }

  const isPanning = computed(() => panning.value)

  const isPinching = computed(() => stagePointers.size >= 2 || pinchSession !== null)

  const naturalSizeText = computed(() => {
    const img = sourceImg.value
    if (!img) return '—'
    const { w, h } = effectiveNaturalSize(img, rotationQuarter.value)
    return `${w} × ${h} px`
  })

  const cropSizeText = computed(() => {
    const img = sourceImg.value
    if (!img) return '—'
    const { w, h } = effectiveNaturalSize(img, rotationQuarter.value)
    const c = crop.value
    return `${Math.max(1, Math.round(w * c.w))} × ${Math.max(1, Math.round(h * c.h))} px`
  })

  function effectiveNaturalSize(img: HTMLImageElement, quarter: number) {
    if (quarter % 2 === 1) {
      return { w: img.naturalHeight, h: img.naturalWidth }
    }
    return { w: img.naturalWidth, h: img.naturalHeight }
  }

  function drawSourceWithTransforms(
    ctx: CanvasRenderingContext2D,
    img: HTMLImageElement,
    destW: number,
    destH: number,
    quarter: number,
    mirrorH: boolean,
    mirrorV: boolean,
  ) {
    const rq = ((quarter % 4) + 4) % 4
    ctx.save()
    ctx.translate(destW / 2, destH / 2)
    if (rq === 1) ctx.rotate(-Math.PI / 2)
    else if (rq === 2) ctx.rotate(-Math.PI)
    else if (rq === 3) ctx.rotate(-(3 * Math.PI) / 2)
    if (mirrorH) ctx.scale(-1, 1)
    if (mirrorV) ctx.scale(1, -1)
    const localW = rq % 2 === 0 ? destW : destH
    const localH = rq % 2 === 0 ? destH : destW
    ctx.drawImage(img, -localW / 2, -localH / 2, localW, localH)
    ctx.restore()
  }

  function resetTransformState() {
    crop.value = { x: 0, y: 0, w: 1, h: 1 }
    outWidthText.value = ''
    outHeightText.value = ''
    flipH.value = false
    flipV.value = false
    rotationQuarter.value = 0
    watermarkText.value = ''
    watermarkOpacity.value = 55
    watermarkRotation.value = -24
    watermarkPosition.value = 'bottom-right'
    watermarkCustomX.value = 50
    watermarkCustomY.value = 50
    watermarkFontSize.value = 0
    watermarkCompactness.value = 100
    exportQualityPercent.value = 100
    exportEncoding.value = 'jpeg'
    zoom.value = 1
    panX.value = 0
    panY.value = 0
    cropReady.value = false
    layerInnerTransform.value = ''
    transformAnimating.value = false
    clearEditHistory()
  }

  async function loadSourceImage(src: string) {
    cropReady.value = false
    await new Promise<void>((resolve, reject) => {
      const img = new Image()
      img.crossOrigin = 'anonymous'
      img.onload = () => {
        sourceImg.value = img
        resolve()
      }
      img.onerror = () => reject(new Error('图片加载失败'))
      img.src = src
    })
    cropReady.value = true
  }

  function clampCrop(next: CropRect): CropRect {
    const minSize = 0.04
    next.w = Math.max(minSize, Math.min(1, next.w))
    next.h = Math.max(minSize, Math.min(1, next.h))
    next.x = Math.max(0, Math.min(1 - next.w, next.x))
    next.y = Math.max(0, Math.min(1 - next.h, next.y))
    return next
  }

  function paintStage(canvas: HTMLCanvasElement | null, stage: HTMLElement | null) {
    const img = sourceImg.value
    if (!canvas || !stage || !img) return

    const pad = 16
    const maxW = Math.max(1, stage.clientWidth - pad)
    const maxH = Math.max(1, stage.clientHeight - pad)
    const { w: effNatW, h: effNatH } = effectiveNaturalSize(img, rotationQuarter.value)
    const ratio = effNatW / effNatH
    let fitW = maxW
    let fitH = fitW / ratio
    if (fitH > maxH) {
      fitH = maxH
      fitW = fitH * ratio
    }
    baseDisplay.value = {
      w: fitW,
      h: fitH,
      offsetX: (stage.clientWidth - fitW) / 2,
      offsetY: (stage.clientHeight - fitH) / 2,
    }

    const previewMax = 8192
    let bufW = effNatW
    let bufH = effNatH
    if (bufW > previewMax || bufH > previewMax) {
      const scale = previewMax / Math.max(bufW, bufH)
      bufW = Math.max(1, Math.round(bufW * scale))
      bufH = Math.max(1, Math.round(bufH * scale))
    }
    canvas.width = bufW
    canvas.height = bufH
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.imageSmoothingEnabled = true
    ctx.imageSmoothingQuality = 'high'
    drawSourceWithTransforms(
      ctx,
      img,
      bufW,
      bufH,
      rotationQuarter.value,
      flipH.value,
      flipV.value,
    )
    drawWatermark(ctx, bufW, bufH)
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  function stagePointFromClient(stage: HTMLElement, clientX: number, clientY: number) {
    const rect = stage.getBoundingClientRect()
    return { x: clientX - rect.left, y: clientY - rect.top }
  }

  function applyZoomAtStagePoint(
    stage: HTMLElement,
    oldZoom: number,
    newZoom: number,
    mx: number,
    my: number,
  ) {
    const d = baseDisplay.value
    const layerX = mx - d.offsetX - panX.value
    const layerY = my - d.offsetY - panY.value
    const ratio = newZoom / oldZoom
    panX.value += layerX * (1 - ratio)
    panY.value += layerY * (1 - ratio)
  }

  function setStageZoom(stage: HTMLElement, newZoom: number, anchorX: number, anchorY: number) {
    const minZoom = getMinZoomToCoverCrop()
    const clamped = Math.min(6, Math.max(minZoom, newZoom))
    const oldZoom = zoom.value
    if (Math.abs(clamped - oldZoom) < 1e-6) return
    applyZoomAtStagePoint(stage, oldZoom, clamped, anchorX, anchorY)
    zoom.value = clamped
    clampPanToCoverCrop()
  }

  function pointerPairDistance(
    a: { x: number; y: number },
    b: { x: number; y: number },
  ): number {
    return Math.hypot(a.x - b.x, a.y - b.y)
  }

  function cancelCropDrag() {
    if (!cropDrag) return
    cropDrag = null
    window.removeEventListener('pointermove', onCropPointerMove)
  }

  function clearPendingCropTouch() {
    pendingCropTouch = null
    window.removeEventListener('pointermove', onPendingCropPointerMove)
    window.removeEventListener('pointerup', onPendingCropPointerUp)
  }

  function startCropDragFromPending() {
    if (!pendingCropTouch) return
    const pending = pendingCropTouch
    window.removeEventListener('pointermove', onPendingCropPointerMove)
    pendingCropTouch = null
    recordEditHistory()
    cropDrag = {
      mode: pending.mode,
      startX: pending.startX,
      startY: pending.startY,
      base: { ...pending.base },
    }
    window.addEventListener('pointermove', onCropPointerMove)
    window.addEventListener('pointerup', onCropPointerUp, { once: true })
  }

  function onPendingCropPointerMove(event: PointerEvent) {
    if (!pendingCropTouch || event.pointerId !== pendingCropTouch.pointerId) return
    if (stagePointers.size > 1 || pinchSession) {
      clearPendingCropTouch()
      return
    }
    const dx = event.clientX - pendingCropTouch.startX
    const dy = event.clientY - pendingCropTouch.startY
    if (Math.hypot(dx, dy) < CROP_TOUCH_MOVE_THRESHOLD) return
    startCropDragFromPending()
    onCropPointerMove(event)
  }

  function onPendingCropPointerUp(event: PointerEvent) {
    if (pendingCropTouch && event.pointerId === pendingCropTouch.pointerId) {
      clearPendingCropTouch()
    }
  }

  function beginPinchSession(stage: HTMLElement) {
    const pts = [...stagePointers.values()]
    const p0 = pts[0]
    const p1 = pts[1]
    if (p0 === undefined || p1 === undefined) return
    clearPendingCropTouch()
    cancelCropDrag()
    onStagePanPointerUp()
    const dist = Math.max(1, pointerPairDistance(p0, p1))
    pinchSession = {
      startDist: dist,
      startZoom: zoom.value,
      anchorX: (p0.x + p1.x) / 2,
      anchorY: (p0.y + p1.y) / 2,
    }
  }

  function onStageWheel(event: WheelEvent, stage: HTMLElement | null) {
    event.preventDefault()
    if (!stage) return
    const factor = event.deltaY > 0 ? 0.92 : 1.08
    const { x: mx, y: my } = stagePointFromClient(stage, event.clientX, event.clientY)
    setStageZoom(stage, zoom.value * factor, mx, my)
  }

  function onStagePointerCaptureDown(event: PointerEvent, stage: HTMLElement | null) {
    if (!stage) return
    stagePointers.set(event.pointerId, stagePointFromClient(stage, event.clientX, event.clientY))
    try {
      stage.setPointerCapture(event.pointerId)
    } catch {
      /* ignore */
    }
    if (stagePointers.size === 2) {
      beginPinchSession(stage)
      event.preventDefault()
      event.stopPropagation()
    }
  }

  function onStagePointerCaptureMove(event: PointerEvent, stage: HTMLElement | null) {
    if (!stage || !stagePointers.has(event.pointerId)) return
    stagePointers.set(event.pointerId, stagePointFromClient(stage, event.clientX, event.clientY))
    if (pinchSession) {
      cancelCropDrag()
    }
    if (stagePointers.size < 2 || !pinchSession) return
    const pts = [...stagePointers.values()]
    const p0 = pts[0]
    const p1 = pts[1]
    if (p0 === undefined || p1 === undefined) return
    const dist = pointerPairDistance(p0, p1)
    const scale = dist / pinchSession.startDist
    setStageZoom(stage, pinchSession.startZoom * scale, pinchSession.anchorX, pinchSession.anchorY)
    event.preventDefault()
  }

  function onStagePointerCaptureEnd(event: PointerEvent, stage: HTMLElement | null) {
    stagePointers.delete(event.pointerId)
    if (stage) {
      try {
        stage.releasePointerCapture(event.pointerId)
      } catch {
        /* ignore */
      }
    }
    if (stagePointers.size < 2) {
      pinchSession = null
    }
    if (stagePointers.size === 0) {
      onStagePanPointerUp(event)
    }
  }

  function clearStageInteraction() {
    stagePointers.clear()
    pinchSession = null
    clearPendingCropTouch()
    cancelCropDrag()
    onStagePanPointerUp()
  }

  function resetZoom() {
    zoom.value = Math.max(1, getMinZoomToCoverCrop())
    panX.value = 0
    panY.value = 0
    clampPanToCoverCrop()
  }

  const isZoomed = computed(
    () =>
      Math.abs(zoom.value - 1) > 0.01 ||
      Math.abs(panX.value) > 0.5 ||
      Math.abs(panY.value) > 0.5,
  )

  /** 100 = 1× 间距；越大越紧凑（间距越小） */
  function watermarkSpacingMultiplier(): number {
    const c = Math.max(50, Math.min(150, watermarkCompactness.value))
    return 2 - c / 100
  }

  function drawWatermark(ctx: CanvasRenderingContext2D, w: number, h: number) {
    const text = watermarkText.value.trim()
    if (!text) return
    const alpha = Math.min(1, Math.max(0.08, watermarkOpacity.value / 100))
    const fontSize =
      watermarkFontSize.value > 0
        ? watermarkFontSize.value
        : Math.max(12, Math.round(Math.min(w, h) * 0.038))
    const spacingMult = watermarkSpacingMultiplier()
    const letterEm = ((100 - watermarkCompactness.value) / 100) * 0.12
    ctx.save()
    ctx.fillStyle = `rgba(255, 255, 255, ${alpha})`
    ctx.font = `600 ${fontSize}px system-ui, sans-serif`
    ctx.letterSpacing = `${letterEm}em`
    ctx.shadowColor = 'rgba(0,0,0,0.35)'
    ctx.shadowBlur = Math.max(2, Math.round(fontSize * 0.15))
    const pos = watermarkPosition.value
    if (pos === 'tile') {
      ctx.translate(w / 2, h / 2)
      ctx.rotate((watermarkRotation.value * Math.PI) / 180)
      const stepX = fontSize * Math.max(6, text.length * 0.55) * spacingMult
      const stepY = fontSize * 3.2 * spacingMult
      for (let y = -h; y < h * 1.5; y += stepY) {
        for (let x = -w; x < w * 1.5; x += stepX) {
          ctx.fillText(text, x - w / 2, y - h / 2)
        }
      }
      ctx.restore()
      return
    }
    const pad = Math.max(8, Math.round(Math.min(w, h) * 0.03))
    let x = pad
    let y = pad
    let align: CanvasTextAlign = 'left'
    let baseline: CanvasTextBaseline = 'top'
    if (pos === 'custom') {
      x = (watermarkCustomX.value / 100) * w
      y = (watermarkCustomY.value / 100) * h
      align = 'center'
      baseline = 'middle'
    } else if (pos === 'center') {
      x = w / 2
      y = h / 2
      align = 'center'
      baseline = 'middle'
    } else if (pos === 'bottom-right') {
      x = w - pad
      y = h - pad
      align = 'right'
      baseline = 'bottom'
    } else if (pos === 'bottom-left') {
      x = pad
      y = h - pad
      align = 'left'
      baseline = 'bottom'
    } else if (pos === 'top-right') {
      x = w - pad
      y = pad
      align = 'right'
      baseline = 'top'
    } else if (pos === 'top-left') {
      x = pad
      y = pad
      align = 'left'
      baseline = 'top'
    }
    ctx.textAlign = align
    ctx.textBaseline = baseline
    if (pos === 'custom') ctx.rotate((watermarkRotation.value * Math.PI) / 180)
    ctx.fillText(text, x, y)
    ctx.restore()
  }

  function parsePositiveInt(raw: string): number | null {
    const n = Number.parseInt(raw.trim(), 10)
    if (!Number.isFinite(n) || n <= 0) return null
    return n
  }

  function hasEdits(): boolean {
    const cropChanged =
      crop.value.x > 0.001 ||
      crop.value.y > 0.001 ||
      crop.value.w < 0.999 ||
      crop.value.h < 0.999
    return (
      cropChanged ||
      flipH.value ||
      flipV.value ||
      rotationQuarter.value !== 0 ||
      outWidthText.value.trim() !== '' ||
      outHeightText.value.trim() !== '' ||
      watermarkText.value.trim() !== ''
    )
  }

  async function renderEditedBlob(): Promise<Blob | null> {
    const img = sourceImg.value
    if (!img) return null
    const { w: effNatW, h: effNatH } = effectiveNaturalSize(img, rotationQuarter.value)
    const c = crop.value
    const srcW = Math.max(1, Math.round(effNatW * c.w))
    const srcH = Math.max(1, Math.round(effNatH * c.h))
    let outW = parsePositiveInt(outWidthText.value) ?? srcW
    let outH = parsePositiveInt(outHeightText.value) ?? srcH
    if (lockAspect.value) {
      const ratio = srcW / srcH
      if (parsePositiveInt(outWidthText.value) != null && parsePositiveInt(outHeightText.value) == null) {
        outH = Math.max(1, Math.round(outW / ratio))
      } else if (parsePositiveInt(outHeightText.value) != null && parsePositiveInt(outWidthText.value) == null) {
        outW = Math.max(1, Math.round(outH * ratio))
      }
    }

    const oriented = document.createElement('canvas')
    oriented.width = effNatW
    oriented.height = effNatH
    const octx = oriented.getContext('2d')
    if (!octx) return null
    drawSourceWithTransforms(
      octx,
      img,
      effNatW,
      effNatH,
      rotationQuarter.value,
      flipH.value,
      flipV.value,
    )

    const canvas = document.createElement('canvas')
    canvas.width = outW
    canvas.height = outH
    const ctx = canvas.getContext('2d')
    if (!ctx) return null
    ctx.drawImage(
      oriented,
      c.x * effNatW,
      c.y * effNatH,
      srcW,
      srcH,
      0,
      0,
      outW,
      outH,
    )
    drawWatermark(ctx, outW, outH)
    const percent = exportQualityPercent.value
    if (usePngLosslessExport(exportEncoding.value)) {
      return new Promise<Blob | null>((resolve) => {
        canvas.toBlob((b) => resolve(b), 'image/png')
      })
    }
    const jpegQ = jpegQualityFromPercent(percent)
    return new Promise<Blob | null>((resolve) => {
      canvas.toBlob((b) => resolve(b), 'image/jpeg', jpegQ)
    })
  }

  async function estimateEditedBlobSize(): Promise<number | null> {
    const blob = await renderEditedBlob()
    return blob?.size ?? null
  }

  async function buildEditedFile(filenameHint: string): Promise<File | null> {
    const blob = await renderEditedBlob()
    if (!blob) return null
    const encoding = exportEncoding.value
    const name = resolveEditedExportFilename(filenameHint, encoding)
    const mime = exportMimeFromEncoding(encoding)
    return new File([blob], name, { type: mime })
  }

  function resetCropInternal() {
    crop.value = { x: 0, y: 0, w: 1, h: 1 }
    syncSizeFromCrop()
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  function resetCrop() {
    recordEditHistory()
    resetCropInternal()
  }

  function applyCropAspect(ratio: number) {
    const img = sourceImg.value
    if (!img || ratio <= 0) return
    const { w: effNatW, h: effNatH } = effectiveNaturalSize(img, rotationQuarter.value)
    const imgRatio = effNatW / effNatH
    let w = 1
    let h = 1
    let x = 0
    let y = 0
    if (imgRatio > ratio) {
      h = 1
      w = ratio / imgRatio
      x = (1 - w) / 2
    } else {
      w = 1
      h = imgRatio / ratio
      y = (1 - h) / 2
    }
    crop.value = clampCrop({ x, y, w, h })
    syncSizeFromCrop()
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  function applyPreset(preset: SizePreset) {
    recordEditHistory()
    outWidthText.value = String(preset.width)
    outHeightText.value = String(preset.height)
    lockAspect.value = true
    applyCropAspect(preset.width / preset.height)
  }

  function applyWatermarkPreset(preset: WatermarkPreset) {
    recordEditHistory()
    watermarkText.value = preset.text
    if (preset.opacity != null) watermarkOpacity.value = preset.opacity
    if (preset.rotation != null) watermarkRotation.value = preset.rotation
    if (preset.position) watermarkPosition.value = preset.position
    if (preset.fontSize != null) watermarkFontSize.value = preset.fontSize
    if (preset.compactness != null) watermarkCompactness.value = preset.compactness
    if (preset.customX != null) watermarkCustomX.value = preset.customX
    if (preset.customY != null) watermarkCustomY.value = preset.customY
  }

  function clearWatermark() {
    recordEditHistory()
    watermarkText.value = ''
  }

  function snapshotWatermarkPreset(label: string): WatermarkPreset {
    return {
      id: `custom-wm-${Date.now()}`,
      label,
      text: watermarkText.value.trim() || 'WATERMARK',
      opacity: watermarkOpacity.value,
      rotation: watermarkRotation.value,
      position: watermarkPosition.value,
      fontSize: watermarkFontSize.value,
      compactness: watermarkCompactness.value,
      customX: watermarkCustomX.value,
      customY: watermarkCustomY.value,
      custom: true,
    }
  }

  function syncSizeFromCrop() {
    if (sizeCropSyncLock) return
    const img = sourceImg.value
    if (!img) return
    const { w: effNatW, h: effNatH } = effectiveNaturalSize(img, rotationQuarter.value)
    const c = crop.value
    outWidthText.value = String(Math.max(1, Math.round(effNatW * c.w)))
    outHeightText.value = String(Math.max(1, Math.round(effNatH * c.h)))
  }

  function currentOutputAspect(): number {
    const img = sourceImg.value
    if (!img) return 1
    const { w: effNatW, h: effNatH } = effectiveNaturalSize(img, rotationQuarter.value)
    const c = crop.value
    const outW = Math.max(1, effNatW * c.w)
    const outH = Math.max(1, effNatH * c.h)
    return outW / outH
  }

  /** 锁定比例时，根据已填一侧补全另一侧输出尺寸 */
  function syncPairedOutputSize(edited?: 'w' | 'h' | null) {
    if (!lockAspect.value || pairingOutputLock) return
    const parsedW = parsePositiveInt(outWidthText.value)
    const parsedH = parsePositiveInt(outHeightText.value)
    if (parsedW == null && parsedH == null) return

    const aspect = currentOutputAspect()
    pairingOutputLock = true
    try {
      if (edited === 'w' && parsedW != null) {
        autoFillingOutput = 'h'
        outHeightText.value = String(Math.max(1, Math.round(parsedW / aspect)))
        return
      }
      if (edited === 'h' && parsedH != null) {
        autoFillingOutput = 'w'
        outWidthText.value = String(Math.max(1, Math.round(parsedH * aspect)))
        return
      }
      if (parsedW != null && parsedH == null) {
        autoFillingOutput = 'h'
        outHeightText.value = String(Math.max(1, Math.round(parsedW / aspect)))
      } else if (parsedH != null && parsedW == null) {
        autoFillingOutput = 'w'
        outWidthText.value = String(Math.max(1, Math.round(parsedH * aspect)))
      }
    } finally {
      pairingOutputLock = false
    }
  }

  function consumeAutoFilledOutputField(field: 'w' | 'h'): boolean {
    if (autoFillingOutput !== field) return false
    autoFillingOutput = null
    return true
  }

  function syncOutputSizeInput(edited?: 'w' | 'h' | null) {
    syncPairedOutputSize(edited)
    syncCropFromSize()
  }

  /** 根据输出宽高（像素）反推并更新裁切选框 */
  function syncCropFromSize() {
    const img = sourceImg.value
    if (!img) return
    const parsedW = parsePositiveInt(outWidthText.value)
    const parsedH = parsePositiveInt(outHeightText.value)
    if (parsedW == null && parsedH == null) return

    const { w: effNatW, h: effNatH } = effectiveNaturalSize(img, rotationQuarter.value)
    const prev = crop.value
    const cx = prev.x + prev.w / 2
    const cy = prev.y + prev.h / 2
    const imgAspect = effNatW / effNatH

    let w = prev.w
    let h = prev.h

    if (lockAspect.value) {
      const outAspect =
        parsedW != null && parsedH != null
          ? parsedW / parsedH
          : ((parsedW ?? Math.round(effNatW * prev.w)) / (parsedH ?? Math.round(effNatH * prev.h)))

      if (parsedW != null) {
        w = parsedW / effNatW
        h = (w * imgAspect) / outAspect
      } else if (parsedH != null) {
        h = parsedH / effNatH
        w = (h * outAspect) / imgAspect
      }
    } else {
      if (parsedW != null) w = parsedW / effNatW
      if (parsedH != null) h = parsedH / effNatH
    }

    if (w > 1) {
      const scale = 1 / w
      w = 1
      h *= scale
    }
    if (h > 1) {
      const scale = 1 / h
      h = 1
      w *= scale
    }

    let x = cx - w / 2
    let y = cy - h / 2
    crop.value = clampCrop({ x, y, w, h })
    sizeCropSyncLock = true
    syncSizeFromCrop()
    sizeCropSyncLock = false
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  function toggleFlipHorizontal(onRepaint: () => void) {
    recordEditHistory()
    const next = !flipH.value
    const from = ''
    const to = overlayCssToReachTarget(flipH.value, flipV.value, next, flipV.value)
    void runLayerTransformAnimation(from, to, () => {
      flipH.value = next
      onRepaint()
    })
  }

  function toggleFlipVertical(onRepaint: () => void) {
    recordEditHistory()
    const next = !flipV.value
    const from = ''
    const to = overlayCssToReachTarget(flipH.value, flipV.value, flipH.value, next)
    void runLayerTransformAnimation(from, to, () => {
      flipV.value = next
      onRepaint()
    })
  }

  function rotateCCW(onRepaint: () => void) {
    recordEditHistory()
    void runLayerTransformAnimation('rotate(0deg)', 'rotate(-90deg)', () => {
      rotationQuarter.value = (rotationQuarter.value + 1) % 4
      resetCropInternal()
      onRepaint()
    })
  }

  function resolveCropDragMode(target: EventTarget | null): CropDragMode {
    const el = target instanceof HTMLElement ? target.closest('.crop-handle') : null
    if (!el) return 'move'
    if (el.classList.contains('crop-handle--se')) return 'se'
    if (el.classList.contains('crop-handle--sw')) return 'sw'
    if (el.classList.contains('crop-handle--ne')) return 'ne'
    if (el.classList.contains('crop-handle--nw')) return 'nw'
    if (el.classList.contains('crop-handle--n')) return 'n'
    if (el.classList.contains('crop-handle--s')) return 's'
    if (el.classList.contains('crop-handle--e')) return 'e'
    if (el.classList.contains('crop-handle--w')) return 'w'
    return 'move'
  }

  /** Shift 拖拽缩放时保持起始裁切框的像素宽高比（归一化 w/h 比） */
  function applyShiftProportionalCrop(
    base: CropRect,
    next: CropRect,
    mode: CropDragMode,
  ): CropRect {
    const ratio = base.w / base.h
    if (!(ratio > 0) || !Number.isFinite(ratio)) return next

    const isCorner = mode === 'se' || mode === 'sw' || mode === 'ne' || mode === 'nw'
    if (isCorner) {
      const dw = next.w - base.w
      const dh = next.h - base.h
      const dwNorm = Math.abs(dw) / Math.max(base.w, 1e-6)
      const dhNorm = Math.abs(dh) / Math.max(base.h, 1e-6)
      let w: number
      let h: number
      if (dwNorm >= dhNorm) {
        w = next.w
        h = w / ratio
      } else {
        h = next.h
        w = h * ratio
      }
      if (mode === 'se') {
        return { x: base.x, y: base.y, w, h }
      }
      if (mode === 'sw') {
        return { x: base.x + base.w - w, y: base.y, w, h }
      }
      if (mode === 'ne') {
        return { x: base.x, y: base.y + base.h - h, w, h }
      }
      return { x: base.x + base.w - w, y: base.y + base.h - h, w, h }
    }

    if (mode === 'e' || mode === 'w') {
      const centerY = base.y + base.h / 2
      const w = next.w
      const h = w / ratio
      const x = mode === 'e' ? base.x : base.x + base.w - w
      return { x, y: centerY - h / 2, w, h }
    }

    if (mode === 'n' || mode === 's') {
      const centerX = base.x + base.w / 2
      const h = next.h
      const w = h * ratio
      const y = mode === 's' ? base.y : base.y + base.h - h
      return { x: centerX - w / 2, y, w, h }
    }

    return next
  }

  function onCropPointerDown(event: PointerEvent) {
    if (event.button !== 0) return
    if (stagePointers.size > 1 || pinchSession) return

    const mode = resolveCropDragMode(event.target)
    // 触摸拖选框内部：先等单指移动再启动，避免与双指捏合抢手势
    if (event.pointerType !== 'mouse' && mode === 'move') {
      clearPendingCropTouch()
      event.preventDefault()
      pendingCropTouch = {
        mode,
        startX: event.clientX,
        startY: event.clientY,
        base: { ...crop.value },
        pointerId: event.pointerId,
      }
      window.addEventListener('pointermove', onPendingCropPointerMove)
      window.addEventListener('pointerup', onPendingCropPointerUp, { once: true })
      return
    }

    event.preventDefault()
    recordEditHistory()
    cropDrag = {
      mode,
      startX: event.clientX,
      startY: event.clientY,
      base: { ...crop.value },
    }
    window.addEventListener('pointermove', onCropPointerMove)
    window.addEventListener('pointerup', onCropPointerUp, { once: true })
  }

  function onCropPointerMove(event: PointerEvent) {
    if (!cropDrag) return
    const d = baseDisplay.value
    const z = zoom.value
    if (d.w <= 0 || d.h <= 0 || z <= 0) return
    const dx = (event.clientX - cropDrag.startX) / (d.w * z)
    const dy = (event.clientY - cropDrag.startY) / (d.h * z)
    const base = cropDrag.base
    if (cropDrag.mode === 'move') {
      crop.value = clampCrop({ x: base.x + dx, y: base.y + dy, w: base.w, h: base.h })
      syncSizeFromCrop()
      clampPanToCoverCrop()
      enforceMinZoom()
      return
    }
    let next = { ...base }
    if (cropDrag.mode === 'se') {
      next.w = base.w + dx
      next.h = base.h + dy
    } else if (cropDrag.mode === 'sw') {
      next.x = base.x + dx
      next.w = base.w - dx
      next.h = base.h + dy
    } else if (cropDrag.mode === 'ne') {
      next.y = base.y + dy
      next.w = base.w + dx
      next.h = base.h - dy
    } else if (cropDrag.mode === 'nw') {
      next.x = base.x + dx
      next.y = base.y + dy
      next.w = base.w - dx
      next.h = base.h - dy
    } else if (cropDrag.mode === 'n') {
      next.y = base.y + dy
      next.h = base.h - dy
    } else if (cropDrag.mode === 's') {
      next.h = base.h + dy
    } else if (cropDrag.mode === 'e') {
      next.w = base.w + dx
    } else if (cropDrag.mode === 'w') {
      next.x = base.x + dx
      next.w = base.w - dx
    }
    if (event.shiftKey) {
      next = applyShiftProportionalCrop(base, next, cropDrag.mode)
    }
    crop.value = clampCrop(next)
    syncSizeFromCrop()
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  function onCropPointerUp() {
    cropDrag = null
    window.removeEventListener('pointermove', onCropPointerMove)
    syncSizeFromCrop()
    enforceMinZoom()
    clampPanToCoverCrop()
  }

  /** 缩小下限：显示尺寸至少与选框等大，才能完全盖住选框 */
  function getMinZoomToCoverCrop(): number {
    const c = crop.value
    return Math.max(c.w, c.h, 1e-4)
  }

  function enforceMinZoom() {
    const minZ = getMinZoomToCoverCrop()
    if (zoom.value < minZ - 1e-6) {
      zoom.value = minZ
      clampPanToCoverCrop()
    }
  }

  /** 保证缩放后的图层始终完全盖住选框（选框内不能露底） */
  function getPanBounds() {
    const d = baseDisplay.value
    const c = crop.value
    const z = zoom.value
    const w = d.w
    const h = d.h
    if (w <= 0 || h <= 0) {
      return { minPanX: 0, maxPanX: 0, minPanY: 0, maxPanY: 0, validX: true, validY: true }
    }
    const minPanX = (c.x + c.w) * w - w * z
    const maxPanX = c.x * w
    const minPanY = (c.y + c.h) * h - h * z
    const maxPanY = c.y * h
    return {
      minPanX,
      maxPanX,
      minPanY,
      maxPanY,
      validX: minPanX <= maxPanX,
      validY: minPanY <= maxPanY,
    }
  }

  function clampPanAxis(value: number, min: number, max: number, valid: boolean): number {
    if (!valid) return (min + max) / 2
    return Math.min(max, Math.max(min, value))
  }

  function clampPanToCoverCrop() {
    if (!cropReady.value) return
    const b = getPanBounds()
    panX.value = clampPanAxis(panX.value, b.minPanX, b.maxPanX, b.validX)
    panY.value = clampPanAxis(panY.value, b.minPanY, b.maxPanY, b.validY)
  }

  function dedupeSnapTargets(targets: number[]): number[] {
    const out: number[] = []
    for (const t of targets) {
      if (!out.some((u) => Math.abs(u - t) < 0.5)) out.push(t)
    }
    return out
  }

  /** 平移吸附目标：使缩放后图层上的裁切区域与选框叠合 */
  function getPanSnapTargets() {
    const d = baseDisplay.value
    const c = crop.value
    const z = zoom.value
    const w = d.w
    const h = d.h
    if (w <= 0 || h <= 0) return { panX: [] as number[], panY: [] as number[] }

    return {
      panX: dedupeSnapTargets([
        c.x * w * (1 - z),
        (c.x + c.w) * w * (1 - z),
        c.x * w,
        (c.x + c.w) * w - w * z,
      ]),
      panY: dedupeSnapTargets([
        c.y * h * (1 - z),
        (c.y + c.h) * h * (1 - z),
        c.y * h,
        (c.y + c.h) * h - h * z,
      ]),
    }
  }

  /** 松手时吸附到最近对齐点（拖动过程不吸附，避免与指针抢位） */
  function snapPanAxisOnRelease(value: number, targets: number[]): number {
    const first = targets[0]
    if (first === undefined) return value
    let nearest = first
    let minDist = Math.abs(nearest - value)
    for (let i = 1; i < targets.length; i += 1) {
      const t = targets[i]
      if (t === undefined) continue
      const dist = Math.abs(t - value)
      if (dist < minDist) {
        minDist = dist
        nearest = t
      }
    }
    return minDist <= PAN_SNAP_THRESHOLD_RELEASE ? nearest : value
  }

  function applyPanSnapOnRelease() {
    const b = getPanBounds()
    const { panX: targetsX, panY: targetsY } = getPanSnapTargets()
    const filterInBounds = (targets: number[], min: number, max: number, valid: boolean) => {
      if (!valid) return targets
      return targets.filter((t) => t >= min - 0.5 && t <= max + 0.5)
    }
    const snapX = filterInBounds(targetsX, b.minPanX, b.maxPanX, b.validX)
    const snapY = filterInBounds(targetsY, b.minPanY, b.maxPanY, b.validY)
    if (snapX.length) panX.value = snapPanAxisOnRelease(panX.value, snapX)
    if (snapY.length) panY.value = snapPanAxisOnRelease(panY.value, snapY)
    clampPanToCoverCrop()
  }

  function onStagePanPointerDown(event: PointerEvent) {
    const isRightMouse = event.button === 2
    const isTouchPan = event.button === 0 && event.pointerType !== 'mouse'
    if (!isRightMouse && !isTouchPan) return
    if (pinchSession || stagePointers.size > 1) return
    event.preventDefault()
    panning.value = true
    panDrag = {
      startX: event.clientX,
      startY: event.clientY,
      basePanX: panX.value,
      basePanY: panY.value,
    }
    window.addEventListener('pointermove', onStagePanPointerMove)
    window.addEventListener('pointerup', onStagePanPointerUp)
  }

  function onStagePanPointerMove(event: PointerEvent) {
    if (!panDrag) return
    panX.value = panDrag.basePanX + (event.clientX - panDrag.startX)
    panY.value = panDrag.basePanY + (event.clientY - panDrag.startY)
    clampPanToCoverCrop()
  }

  function onStagePanPointerUp(event?: PointerEvent) {
    if (panDrag && event) {
      panX.value = panDrag.basePanX + (event.clientX - panDrag.startX)
      panY.value = panDrag.basePanY + (event.clientY - panDrag.startY)
      clampPanToCoverCrop()
      applyPanSnapOnRelease()
    }
    panDrag = null
    panning.value = false
    window.removeEventListener('pointermove', onStagePanPointerMove)
    window.removeEventListener('pointerup', onStagePanPointerUp)
  }

  return {
    sourceImg,
    cropReady,
    crop,
    baseDisplay,
    lockAspect,
    outWidthText,
    outHeightText,
    flipH,
    flipV,
    rotationQuarter,
    watermarkText,
    watermarkOpacity,
    watermarkRotation,
    watermarkPosition,
    watermarkCustomX,
    watermarkCustomY,
    watermarkFontSize,
    watermarkCompactness,
    exportQualityPercent,
    exportEncoding,
    zoom,
    panX,
    panY,
    isPanning,
    isPinching,
    cropBoxStyle,
    layerStyle,
    layerInnerStyle,
    transformAnimating,
    toggleFlipHorizontal,
    toggleFlipVertical,
    naturalSizeText,
    cropSizeText,
    resetTransformState,
    loadSourceImage,
    paintStage,
    hasEdits,
    estimateEditedBlobSize,
    buildEditedFile,
    resetCrop,
    rotateCCW,
    recordEditHistory,
    clearEditHistory,
    canUndo,
    canRedo,
    undoEdit,
    redoEdit,
    applyCropAspect,
    applyPreset,
    applyWatermarkPreset,
    clearWatermark,
    syncSizeFromCrop,
    syncPairedOutputSize,
    consumeAutoFilledOutputField,
    syncCropFromSize,
    syncOutputSizeInput,
    onStageWheel,
    onStagePointerCaptureDown,
    onStagePointerCaptureMove,
    onStagePointerCaptureEnd,
    clearStageInteraction,
    resetZoom,
    isZoomed,
    onStagePanPointerDown,
    onStagePanPointerUp,
    snapshotWatermarkPreset,
    onCropPointerDown,
    onCropPointerUp,
  }
}
