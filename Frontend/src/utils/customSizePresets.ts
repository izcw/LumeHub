import type { SizePreset } from '@/composables/useGalleryImageEditor'

const STORAGE_KEY = 'lumehub-custom-size-presets'

export function loadCustomSizePresets(): SizePreset[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as SizePreset[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (p) =>
        p &&
        typeof p.id === 'string' &&
        typeof p.label === 'string' &&
        typeof p.width === 'number' &&
        typeof p.height === 'number',
    )
  } catch {
    return []
  }
}

export function saveCustomSizePresets(presets: SizePreset[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(presets))
}

export function addCustomSizePreset(preset: Omit<SizePreset, 'id'> & { id?: string }): SizePreset[] {
  const next: SizePreset = {
    id: preset.id || `custom-${Date.now()}`,
    label: preset.label,
    width: preset.width,
    height: preset.height,
    desc: preset.desc || '自定义预设',
    custom: true,
  }
  const all = [...loadCustomSizePresets(), next]
  saveCustomSizePresets(all)
  return all
}

export function removeCustomSizePreset(id: string): SizePreset[] {
  const all = loadCustomSizePresets().filter((p) => p.id !== id)
  saveCustomSizePresets(all)
  return all
}
