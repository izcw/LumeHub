<template>
  <Dialog
    :open="resolvedOpen"
    :title="resolvedTitle"
    :show-actions="resolvedShowActions"
    :body-padded="true"
    cancel-text="取消"
    :confirm-text="resolvedConfirmText"
    :confirm-disabled="resolvedConfirmDisabled"
    :width="resolvedWidth"
    :height="resolvedHeight"
    :z-index="resolvedZIndex"
    @cancel="onCancel"
    @confirm="onConfirm"
    @close="onClose"
  >
    <div class="nav-add__body">
      <template v-if="!resolvedShowDefaultForm">
        <slot />
      </template>
      <div v-else-if="!canManageLayout">
        <p class="nav-add__err">你没有「目录与布局」管理权限，无法在线添加导航或画廊。</p>
        <div class="nav-add__actions">
          <Button native-type="button" @click="onClose">关闭</Button>
        </div>
      </div>

      <form v-else @submit.prevent="onConfirm">
        <template v-if="resolvedMode === 'primary'">
          <label class="nav-add__label" for="nav-add-major">导航名称</label>
          <Input
            id="nav-add-major"
            v-model="majorNameModel"
            type="text"
            autocomplete="off"
            placeholder="输入导航名称"
          />
        </template>
        <template v-else>
          <label v-if="resolvedShowMajorSelect" class="nav-add__label" for="nav-add-major-select">所属导航</label>
          <Select
            v-if="resolvedShowMajorSelect"
            id="nav-add-major-select"
            v-model="majorIdModel"
            :options="resolvedMajorOptions"
            class="nav-add__major-select"
          />
          <p v-else-if="resolvedGalleryMajorName" class="nav-add__ctx">导航：{{ resolvedGalleryMajorName }}</p>
          <label class="nav-add__label" for="nav-add-subonly">画廊名称</label>
          <Input
            id="nav-add-subonly"
            v-model="galleryNameModel"
            type="text"
            autocomplete="off"
            placeholder="输入画廊名称"
          />
        </template>
        <label v-if="resolvedShowFolderKey" class="nav-add__label" for="nav-add-fk">目录标识</label>
        <Input
          v-if="resolvedShowFolderKey"
          id="nav-add-fk"
          v-model="folderKeyModel"
          type="text"
          class="nav-add__input--mono"
          autocomplete="off"
          placeholder="例如：jingxuan_9"
        />

        <Checkbox v-if="resolvedShowPublic" v-model="isPublicModel" class="nav-add__check">
          公开（未勾选则仅登录后可见）
        </Checkbox>

        <slot name="extra" />

      </form>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import axios from 'axios'
import { useNavAddModalStore } from '@/stores/navAddModal'
import { useAuthStore } from '@/stores/auth'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { useMessageStore } from '@/stores/message'
import { postCategoryMajor, postCategorySub } from '@/api/adminApi'
import { isReservedGalleryFolderKey } from '@/utils/gallerySearchFolder'
import Button from '@/components/shared-ui/Button.vue'
import Checkbox from '@/components/shared-ui/Checkbox.vue'
import Dialog from '@/components/shared-ui/Dialog.vue'
import Input from '@/components/shared-ui/Input.vue'
import Select from '@/components/shared-ui/Select.vue'

type DialogSize = string | number
type ModalMode = 'primary' | 'gallery'
type SelectOption = { label: string; value: string }
type NavAddConfirmPayload = {
  mode: ModalMode
  majorName: string
  galleryName: string
  folderKey: string
  isPublic: boolean
  majorId: string
}

