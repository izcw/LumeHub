<template>
  <GalleryView v-if="folderKey && routeAllowed" :folder-key="folderKey" />
  <div v-else-if="folderKey && !routeAllowed" class="empty">正在跳转…</div>
  <div v-else class="empty">无效的分类路径</div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import GalleryView from '@/views/GalleryView.vue'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { useAuthStore } from '@/stores/auth'
import { folderNavEntryVisible } from '@/utils/galleryAccess'

const route = useRoute()
const router = useRouter()
const categoryNav = useCategoryNavStore()
const auth = useAuthStore()
const { canAccessPrivate } = storeToRefs(auth)

const folderKey = computed(() => {
  const raw = route.params.folderKey
  return typeof raw === 'string' ? raw : Array.isArray(raw) ? (raw[0] ?? '') : ''
})

const routeAllowed = ref(true)

watch(
  [folderKey, canAccessPrivate, () => categoryNav.loaded],
  async ([fk, authed]) => {
    if (!fk) {
      routeAllowed.value = true
      return
    }
    if (!categoryNav.loaded) {
      await categoryNav.fetchFromServer()
    }
    const visible = folderNavEntryVisible(categoryNav.doc, fk, authed)
    routeAllowed.value = visible
    if (!visible) {
      await router.replace('/')
    }
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  color: #999;
  font-size: 15px;
}
</style>
