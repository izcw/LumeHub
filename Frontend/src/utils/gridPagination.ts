import type { MasonryColumnChoice } from '@/stores/masonryLayout'

export const GRID_COL_BREAKPOINTS: readonly { minWidth: number; cols: number }[] = [
  { minWidth: 1200, cols: 4 },
  { minWidth: 768, cols: 3 },
  { minWidth: 481, cols: 2 },
  { minWidth: 0, cols: 2 },
]

export const GRID_PAGE_SIZE_TARGETS = [10, 20, 30, 50, 100] as const

/** 与 GalleryView 中 --index-gallery-gap 一致 */
export const GRID_GAP_FALLBACK_PX = 12

const PAGINATION_RESERVE_PX = 88

export function gridColsForWidth(w: number): number {
  for (const bp of GRID_COL_BREAKPOINTS) {
    if (w >= bp.minWidth) return bp.cols
  }
  return 2
}

export function resolveGridColumnCount(
  containerWidth: number,
  columnChoice: MasonryColumnChoice,
): number {
  if (containerWidth <= 0) return 1
  if (columnChoice === 'auto') return gridColsForWidth(containerWidth)
  return Math.min(6, Math.max(1, columnChoice))
}

export function readGalleryGapPx(el: HTMLElement): number {
  const raw = getComputedStyle(el).getPropertyValue('--index-gallery-gap').trim()
  if (!raw) return GRID_GAP_FALLBACK_PX
  const probe = document.createElement('div')
  probe.style.position = 'absolute'
  probe.style.visibility = 'hidden'
  probe.style.width = raw
  document.body.appendChild(probe)
  const px = probe.getBoundingClientRect().width
  probe.remove()
  return px > 0 ? px : GRID_GAP_FALLBACK_PX
}

export function measureAvailableGridHeight(gridEl: HTMLElement): number {
  const top = gridEl.getBoundingClientRect().top
  return Math.max(160, window.innerHeight - top - PAGINATION_RESERVE_PX)
}

/** 网格卡片固定 4:3，按容器宽度估算每屏可容纳行数 */
export function estimateGridRowsPerScreen(input: {
  containerWidth: number
  columns: number
  gapPx: number
  availableHeight: number
}): number {
  const { containerWidth, columns, gapPx, availableHeight } = input
  if (columns <= 0 || containerWidth <= 0 || availableHeight <= 0) return 1
  const cellWidth = (containerWidth - (columns - 1) * gapPx) / columns
  const cellHeight = cellWidth * (3 / 4)
  const rowStride = cellHeight + gapPx
  return Math.max(1, Math.floor((availableHeight + gapPx) / rowStride))
}

/** 以「列×行」为块，生成接近 10/20/30/50/100 的每页选项 */
export function buildGridPageSizeOptions(columns: number, rowsPerScreen: number): number[] {
  const block = columns * rowsPerScreen
  if (block <= 0) return [20]
  const sizes = GRID_PAGE_SIZE_TARGETS.map((target) => {
    const multiplier = Math.max(1, Math.ceil(target / block))
    return multiplier * block
  })
  return [...new Set(sizes)]
}

export function snapPageSizeToGridOptions(
  current: number,
  options: readonly number[],
): number {
  if (options.length === 0) return Math.max(1, current)
  if (options.includes(current)) return current
  let best = options[0]!
  let bestDist = Math.abs(current - best)
  for (const opt of options) {
    const dist = Math.abs(current - opt)
    if (dist < bestDist) {
      best = opt
      bestDist = dist
    }
  }
  return best
}
