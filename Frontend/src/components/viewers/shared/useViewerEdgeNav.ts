import { ref } from 'vue'

export const VIEWER_NAV_EDGE_THRESHOLD = 150

export function useViewerEdgeNav() {
  const showPrevNav = ref(false)
  const showNextNav = ref(false)
  const showBottomBar = ref(false)

  function onViewerMouseMove(event: MouseEvent) {
    const root = event.currentTarget as HTMLElement
    const rect = root.getBoundingClientRect()
    const x = event.clientX - rect.left
    const y = event.clientY - rect.top
    showPrevNav.value = x <= VIEWER_NAV_EDGE_THRESHOLD
    showNextNav.value = rect.width - x <= VIEWER_NAV_EDGE_THRESHOLD
    showBottomBar.value = rect.height - y <= VIEWER_NAV_EDGE_THRESHOLD
  }

  function onViewerMouseLeave() {
    showPrevNav.value = false
    showNextNav.value = false
    showBottomBar.value = false
  }

  return {
    showPrevNav,
    showNextNav,
    showBottomBar,
    onViewerMouseMove,
    onViewerMouseLeave,
  }
}
