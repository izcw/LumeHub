import { defineStore } from 'pinia'
import { ref } from 'vue'

export type MasonryColumnChoice = 'auto' | 1 | 2 | 3 | 4 | 5 | 6

export type GalleryLayoutMode = 'masonry' | 'grid'

const STORAGE_KEY = 'lumehub-masonry-column-choice'
const LAYOUT_MODE_KEY = 'lumehub-gallery-layout-mode'

function readStoredLayoutMode(): GalleryLayoutMode {
  try {
    const raw = localStorage.getItem(LAYOUT_MODE_KEY)
    if (raw === 'grid' || raw === 'masonry') return raw
  } catch {
    /* ignore */
  }
  return 'masonry'
}

function persistLayoutMode(mode: GalleryLayoutMode) {
  try {
    localStorage.setItem(LAYOUT_MODE_KEY, mode)
  } catch {
    /* ignore */
  }
}

function readStoredColumnChoice(): MasonryColumnChoice {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === null) return 'auto'
    if (raw === 'auto') return 'auto'
    const n = Number(raw)
    if (n >= 1 && n <= 6) return n as Exclude<MasonryColumnChoice, 'auto'>
  } catch {
    /* ignore */
  }
  return 'auto'
}

function persistColumnChoice(choice: MasonryColumnChoice) {
  try {
    localStorage.setItem(STORAGE_KEY, choice === 'auto' ? 'auto' : String(choice))
  } catch {
    /* ignore */
  }
}

export const useMasonryLayoutStore = defineStore('masonryLayout', () => {
  const columnChoice = ref<MasonryColumnChoice>(readStoredColumnChoice())
  const galleryLayoutMode = ref<GalleryLayoutMode>(readStoredLayoutMode())

  function setColumnChoice(choice: MasonryColumnChoice) {
    columnChoice.value = choice
    persistColumnChoice(choice)
  }

  function setGalleryLayoutMode(mode: GalleryLayoutMode) {
    galleryLayoutMode.value = mode
    persistLayoutMode(mode)
  }

  return { columnChoice, setColumnChoice, galleryLayoutMode, setGalleryLayoutMode }
})
