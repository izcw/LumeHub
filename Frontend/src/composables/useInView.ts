import { onMounted, onUnmounted, ref, unref, watch, type Ref } from 'vue'

export type UseInViewOptions = {
  /** 滚动容器；回收站等 overflow 区域需传入，否则懒加载无效 */
  root?: Ref<Element | null | undefined>
  rootMargin?: string
  /** 进入视口后是否保持可见（默认 true，避免滚出再卸载） */
  once?: boolean
}

export function useInView(options: UseInViewOptions = {}) {
  const target = ref<HTMLElement | null>(null)
  const inView = ref(false)
  let observer: IntersectionObserver | null = null

  function cleanup() {
    observer?.disconnect()
    observer = null
  }

  function setup() {
    cleanup()
    const el = target.value
    if (!el) return

    observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) {
          inView.value = true
          if (options.once !== false) cleanup()
        } else if (options.once === false) {
          inView.value = false
        }
      },
      {
        root: unref(options.root) ?? null,
        rootMargin: options.rootMargin ?? '240px 0px',
        threshold: 0.01,
      },
    )
    observer.observe(el)
  }

  onMounted(setup)

  watch(
    () => unref(options.root),
    () => {
      if (target.value) setup()
    },
  )

  onUnmounted(cleanup)

  return { target, inView }
}
