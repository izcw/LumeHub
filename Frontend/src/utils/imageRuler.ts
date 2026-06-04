/** 根据图像像素跨度选择标尺主刻度步长（像素） */
export function chooseRulerStep(spanPx: number): number {
  const steps = [1, 2, 5, 10, 20, 50, 100, 200, 250, 500, 1000, 2000, 5000, 10000]
  if (spanPx <= 0) return 100
  for (const step of steps) {
    if (spanPx / step <= 10) return step
  }
  return steps[steps.length - 1]!
}

export type RulerMark = {
  pos: number
  label?: string
  major: boolean
}

/** 生成 Photoshop 风格标尺刻度（主刻度带数字，次刻度仅在间距足够时显示） */
export function buildRulerMarks(imagePx: number, displayPx: number): RulerMark[] {
  if (imagePx <= 0 || displayPx <= 0) return []
  const majorStep = chooseRulerStep(imagePx)
  const marks: RulerMark[] = []

  for (let px = 0; px <= imagePx + 0.5; px += majorStep) {
    const rounded = Math.min(imagePx, Math.round(px))
    marks.push({
      pos: (rounded / imagePx) * displayPx,
      label: String(rounded),
      major: true,
    })
  }

  const minorDiv = majorStep >= 200 ? 10 : 5
  const minorStep = majorStep / minorDiv
  const minorSpacingPx = (minorStep / imagePx) * displayPx
  if (minorSpacingPx < 7) return marks

  const majorPositions = new Set(marks.map((m) => Math.round(m.pos * 100)))
  for (let px = minorStep; px < imagePx; px += minorStep) {
    if (Math.abs(px % majorStep) < minorStep * 0.25) continue
    const pos = (px / imagePx) * displayPx
    const key = Math.round(pos * 100)
    if (majorPositions.has(key)) continue
    marks.push({ pos, major: false })
  }

  return marks.sort((a, b) => a.pos - b.pos)
}
