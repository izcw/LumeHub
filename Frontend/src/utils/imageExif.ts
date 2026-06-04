export type ExifResult = {
  dateTime?: string
  dateTimeOriginal?: string
  pixelWidth?: string
  pixelHeight?: string
  artist?: string
  make?: string
  model?: string
  aperture?: string
  exposure?: string
  iso?: string
  exposureBias?: string
  focalLength?: string
  maxAperture?: string
  meteringMode?: string
  flash?: string
  software?: string
}

const METERING: Record<number, string> = {
  0: '未知',
  1: '平均',
  2: '中央重点',
  3: '点测光',
  4: '多点',
  5: '多区',
  6: '局部',
  255: '其他',
}

const FLASH: Record<number, string> = {
  0: '未闪光',
  1: '闪光',
  5: '闪光（未检测到回闪）',
  7: '闪光（检测到回闪）',
  16: '强制关闭',
  24: '强制关闭（未检测到回闪）',
  25: '强制闪光',
  31: '强制闪光（检测到回闪）',
  32: '未闪光',
  65: '闪光（红眼）',
}

function readAscii(data: DataView, offset: number, len: number): string {
  let s = ''
  for (let i = 0; i < len; i++) {
    const c = data.getUint8(offset + i)
    if (c === 0) break
    s += String.fromCharCode(c)
  }
  return s.trim()
}

function parseRational(data: DataView, tiff: number, offset: number, little: boolean): number | null {
  const u32 = (o: number) => (little ? data.getUint32(o, true) : data.getUint32(o, false))
  const num = u32(tiff + offset)
  const den = u32(tiff + offset + 4)
  if (den === 0) return null
  return num / den
}

function formatExposure(v: number): string {
  if (v >= 1) return `${v.toFixed(1)}s`
  return `1/${Math.max(1, Math.round(1 / v))}s`
}

function formatBias(v: number): string {
  const sign = v > 0 ? '+' : ''
  return `${sign}${v.toFixed(1)} EV`
}

type IfdCtx = {
  data: DataView
  tiff: number
  little: boolean
  u16: (o: number) => number
  u32: (o: number) => number
}

function readIfd(ctx: IfdCtx, ifdOffset: number, out: ExifResult) {
  const { data, tiff, little, u16, u32 } = ctx
  const ifd = tiff + ifdOffset
  if (ifd + 2 > data.byteLength) return
  const n = u16(ifd)
  for (let i = 0; i < n; i++) {
    const e = ifd + 2 + i * 12
    if (e + 12 > data.byteLength) break
    const tag = u16(e)
    const type = u16(e + 2)
    const count = u32(e + 4)
    let valOff = e + 8

    if (type === 2 && count > 0) {
      if (count > 4) valOff = tiff + u32(valOff)
      const text = readAscii(data, valOff, Math.min(count, 128))
      switch (tag) {
        case 0x010e:
          out.software = text
          break
        case 0x010f:
          out.make = text
          break
        case 0x0110:
          out.model = text
          break
        case 0x0132:
          out.dateTime = text
          break
        case 0x013b:
          out.artist = text
          break
        case 0x9003:
          out.dateTimeOriginal = text
          break
        default:
          break
      }
    } else if (type === 3 && count === 1) {
      const v = u16(valOff)
      if (tag === 0x8827) out.iso = String(v)
      if (tag === 0x9207) out.meteringMode = METERING[v] ?? String(v)
      if (tag === 0x9209) out.flash = FLASH[v] ?? String(v)
      if (tag === 0xa002) out.pixelWidth = String(v)
      if (tag === 0xa003) out.pixelHeight = String(v)
    } else if (type === 5 && count === 1) {
      const off = count > 1 ? tiff + u32(valOff) : tiff + valOff
      const v = parseRational(data, tiff, off - tiff, little)
      if (v == null) continue
      if (tag === 0x829a) out.exposure = formatExposure(v)
      if (tag === 0x829d) out.aperture = `f/${v.toFixed(1)}`
      if (tag === 0x9204) out.exposureBias = formatBias(v)
      if (tag === 0x920a) out.focalLength = `${v.toFixed(0)}mm`
      if (tag === 0x9205) out.maxAperture = `f/${v.toFixed(1)}`
    } else if (type === 10 && count === 1) {
      const off = tiff + u32(valOff)
      const v = parseRational(data, tiff, off - tiff, little)
      if (v == null) continue
      if (tag === 0x9204) out.exposureBias = formatBias(v)
    }

    if (tag === 0x8769 && count === 1) {
      readIfd(ctx, u32(valOff), out)
    }
  }
}

