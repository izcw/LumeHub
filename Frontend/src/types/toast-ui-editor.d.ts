declare module '@toast-ui/editor' {
  type EditorOptions = {
    el: HTMLElement
    height?: string
    initialEditType?: 'markdown' | 'wysiwyg'
    previewStyle?: 'tab' | 'vertical'
    initialValue?: string
    usageStatistics?: boolean
    hideModeSwitch?: boolean
    toolbarItems?: string[][]
  }

  export class Editor {
    constructor(options: EditorOptions)
    on(event: 'change', listener: () => void): void
    getMarkdown(): string
    setMarkdown(markdown: string, cursorToEnd?: boolean): void
    destroy(): void
  }

}

declare module '@toast-ui/editor/viewer' {
  type ViewerOptions = {
    el: HTMLElement
    initialValue?: string
    usageStatistics?: boolean
  }

  export default class Viewer {
    constructor(options: ViewerOptions)
    setMarkdown(markdown: string): void
    destroy(): void
  }
}
