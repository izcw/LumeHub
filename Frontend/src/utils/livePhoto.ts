const STILL_EXTS = new Set(['jpg', 'jpeg', 'heic', 'png', 'webp'])
const MOTION_EXTS = new Set(['mov', 'm4v'])

export type LivePhotoUploadBatch = {
  main: File
  motion?: File
  label: string
}

export function fileExtLower(name: string): string {
  const base = name.replace(/\\/g, '/').split('/').pop() ?? name
  const i = base.lastIndexOf('.')
  return i > 0 ? base.slice(i + 1).toLowerCase() : ''
}

export function fileBaseStem(name: string): string {
  const base = name.replace(/\\/g, '/').split('/').pop() ?? name
  const i = base.lastIndexOf('.')
  return (i > 0 ? base.slice(0, i) : base).toLowerCase()
}

/** 将同主名的静图 + MOV 配对为实况图上传批次。 */
export function buildLivePhotoUploadBatches(files: readonly File[]): LivePhotoUploadBatch[] {
  const stills = new Map<string, File>()
  const motions = new Map<string, File>()
  const others: File[] = []

  for (const file of files) {
    const ext = fileExtLower(file.name)
    const stem = fileBaseStem(file.name)
    if (STILL_EXTS.has(ext)) {
      stills.set(stem, file)
    } else if (MOTION_EXTS.has(ext)) {
      motions.set(stem, file)
    } else {
      others.push(file)
    }
  }

  const usedMotion = new Set<string>()
  const batches: LivePhotoUploadBatch[] = []

  for (const [stem, main] of stills) {
    const motion = motions.get(stem)
    if (motion) {
      batches.push({ main, motion, label: main.name })
      usedMotion.add(stem)
    } else {
      batches.push({ main, label: main.name })
    }
  }

  for (const [stem, motion] of motions) {
    if (!usedMotion.has(stem)) {
      batches.push({ main: motion, label: motion.name })
    }
  }

  for (const file of others) {
    batches.push({ main: file, label: file.name })
  }

  return batches
}

export function isLivePhotoItem(item: {
  isLivePhoto?: boolean
  liveVideoUrl?: string
  tags?: readonly string[]
}): boolean {
  if (item.isLivePhoto) return true
  if (item.liveVideoUrl?.trim()) return true
  return item.tags?.some((t) => t.toLowerCase() === 'live') ?? false
}