function parseExifFromAPP1(data: DataView, offset: number, length: number): ExifResult {
  const out: ExifResult = {}
  if (length < 8) return out
  if (readAscii(data, offset, 4) !== 'Exif') return out
  const tiff = offset + 6
  const little = data.getUint8(tiff) === 0x49
  const u16 = (o: number) => (little ? data.getUint16(o, true) : data.getUint16(o, false))
  const u32 = (o: number) => (little ? data.getUint32(o, true) : data.getUint32(o, false))
  readIfd({ data, tiff, little, u16, u32 }, u32(tiff + 4), out)
  return out
}

/** 从图片 URL 读取 EXIF，失败时返回空字段 */
export async function readImageExif(url: string): Promise<ExifResult> {
  try {
    const res = await fetch(url)
    if (!res.ok) return {}
    const buf = await res.arrayBuffer()
    const data = new DataView(buf)
    if (data.byteLength < 4 || data.getUint16(0) !== 0xffd8) return {}
    let offset = 2
    while (offset + 4 < data.byteLength) {
      if (data.getUint8(offset) !== 0xff) break
      const marker = data.getUint8(offset + 1)
      if (marker === 0xda) break
      const len = data.getUint16(offset + 2)
      if (len < 2) break
      if (marker === 0xe1) {
        return parseExifFromAPP1(data, offset + 4, len - 2)
      }
      offset += 2 + len
    }
  } catch {
    /* ignore */
  }
  return {}
}

export function exifToDisplayRows(
  exif: ExifResult,
  imageSize?: { w: number; h: number },
): { label: string; value: string }[] {
  const rows: { label: string; value: string }[] = []
  const resolution =
    exif.pixelWidth && exif.pixelHeight
      ? `${exif.pixelWidth} × ${exif.pixelHeight} px`
      : imageSize && imageSize.w > 0
        ? `${imageSize.w} × ${imageSize.h} px`
        : ''
  if (exif.dateTime) rows.push({ label: '修改日期', value: exif.dateTime })
  if (exif.dateTimeOriginal) rows.push({ label: '拍摄日期', value: exif.dateTimeOriginal })
  if (resolution) rows.push({ label: '分辨率', value: resolution })
  if (exif.artist) rows.push({ label: '作者', value: exif.artist })
  if (exif.make) rows.push({ label: '制造商', value: exif.make })
  if (exif.model) rows.push({ label: '相机型号', value: exif.model })
  if (exif.aperture) rows.push({ label: '光圈', value: exif.aperture })
  if (exif.exposure) rows.push({ label: '曝光', value: exif.exposure })
  if (exif.iso) rows.push({ label: 'ISO', value: exif.iso })
  if (exif.exposureBias) rows.push({ label: '曝光补偿', value: exif.exposureBias })
  if (exif.focalLength) rows.push({ label: '焦距', value: exif.focalLength })
  if (exif.maxAperture) rows.push({ label: '最大光圈', value: exif.maxAperture })
  if (exif.meteringMode) rows.push({ label: '测光模式', value: exif.meteringMode })
  if (exif.flash) rows.push({ label: '闪光灯', value: exif.flash })
  if (exif.software) rows.push({ label: '软件', value: exif.software })
  return rows
}
