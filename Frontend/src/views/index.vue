<template>
  <GalleryView :folder-key="homeFolderKey" />
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import GalleryView from '@/views/GalleryView.vue'
import { useCategoryNavStore } from '@/stores/categoryNav'

const { homeFolderKey } = storeToRefs(useCategoryNavStore())
</script>
