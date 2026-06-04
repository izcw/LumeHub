/** /gallery/search 地址栏与搜索 store 同步（刷新可恢复） */

export type GallerySearchQueryState = {
  query: string
  categories: string[]
  extensions: string[]
  tags: string[]
  /** 进入搜索前所在画廊 folderKey */
  from?: string
}

function pickQueryParam(
  query: Record<string, string | string[] | undefined | null>,
  key: string,
): string {
  const v = query[key]
  if (typeof v === 'string') return v
  if (Array.isArray(v)) return (v[0] ?? '').trim()
  return ''
}

function splitCsv(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

export function parseGallerySearchQuery(
  query: Record<string, string | string[] | undefined | null>,
): GallerySearchQueryState {
  const categoriesRaw =
    pickQueryParam(query, 'categories') || pickQueryParam(query, 'category')
  const extensionsRaw =
    pickQueryParam(query, 'extensions') ||
    pickQueryParam(query, 'format') ||
    pickQueryParam(query, 'formats')
  const tagsRaw = pickQueryParam(query, 'tags') || pickQueryParam(query, 'tag')
  return {
    query: pickQueryParam(query, 'q'),
    categories: splitCsv(categoriesRaw),
    extensions: splitCsv(extensionsRaw),
    tags: splitCsv(tagsRaw),
    from: pickQueryParam(query, 'from') || undefined,
  }
}

export function buildGallerySearchQuery(state: GallerySearchQueryState): Record<string, string> {
  const out: Record<string, string> = {}
  const q = state.query.trim()
  if (q) out.q = q
  if (state.categories.length) out.categories = state.categories.join(',')
  if (state.extensions.length) out.extensions = state.extensions.join(',')
  if (state.tags.length) out.tags = state.tags.join(',')
  const from = state.from?.trim()
  if (from) out.from = from
  return out
}

export function gallerySearchQueryHasCriteria(
  state: Pick<GallerySearchQueryState, 'query' | 'categories' | 'extensions' | 'tags'>,
): boolean {
  return (
    state.query.trim().length > 0 ||
    state.categories.length > 0 ||
    state.extensions.length > 0 ||
    state.tags.length > 0
  )
}
