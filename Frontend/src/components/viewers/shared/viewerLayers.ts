import type { InjectionKey, Ref } from 'vue'

/** 大图查看器层级（与 viewer-shell / viewer-chrome 保持一致） */
export const VIEWER_Z = {
  mask: 9999,
  root: 10000,
  chrome: 10002,
  loading: 10003,
  /** 查看器内工具栏 tooltip / popconfirm 等浮层 */
  float: 10100,
  /** 查看器打开时其上的对话框（编辑 / 转移等） */
  dialog: 10200,
  /** 全局 toast，需高于查看器及其对话框 */
  toast: 10300,
} as const

export const FLOATING_UI_Z_INDEX_KEY: InjectionKey<Ref<number | undefined>> =
  Symbol('floatingUiZIndex')