const props = withDefaults(
  defineProps<{
    useStore?: boolean
    open?: boolean
    title?: string
    confirmText?: string
    confirmDisabled?: boolean
    width?: DialogSize
    height?: DialogSize
    zIndex?: number
    showActions?: boolean
    mode?: ModalMode
    showDefaultForm?: boolean
    showFolderKey?: boolean
    showPublic?: boolean
    majorName?: string
    galleryName?: string
    folderKeyValue?: string
    publicValue?: boolean
    galleryMajorName?: string
    showMajorSelect?: boolean
    majorOptions?: SelectOption[]
    majorId?: string
  }>(),
  {
    useStore: true,
    open: false,
    title: '',
    confirmText: '保存',
    confirmDisabled: false,
    width: '420px',
    height: '400px',
    zIndex: 2600,
    showActions: true,
    mode: 'primary',
    showDefaultForm: true,
    showFolderKey: true,
    showPublic: true,
    majorName: '',
    galleryName: '',
    folderKeyValue: '',
    publicValue: true,
    galleryMajorName: '',
    showMajorSelect: false,
    majorOptions: () => [],
    majorId: '',
  },
)

const emit = defineEmits<{
  confirm: [payload?: NavAddConfirmPayload]
  cancel: []
  close: []
  'update:majorName': [value: string]
  'update:galleryName': [value: string]
  'update:folderKeyValue': [value: string]
  'update:publicValue': [value: boolean]
  'update:majorId': [value: string]
}>()

const store = useNavAddModalStore()
const auth = useAuthStore()
const categoryNav = useCategoryNavStore()
const message = useMessageStore()
const { open: storeOpen, mode: storeMode, galleryMajorName: storeGalleryMajorName, galleryMajorId } =
  storeToRefs(store)

const storeTitle = computed(() => (storeMode.value === 'primary' ? '添加导航' : '添加画廊'))
const resolvedOpen = computed(() => (props.useStore ? storeOpen.value : props.open))
const resolvedTitle = computed(() => (props.useStore ? storeTitle.value : props.title))
const resolvedShowActions = computed(() => (props.useStore ? canManageLayout.value : props.showActions))
const resolvedConfirmText = computed(() =>
  props.useStore ? (submitting.value ? '创建中…' : '创建') : props.confirmText,
)
const resolvedConfirmDisabled = computed(() => (props.useStore ? submitting.value : props.confirmDisabled))
const resolvedWidth = computed(() => props.width)
const resolvedHeight = computed(() => props.height)
const resolvedZIndex = computed(() => props.zIndex)
const resolvedMode = computed<ModalMode>(() => (props.useStore ? storeMode.value : props.mode))
const resolvedShowDefaultForm = computed(() => props.useStore || props.showDefaultForm)
const resolvedShowFolderKey = computed(
  () => props.showFolderKey && resolvedMode.value !== 'primary',
)
const resolvedShowPublic = computed(() => props.showPublic)
const resolvedShowMajorSelect = computed(
  () =>
    !props.useStore &&
    resolvedMode.value === 'gallery' &&
    props.showMajorSelect &&
    props.majorOptions.length > 0,
)
const resolvedMajorOptions = computed(() => props.majorOptions)

const canManageLayout = computed(() =>
  !!(auth.currentUser?.permissions ?? []).includes('manage_layout'),
)

const majorName = ref('')
const subOnlyName = ref('')
const folderKey = ref('')
const isPublic = ref(true)
const selectedMajorId = ref('')
const submitting = ref(false)
const folderKeyPattern = /^[a-z0-9][a-z0-9_]{1,62}$/

const majorNameModel = computed({
  get: () => (props.useStore ? majorName.value : props.majorName),
  set: (value: string) => {
    if (props.useStore) majorName.value = value
    else emit('update:majorName', value)
  },
})
const galleryNameModel = computed({
  get: () => (props.useStore ? subOnlyName.value : props.galleryName),
  set: (value: string) => {
    if (props.useStore) subOnlyName.value = value
    else emit('update:galleryName', value)
  },
})
const folderKeyModel = computed({
  get: () => (props.useStore ? folderKey.value : props.folderKeyValue),
  set: (value: string) => {
    if (props.useStore) folderKey.value = value
    else emit('update:folderKeyValue', value)
  },
})
const isPublicModel = computed({
  get: () => (props.useStore ? isPublic.value : props.publicValue),
  set: (value: boolean) => {
    if (props.useStore) isPublic.value = value
    else emit('update:publicValue', value)
  },
})
const majorIdModel = computed({
  get: () => (props.useStore ? selectedMajorId.value : props.majorId),
  set: (value: string) => {
    if (props.useStore) selectedMajorId.value = value
    else emit('update:majorId', value)
  },
})
const resolvedGalleryMajorName = computed(() =>
  props.useStore ? storeGalleryMajorName.value : props.galleryMajorName,
)

