import type { WatermarkPreset } from '@/composables/useGalleryImageEditor'

const STORAGE_KEY = 'lumehub-custom-watermark-presets'

export function loadCustomWatermarkPresets(): WatermarkPreset[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as WatermarkPreset[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter((p) => p && typeof p.id === 'string' && typeof p.label === 'string' && typeof p.text === 'string')
  } catch {
    return []
  }
}

export function saveCustomWatermarkPresets(presets: WatermarkPreset[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(presets))
}

export function addCustomWatermarkPreset(preset: WatermarkPreset): WatermarkPreset[] {
  const next = { ...preset, custom: true as const }
  const all = [...loadCustomWatermarkPresets(), next]
  saveCustomWatermarkPresets(all)
  return all
}

export function removeCustomWatermarkPreset(id: string): WatermarkPreset[] {
  const all = loadCustomWatermarkPresets().filter((p) => p.id !== id)
  saveCustomWatermarkPresets(all)
  return all
}
