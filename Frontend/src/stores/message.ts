import { defineStore } from 'pinia'
import { ref } from 'vue'

export type MessageType = 'info' | 'success' | 'warning' | 'error'
export type MessageShowOptions = {
  type?: MessageType
  /** 显示时间（毫秒）；0 表示不自动关闭 */
  duration?: number
}

export const useMessageStore = defineStore('appMessage', () => {
  const visible = ref(false)
  const text = ref('')
  const type = ref<MessageType>('info')
  let timer: ReturnType<typeof setTimeout> | null = null

  function clearTimer() {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  function normalizeDuration(duration: number | undefined): number {
    if (typeof duration !== 'number' || !Number.isFinite(duration)) return 2200
    if (duration <= 0) return 0
    return Math.floor(duration)
  }

  function show(
    message: string,
    typeOrOptions: MessageType | MessageShowOptions = 'info',
    durationArg?: number,
  ) {
    const next = message.trim()
    if (!next) return
    const messageType =
      typeof typeOrOptions === 'string' ? typeOrOptions : (typeOrOptions.type ?? 'info')
    const duration =
      typeof typeOrOptions === 'string'
        ? normalizeDuration(durationArg)
        : normalizeDuration(typeOrOptions.duration)

    clearTimer()
    text.value = next
    type.value = messageType
    visible.value = true
    if (duration > 0) {
      timer = setTimeout(() => {
        visible.value = false
        timer = null
      }, duration)
    }
  }

  function close() {
    clearTimer()
    visible.value = false
  }

  return {
    visible,
    text,
    type,
    show,
    close,
  }
})
