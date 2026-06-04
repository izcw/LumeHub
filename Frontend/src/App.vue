<template>
  <router-view />
  <GlobalOverlays />
  <NavAddModal :z-index="3200" />
  <SettingsModal />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import GlobalOverlays from '@/components/modals/GlobalOverlays.vue'
import NavAddModal from '@/components/modals/NavAddModal.vue'
import SettingsModal from '@/components/modals/SettingsModal.vue'
import { useAuthStore } from '@/stores/auth'
import { useCategoryNavStore } from '@/stores/categoryNav'

onMounted(() => {
  const auth = useAuthStore()
  auth.applyStoredToken()
  void auth.refreshStatus()
  void useCategoryNavStore().fetchFromServer()
})
</script>

<style scoped lang="scss"></style>
