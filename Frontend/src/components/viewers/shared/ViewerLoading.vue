<template>
  <div v-show="visible" class="viewer-loading" aria-hidden="true">
    <span class="viewer-loading__dot" />
    <span class="viewer-loading__dot" />
    <span class="viewer-loading__dot" />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
}>()
</script>

<style scoped lang="scss">
@use './viewer-shell.scss';
</style>
