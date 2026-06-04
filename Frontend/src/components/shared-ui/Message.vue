<template>
  <Teleport to="body">
    <Transition name="app-message-fade">
      <div
        v-if="visible"
        class="app-message"
        :class="`is-${type}`"
        :style="messageStyle"
        role="status"
        aria-live="polite"
      >
        {{ text }}
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useMessageStore } from '@/stores/message'
import { VIEWER_Z } from '@/components/viewers/shared/viewerLayers'

const store = useMessageStore()
const { visible, text, type } = storeToRefs(store)

/** 内联 z-index：Teleport 到 body 时 scoped v-bind 可能不生效 */
const messageStyle = computed(() => ({
  zIndex: VIEWER_Z.toast,
}))
</script>

<style scoped lang="scss">
.app-message {
  --message-fg: #1d1d1f;
  --message-border: rgba(255, 255, 255, 0.72);
  --message-bg-tint: rgba(255, 255, 255, 0.52);

  position: fixed;
  left: 50%;
  top: max(12px, env(safe-area-inset-top));
  transform: translateX(-50%);
  z-index: 10300;
  display: block;
  max-width: min(92vw, 640px);
  padding: 10px 15px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 520;
  line-height: 1.4;
  letter-spacing: 0.005em;
  color: var(--message-fg);
  font-family:
    -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'PingFang SC', 'Helvetica Neue', 'Segoe UI',
    sans-serif;
  text-wrap: pretty;
  border: 1px solid var(--message-border);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.88) 0%, rgba(255, 255, 255, 0.66) 100%),
    linear-gradient(90deg, var(--message-bg-tint) 0%, rgba(255, 255, 255, 0.4) 100%);
  backdrop-filter: blur(20px) saturate(1.22);
  -webkit-backdrop-filter: blur(20px) saturate(1.22);
  box-shadow:
    0 10px 24px rgba(0, 0, 0, 0.11),
    0 1px 2px rgba(0, 0, 0, 0.05),
    inset 0 1px 0 rgba(255, 255, 255, 0.86);

  &.is-success {
    --message-fg: #1f4d2e;
    --message-border: rgba(52, 199, 89, 0.28);
    --message-bg-tint: rgba(52, 199, 89, 0.09);
  }

  &.is-warning {
    --message-fg: #2f3640;
    --message-border: rgba(255, 159, 10, 0.26);
    --message-bg-tint: rgba(255, 159, 10, 0.08);
  }

  &.is-info {
    --message-fg: #3a3f45;
    --message-border: rgba(90, 99, 112, 0.28);
    --message-bg-tint: rgba(90, 99, 112, 0.08);
  }

  &.is-error {
    --message-fg: #b4232f;
    --message-border: rgba(255, 69, 58, 0.3);
    --message-bg-tint: rgba(255, 69, 58, 0.09);
  }

  &.is-primary {
    --message-fg: #111111;
    --message-border: rgba(0, 0, 0, 0.16);
    --message-bg-tint: rgba(0, 0, 0, 0.04);
  }
}

.app-message-fade-enter-active,
.app-message-fade-leave-active {
  transition:
    opacity 0.24s ease,
    transform 0.3s cubic-bezier(0.22, 1, 0.36, 1),
    filter 0.24s ease;
}

.app-message-fade-enter-from,
.app-message-fade-leave-to {
  opacity: 0;
  transform: translate(-50%, -10px) scale(0.98);
  filter: blur(1px);
}
</style>
