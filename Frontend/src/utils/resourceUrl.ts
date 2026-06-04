/** 从 payload 解析短链别名（不含后缀） */
export function resolveLinkStem(payload: {
  linkName?: string
  shortUrl?: string
}): string {
  const ln = payload.linkName?.trim()
  if (ln) return linkNameStem(ln)
  const short = payload.shortUrl?.trim()
  if (short) return linkNameStem(short.split('/').pop() ?? short)
  return ''
}

/** 转为完整绝对 URL */
export function toAbsoluteResourceUrl(url: string): string {
  const raw = url.trim()
  if (!raw) return ''
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw
  const path = raw.startsWith('/') ? raw : `/${raw}`
  return `${window.location.origin}${path}`
}

function fileBaseName(name: string): string {
  const raw = name.trim()
  if (!raw) return ''
  const segment = raw.split('/').pop() ?? raw
  return segment.split('?')[0]?.split('#')[0] ?? segment
}

/** 从 linkName 或文件名提取不含后缀的短链别名 */
export function linkNameStem(name: string): string {
  const base = fileBaseName(name.replace(/^\/+/, ''))
  if (!base) return ''
  const dot = base.lastIndexOf('.')
  return dot > 0 ? base.slice(0, dot) : base
}

/** 从原版文件名提取后缀（含点） */
export function linkNameExtension(name: string): string {
  const base = fileBaseName(name)
  if (!base) return ''
  const dot = base.lastIndexOf('.')
  return dot >= 0 ? base.slice(dot).toLowerCase() : ''
}

/** 全局短链预览：http://host/resource/{stem} */
export function buildShortLinkPreview(stem: string): string {
  const alias = linkNameStem(stem)
  if (!alias) return ''
  return `${window.location.origin}/resource/${alias}`
}

/** 为资源 URL 附加加密查看参数 ?k= */
export function appendViewKeyParam(rawURL: string, viewKey: string): string {
  const url = rawURL.trim()
  const key = viewKey.trim()
  if (!url || !key) return url
  const base = url.split('?')[0] ?? url
  return `${base}?k=${encodeURIComponent(key)}`
}

/** 生成可分享的绝对资源链接（加密相册在已有 viewKey 时自动附带 ?k=） */
export function buildShareableResourceUrl(
  rawUrl: string,
  opts?: { requiresViewKey?: boolean; viewKey?: string },
): string {
  const abs = toAbsoluteResourceUrl(rawUrl)
  if (!abs) return ''
  if (!opts?.requiresViewKey) return abs
  const key = opts.viewKey?.trim()
  if (!key) return abs.split('?')[0] ?? abs
  return appendViewKeyParam(abs, key)
}

/** 组合完整 linkName（别名 + 固定后缀） */
export function composeLinkName(stem: string, ext: string): string {
  const s = linkNameStem(stem)
  if (!s) return ''
  const suffix = ext.startsWith('.') ? ext : ext ? `.${ext}` : ''
  return `${s}${suffix}`
}

/** 编辑弹窗应始终加载 original/ 原图，避免误用缩略图或 edited/ 展示图 */
export function resolveEditorOriginalUrl(payload: {
  originalUrl?: string
  fullSrc?: string
  editedUrl?: string
}): string {
  const orig = payload.originalUrl?.trim()
  if (orig && !orig.includes('/thumb/')) {
    return orig
  }

  const tryReplaceEdited = (url: string) => {
    if (url.includes('/edited/')) {
      return url.replace('/edited/', '/original/')
    }
    return url
  }

  const edited = payload.editedUrl?.trim()
  if (edited) {
    return tryReplaceEdited(edited)
  }

  const full = payload.fullSrc?.trim() ?? ''
  if (full.includes('/thumb/')) {
    return full.replace(/\/thumb\//, '/original/').replace(/-thumbnail(?=\.)/i, '')
  }
  return tryReplaceEdited(full)
}

/** 列表/预览用缩略图地址 */
export function resolveEditorThumbnailUrl(payload: {
  thumbSrc?: string
  originalUrl?: string
  fullSrc?: string
  editedUrl?: string
}): string {
  const thumb = payload.thumbSrc?.trim()
  if (thumb) return thumb

  const candidates = [
    payload.fullSrc?.trim(),
    payload.originalUrl?.trim(),
    payload.editedUrl?.trim(),
  ].filter((u): u is string => Boolean(u))

  for (const url of candidates) {
    if (url.includes('/thumb/')) return url
    if (url.includes('/original/')) return url.replace('/original/', '/thumb/')
    if (url.includes('/edited/')) return url.replace('/edited/', '/thumb/')
  }
  return ''
}
