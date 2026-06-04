/** 叠加层打开时锁定页面滚动（含 iOS）；支持多处同时加锁 */
let lockCount = 0
let savedScrollY = 0

export function lockBodyScroll() {
  if (typeof document === 'undefined') return
  if (lockCount === 0) {
    savedScrollY = window.scrollY
    document.documentElement.style.overflow = 'hidden'
    document.documentElement.style.overscrollBehavior = 'none'
    document.body.style.overflow = 'hidden'
    document.body.style.position = 'fixed'
    document.body.style.top = `-${savedScrollY}px`
    document.body.style.left = '0'
    document.body.style.right = '0'
    document.body.style.width = '100%'
  }
  lockCount += 1
}

export function unlockBodyScroll() {
  if (typeof document === 'undefined') return
  if (lockCount <= 0) return
  lockCount -= 1
  if (lockCount > 0) return
  document.documentElement.style.overflow = ''
  document.documentElement.style.overscrollBehavior = ''
  document.body.style.overflow = ''
  document.body.style.position = ''
  document.body.style.top = ''
  document.body.style.left = ''
  document.body.style.right = ''
  document.body.style.width = ''
  window.scrollTo(0, savedScrollY)
}