function resetForm() {
  majorName.value = ''
  subOnlyName.value = ''
  folderKey.value = ''
  isPublic.value = true
  selectedMajorId.value = ''
  submitting.value = false
}

watch(storeOpen, (v) => {
  if (v) resetForm()
})

function close() {
  store.close()
}

function onCancel() {
  if (props.useStore) {
    close()
    return
  }
  emit('cancel')
}

function onClose() {
  if (props.useStore) {
    close()
    return
  }
  emit('close')
}

function messageFromAxios(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const d = e.response?.data
    if (typeof d === 'string' && d.trim()) return d.trim()
    if (d && typeof d === 'object' && 'error' in d && typeof (d as { error?: string }).error === 'string') {
      return (d as { error: string }).error
    }
    return e.message || '请求失败'
  }
  return e instanceof Error ? e.message : '请求失败'
}

async function onSubmit() {
  submitting.value = true
  try {
    if (resolvedMode.value === 'primary') {
      const mn = majorNameModel.value.trim()
      if (!mn) {
        message.show('请填写导航名称', 'warning')
        submitting.value = false
        return
      }
      const doc = await postCategoryMajor({
        majorName: mn,
        public: isPublicModel.value,
      })
      categoryNav.replaceDoc(doc)
    } else {
      const fallbackMajorId = Number.parseInt(majorIdModel.value, 10)
      const mid = galleryMajorId.value ?? (Number.isFinite(fallbackMajorId) ? fallbackMajorId : null)
      if (mid == null) {
        message.show('无法定位当前导航，请刷新后重试', 'error')
        submitting.value = false
        return
      }
      const sn = galleryNameModel.value.trim()
      if (!sn) {
        message.show('请填写画廊名称', 'warning')
        submitting.value = false
        return
      }
      const fk = folderKeyModel.value.trim().toLowerCase()
      if (!folderKeyPattern.test(fk)) {
        message.show('目录标识格式错误，请使用 2-63 位小写字母/数字/下划线', 'error')
        submitting.value = false
        return
      }
      if (isReservedGalleryFolderKey(fk)) {
        message.show('目录标识 search 为系统保留，请换一个名称', 'error')
        submitting.value = false
        return
      }
      const doc = await postCategorySub({
        majorId: mid,
        name: sn,
        folderKey: fk,
        public: isPublicModel.value,
      })
      categoryNav.replaceDoc(doc)
    }
    store.close()
  } catch (e: unknown) {
    message.show(messageFromAxios(e), 'error')
  } finally {
    submitting.value = false
  }
}

function onConfirm() {
  if (props.useStore) {
    void onSubmit()
    return
  }
  emit('confirm', {
    mode: resolvedMode.value,
    majorName: majorNameModel.value,
    galleryName: galleryNameModel.value,
    folderKey: folderKeyModel.value,
    isPublic: isPublicModel.value,
    majorId: majorIdModel.value,
  })
}
</script>

<style scoped lang="scss">
.nav-add__body {
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}

.nav-add__ctx {
  margin:1rem 0 0.75rem;
  font-size: 12px;
  color: #666;
  font-weight: 400;
}

.nav-add__label {
  display: block;
  margin: 0.65rem 0 0.35rem;
  font-size: 12px;
  font-weight: 600;
  color: #000;
}

.nav-add__check {
  margin-top: 0.65rem;
}

.nav-add__err {
  margin: 0.75rem 0 0;
  font-size: 12px;
  line-height: 1.45;
  color: #b42318;
}

.nav-add__actions {
  width: 100%;
  margin-top: 1.1rem;
}

.nav-add__actions--row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.nav-add__input--mono {
  :deep(input) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  }
}
</style>
