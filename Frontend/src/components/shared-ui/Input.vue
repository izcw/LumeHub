<template>
  <div class="ui-input-wrap" :class="{ 'has-password-toggle': isPasswordType }">
    <input
      ref="inputRef"
      :id="id"
      class="ui-input"
      :class="{ 'with-password-toggle': isPasswordType }"
      :type="resolvedType"
      :name="name"
      :autocomplete="autocomplete"
      :placeholder="placeholder"
      :value="modelValue"
      :disabled="disabled"
      :autofocus="autofocus"
      @input="onInput"
      @focus="onFocus"
      @keydown="onKeydown"
    />
    <button
      v-if="isPasswordType"
      type="button"
      class="ui-input__toggle"
      :aria-label="showPassword ? '隐藏密码' : '显示密码'"
      :disabled="disabled"
      @click="togglePassword"
    >
      <img
        class="ui-input__toggle-icon"
        :src="showPassword ? eyeIconSrc : eyeInvisibleIconSrc"
        alt=""
        aria-hidden="true"
      />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import eyeIconSrc from '@/assets/icon/eye.svg'
import eyeInvisibleIconSrc from '@/assets/icon/eye-invisible.svg'

const props = withDefaults(
  defineProps<{
    modelValue: string
    type?: string
    id?: string
    name?: string
    autocomplete?: string
    placeholder?: string
    disabled?: boolean
    autofocus?: boolean
  }>(),
  {
    type: 'text',
    id: undefined,
    name: undefined,
    autocomplete: undefined,
    placeholder: '',
    disabled: false,
    autofocus: false,
  },
)

const inputRef = ref<HTMLInputElement | null>(null)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  focus: [event: FocusEvent]
  keydown: [event: KeyboardEvent]
}>()

const showPassword = ref(false)
const isPasswordType = computed(() => props.type === 'password')
const resolvedType = computed(() => {
  if (!isPasswordType.value) return props.type
  return showPassword.value ? 'text' : 'password'
})

function onInput(event: Event) {
  const target = event.target as HTMLInputElement | null
  emit('update:modelValue', target?.value ?? '')
}

function onFocus(event: FocusEvent) {
  emit('focus', event)
}

function onKeydown(event: KeyboardEvent) {
  emit('keydown', event)
}

function focusInput() {
  inputRef.value?.focus()
}

defineExpose({ focus: focusInput })

function togglePassword() {
  showPassword.value = !showPassword.value
}
</script>

<style scoped lang="scss">
.ui-input-wrap {
  position: relative;
  width: 100%;
}

.ui-input {
  width: 100%;
  box-sizing: border-box;
  padding: 10px 12px;
  height: 36px;
  font-size: 12px;
  line-height: 36px;
  border: 1px solid #ccc;
  outline: none;
  color: #111111;
  background: #fff;
  border-radius: 6px;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;

  &::placeholder {
    color: #c3cad8;
  }
}

.ui-input.with-password-toggle {
  padding-right: 40px;
}

.ui-input__toggle {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  border: 0;
  background: transparent;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;

  &:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.ui-input__toggle-icon {
  width: 16px;
  height: 16px;
  display: block;
  object-fit: contain;
  opacity: 0.66;
}
</style>
