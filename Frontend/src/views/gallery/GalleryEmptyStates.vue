<template>
  <div v-if="loading" class="empty-state">
    <span>加载中…</span>
  </div>
  <div v-else-if="accessBlocked" class="empty-state empty-state--locked">
    <p class="empty-state__message">{{ accessBlockedMessage }}</p>
    <div v-if="accessBlockedMode === 'password'" class="empty-state__password-form">
      <Input
        ref="passwordInputRef"
        :model-value="passwordInput"
        type="password"
        placeholder="输入查看密码"
        @update:model-value="emit('update:passwordInput', $event)"
        @keydown.enter.prevent="emit('password-submit')"
      />
      <p v-if="passwordError" class="empty-state__password-error">{{ passwordError }}</p>
      <Button native-type="button" @click="emit('password-submit')">确定</Button>
    </div>
    <Button v-else native-type="button" @click="emit('blocked-action')">
      {{ accessBlockedActionText }}
    </Button>
  </div>
  <div v-else-if="showAddGalleryEmpty" class="empty-state empty-state--add-gallery">
    <Button native-type="button" @click="emit('add-gallery')">添加画廊</Button>
  </div>
  <div v-else-if="loadError" class="empty-state">
    <span>{{ loadError }}</span>
  </div>
  <div v-else-if="navLoadError" class="empty-state">
    <span>{{ navLoadError }}</span>
  </div>
  <div v-else-if="searchNeedsCriteria" class="empty-state">
    <span>请通过顶部搜索设置关键词、分类、格式或标签</span>
  </div>
  <div
    v-else-if="isGlobalSearchView && (globalPoolLoading || loading)"
    class="empty-state"
  >
    <span>正在加载全部分类…</span>
  </div>
  <div v-else-if="searchSourceRowsEmpty" class="empty-state">
    <span>{{ isGlobalSearchView ? '暂无可搜索的资源' : '暂无图片' }}</span>
  </div>
  <div v-else-if="filteredRowsEmpty" class="empty-state">
    <span>没有符合当前筛选条件的图片</span>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import Button from '@/components/shared-ui/Button.vue'
import Input from '@/components/shared-ui/Input.vue'

const props = defineProps<{
  loading: boolean
  accessBlocked: boolean
  accessBlockedMode: 'auth' | 'password'
  accessBlockedMessage: string
  accessBlockedActionText: string
  passwordInput: string
  passwordError: string
  showAddGalleryEmpty: boolean
  loadError: string
  navLoadError?: string
  isGlobalSearchView?: boolean
  searchNeedsCriteria?: boolean
  searchScope: string
  globalPoolLoading: boolean
  searchSourceRowsEmpty: boolean
  filteredRowsEmpty: boolean
}>()

const emit = defineEmits<{
  'blocked-action': []
  'add-gallery': []
  'update:passwordInput': [value: string]
  'password-submit': []
}>()

const passwordInputRef = ref<{ focus?: () => void } | null>(null)

watch(
  () => props.accessBlocked && props.accessBlockedMode === 'password',
  (showForm) => {
    if (!showForm) return
    nextTick(() => passwordInputRef.value?.focus?.())
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.empty-state--locked {
  flex-direction: column;
  gap: 1rem;
  text-align: center;
}

.empty-state--add-gallery {
  flex-direction: column;
}

.empty-state__message {
  margin: 0;
  color: #666;
}

.empty-state__password-form {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
  width: min(320px, 100%);
}

.empty-state__password-error {
  margin: 0;
  font-size: 12px;
  color: #c53030;
  text-align: left;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: #bbb;
  font-size: 15px;
}
</style>
