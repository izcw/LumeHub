/** 与后端 ordering.RankInt 一致：0 / undefined 排靠后 */
export function rankSortField(sort: number | undefined): number {
  if (sort == null || sort === 0) {
    return 1 << 30
  }
  return sort
}
