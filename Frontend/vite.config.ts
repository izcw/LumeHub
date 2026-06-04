import path from 'node:path'
import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: process.env.VITE_DEV_API_PROXY ?? 'http://127.0.0.1:5353',
        changeOrigin: true,
      },
      '/resource': {
        target: process.env.VITE_DEV_API_PROXY ?? 'http://127.0.0.1:5353',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: fileURLToPath(new URL('../Backend/www', import.meta.url)),
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/js/[name]-[hash].js',
        chunkFileNames: 'assets/js/[name]-[hash].js',
        assetFileNames(assetInfo) {
          const names = assetInfo.names?.length ? assetInfo.names : [assetInfo.name ?? '']
          const label = names.find(Boolean) ?? ''
          const ext = path.extname(label.split('?')[0] ?? '').toLowerCase()
          if (
            ext === '.css' ||
            label.includes('type=style') ||
            (label.includes('.vue') && label.includes('lang.scss'))
          ) {
            return 'assets/css/[name]-[hash][extname]'
          }
          if (['.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.ico', '.avif'].includes(ext)) {
            return 'assets/images/[name]-[hash][extname]'
          }
          return 'assets/[name]-[hash][extname]'
        },
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('vue-router')) return 'vue-router'
          if (id.includes('pinia')) return 'pinia'
          if (id.includes('@vue/')) return 'vue-core'
          if (/[/\\]node_modules[/\\]vue[/\\]/.test(id)) return 'vue-core'
        },
      },
    },
    chunkSizeWarningLimit: 700,
  },
}))
