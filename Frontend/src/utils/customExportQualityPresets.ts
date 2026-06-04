import type { ExportQualityPreset } from '@/composables/useGalleryImageEditor'

const STORAGE_KEY = 'lumehub-custom-export-quality-presets'

export function loadCustomExportQualityPresets(): ExportQualityPreset[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as ExportQualityPreset[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (p) =>
        p &&
        typeof p.id === 'string' &&
        typeof p.label === 'string' &&
        typeof p.percent === 'number' &&
        p.percent >= 10 &&
        p.percent <= 100,
    )
  } catch {
    return []
  }
}

export function saveCustomExportQualityPresets(presets: ExportQualityPreset[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(presets))
}

export function addCustomExportQualityPreset(preset: ExportQualityPreset): ExportQualityPreset[] {
  const next = { ...preset, custom: true as const }
  const all = [...loadCustomExportQualityPresets(), next]
  saveCustomExportQualityPresets(all)
  return all
}

export function removeCustomExportQualityPreset(id: string): ExportQualityPreset[] {
  const all = loadCustomExportQualityPresets().filter((p) => p.id !== id)
  saveCustomExportQualityPresets(all)
  return all
}
