<template>
  <Dialog
    :open="open"
    title="转移到"
    :show-actions="true"
    cancel-text="取消"
    confirm-text="转移"
    :confirm-disabled="effectiveConfirmDisabled"
    :z-index="dialogZIndex"
    width="420px"
    height="400px"
    @cancel="emit('close')"
    @confirm="onConfirm"
    @close="emit('close')"
  >
    <p v-if="itemLabel" class="transfer-dialog-desc">将「{{ itemLabel }}」转移到：</p>
    <p v-else class="transfer-dialog-desc">选择目标画廊</p>
    <Select
      v-model="selectedFolderKey"
      :options="options"
      :menu-z-index="selectMenuZIndex"
      trigger-label="目标画廊"
      menu-aria-label="目标画廊选项"
    />
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Dialog from '@/components/shared-ui/Dialog.vue'
import Select from '@/components/shared-ui/Select.vue'

type TransferOption = { label: string; value: string; disabled?: boolean }

const props = withDefaults(
  defineProps<{
    open: boolean
    itemLabel?: string
    currentFolderKey?: string
    options: TransferOption[]
    confirmDisabled?: boolean
    zIndex?: number
  }>(),
  {
    zIndex: 2450,
  },
)

const dialogZIndex = computed(() => props.zIndex)
const selectMenuZIndex = computed(() => props.zIndex + 50)

const emit = defineEmits<{
  close: []
  confirm: [targetFolderKey: string]
}>()

const selectedFolderKey = ref('')

const isCurrentSelected = computed(() => {
  const current = props.currentFolderKey?.trim()
  if (!current) return false
  return selectedFolderKey.value === current
})

const effectiveConfirmDisabled = computed(
  () => props.confirmDisabled || isCurrentSelected.value || !selectedFolderKey.value.trim(),
)

watch(
  () => [props.open, props.currentFolderKey] as const,
  ([open, currentFolderKey]) => {
    if (!open) {
      selectedFolderKey.value = ''
      return
    }
    selectedFolderKey.value = currentFolderKey?.trim() ?? ''
  },
  { immediate: true },
)

function onConfirm() {
  const target = selectedFolderKey.value.trim()
  if (!target || effectiveConfirmDisabled.value) return
  emit('confirm', target)
}
</script>

<style scoped lang="scss">
.transfer-dialog-desc {
  margin: 0 0 12px;
  font-size: 13px;
  line-height: 1.5;
  color: #555;
}
</style>
