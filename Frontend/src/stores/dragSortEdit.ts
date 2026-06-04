import { defineStore } from 'pinia'
import { ref } from 'vue'

/** 拖拽编辑模式：开启后显示拖拽手柄并允许通过手柄拖拽排序。 */
export const useDragSortEditStore = defineStore('dragSortEdit', () => {
  const enabled = ref(false)

  function setEnabled(next: boolean) {
    enabled.value = next
  }

  function toggle() {
    enabled.value = !enabled.value
  }

  return { enabled, setEnabled, toggle }
})
