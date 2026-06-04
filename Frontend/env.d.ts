/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module 'vuedraggable' {
  import type { DefineComponent } from 'vue'
  const draggable: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default draggable
}
