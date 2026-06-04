// 全局 RAF 调度（多个卡片共享一条渲染循环）
// 目的：避免每个组件各自 requestAnimationFrame 造成性能浪费

import { onMounted, onUnmounted } from 'vue'

type Callback = () => void

const callbacks = new Set<Callback>()
let rafId: number | null = null

/** 后台标签页暂停，避免大量卡片 3D 回调空转 */
let documentVisible = typeof document !== 'undefined' ? document.visibilityState !== 'hidden' : true
if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    documentVisible = document.visibilityState !== 'hidden'
  })
}

const loop = () => {
  if (documentVisible && callbacks.size > 0) {
    callbacks.forEach((cb) => cb())
  }
  rafId = requestAnimationFrame(loop)
}

export function useRafLoop(cb: Callback) {
    onMounted(() => {
        callbacks.add(cb)
        if (!rafId) rafId = requestAnimationFrame(loop)
    })

    onUnmounted(() => {
        callbacks.delete(cb)
        if (callbacks.size === 0 && rafId) {
            cancelAnimationFrame(rafId)
            rafId = null
        }
    })
}