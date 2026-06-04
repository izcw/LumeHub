<template>
  <!-- 主导航与画廊分类条拆分为子组件；外壳仅负责滚动锁与 Esc 优先级 -->
  <PrimaryNavBar v-model:menu-open="primaryNavOpen" />
  <GalleryNavBar ref="galleryNavRef" v-model:search-open="searchOpen" />
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { lockBodyScroll, unlockBodyScroll } from '@/utils/bodyScrollLock'
import PrimaryNavBar from './nav/PrimaryNavBar.vue'
import GalleryNavBar from './nav/GalleryNavBar.vue'

const primaryNavOpen = ref(false)
const searchOpen = ref(false)
const galleryNavRef = ref<InstanceType<typeof GalleryNavBar> | null>(null)

/** 打开抽屉/全屏搜索时锁定背景滚动（含移动端） */
let overlayScrollLocked = false

function setOverlayScrollLock(locked: boolean) {
  if (locked && !overlayScrollLocked) {
    overlayScrollLocked = true
    lockBodyScroll()
    return
  }
  if (!locked && overlayScrollLocked) {
    overlayScrollLocked = false
    unlockBodyScroll()
  }
}

watch([primaryNavOpen, searchOpen], ([nav, search]) => {
  setOverlayScrollLock(nav || search)
})

function onDocKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  if (primaryNavOpen.value) {
    e.preventDefault()
    primaryNavOpen.value = false
    return
  }
  if (galleryNavRef.value?.columnMenuOpen) {
    e.preventDefault()
    galleryNavRef.value.closeColumnMenu()
    return
  }
  if (searchOpen.value) {
    e.preventDefault()
    searchOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('keydown', onDocKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onDocKeydown)
  setOverlayScrollLock(false)
})
</script>
