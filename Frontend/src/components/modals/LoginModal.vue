<template>
  <Dialog
    :open="open"
    :show-close="true"
    :show-actions="false"
    :body-padded="true"
    @close="close"
    width="380px"
    height="420px"
    title="LumeHub · 登录"
  >
    <div class="login-content">
      <div v-if="authStatusError" class="panel-box">
        <p class="disabled-text">{{ authStatusError }}</p>
        <Button
          class="login-submit-btn"
          native-type="button"
          :disabled="statusRetrying"
          @click="retryAuthStatus"
        >
          {{ statusRetrying ? '连接中…' : '重试' }}
        </Button>
      </div>

      <div v-else-if="authStatusLoaded && !authConfigured" class="panel-box">
        <p class="disabled-text">
          当前服务器未启用登录（无 accounts.json 用户且无 LUMEHUB_PASSWORD）。
        </p>
        <Button class="login-submit-btn" native-type="button" @click="close">知道了</Button>
      </div>

      <div v-else-if="!authStatusLoaded" class="panel-box">
        <p class="disabled-text">正在检查登录配置…</p>
      </div>

      <form v-else class="panel-box login-panel" autocomplete="on" @submit.prevent="onSubmit">
        <template v-if="!qrMode">
          <template v-if="useAccounts">
            <Input
              id="auth-login-user"
              v-model="email"
              type="email"
              name="username"
              class="login-input"
              autocomplete="username"
              placeholder="请输入邮箱"
              @keydown.enter="onLoginFieldEnter"
            />
          </template>
          <Input
            id="auth-login-pw"
            v-model="password"
            type="password"
            name="password"
            class="login-input"
            autocomplete="current-password"
            placeholder="请输入密码"
            @keydown.enter="onLoginFieldEnter"
          />
          <div class="form-actions">
            <Button
              native-type="submit"
              class="login-submit-btn"
              width="100%"
              :disabled="submitting"
            >
              {{ submitting ? '登录中...' : '登录' }}
            </Button>
          </div>
        </template>
        <template v-else>
          <div class="qr-login-wrap">
            <div v-if="qrLoading" class="qr-placeholder">二维码加载中...</div>
            <div v-else-if="qrLoginUrl" class="qr-code-stage" :class="{ 'is-expired': qrExpired }">
              <QrcodeVue :value="qrLoginUrl" :size="110" level="M" render-as="svg" />
              <div
                v-if="qrExpired"
                class="qr-refresh-mask"
                role="button"
                tabindex="0"
                aria-label="刷新二维码"
                title="二维码已过期，点击刷新"
                @click="onManualRefreshQr"
                @keydown.enter.prevent="onManualRefreshQr"
                @keydown.space.prevent="onManualRefreshQr"
              >
                <img :src="refreshIcon" alt="" aria-hidden="true" class="qr-refresh-icon" />
              </div>
            </div>
            <div v-else class="qr-placeholder">二维码暂不可用</div>
            <p class="qr-login-hint">请使用手机"系统相机"扫码</p>
            <p class="qr-login-expire">
              <template v-if="qrLoading">正在创建一次性二维码...</template>
              <template v-else-if="qrExpired">二维码已过期，请点击二维码中心刷新</template>
              <template v-else-if="qrStage === 'waiting_confirm'"
                >已扫码，请在手机上完成通行证确认</template
              >
              <template v-else>二维码将在 {{ qrCountdown }}s 后刷新</template>
            </p>
          </div>
        </template>
        <div v-if="!qrMode" class="footer-links" aria-hidden="true">
          <span class="footer-link footer-link--muted">账号常见问题</span>
          <div class="footer-links-extra">
            <span class="footer-link">立即注册</span>
            <span class="footer-divider"></span>
            <span class="footer-link">忘记密码</span>
          </div>
        </div>
      </form>
      <div
        v-if="authConfigured"
        class="qr-corner-toggle"
        role="button"
        tabindex="0"
        :aria-label="qrMode ? '切换到邮箱密码登录' : '切换到二维码登录'"
        :title="qrMode ? '切换到邮箱密码登录' : '切换到二维码登录'"
        @click="toggleQrMode"
        @keydown.enter.prevent="toggleQrMode"
        @keydown.space.prevent="toggleQrMode"
      >
        <img
          :src="qrMode ? emailIcon : qrCodeIcon"
          alt=""
          aria-hidden="true"
          class="qr-corner-icon"
        />
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import QrcodeVue from 'qrcode.vue'
import Button from '@/components/shared-ui/Button.vue'
import Dialog from '@/components/shared-ui/Dialog.vue'
import Input from '@/components/shared-ui/Input.vue'
import qrCodeIcon from '@/assets/icon/QRcode.svg'
import emailIcon from '@/assets/icon/email.svg'
import refreshIcon from '@/assets/icon/refresh.svg'
import { fetchAuthQrSessionStatus, postAuthQrSession } from '@/api/authApi'
import { useAuthStore } from '@/stores/auth'
import { useMessageStore } from '@/stores/message'

const auth = useAuthStore()
const appMessage = useMessageStore()
const {
  loginModalOpen: open,
  authConfigured,
  authStatusLoaded,
  authStatusError,
  authenticated,
  useAccounts,
} = storeToRefs(auth)

const statusRetrying = ref(false)

const email = ref('')
const password = ref('')
const submitting = ref(false)
const qrMode = ref(false)
const qrLoading = ref(false)
const qrLoginUrl = ref('')
const qrSessionId = ref('')
const qrExpireAtMs = ref(0)
const qrCountdown = ref(0)
const qrPollingIntervalMs = ref(2000)
const qrStage = ref<'waiting_scan' | 'waiting_confirm'>('waiting_scan')
const qrExpired = ref(false)

let qrPollTimer: number | null = null
let qrCountdownTimer: number | null = null
let qrPollingRequestInFlight = false
let qrRefreshing = false

watch(open, (v) => {
  if (v) {
    email.value = ''
    password.value = ''
    submitting.value = false
    qrMode.value = false
    clearQrRuntime()
    void prefillCredential()
    return
  }
  clearQrRuntime()
})

watch(authenticated, (authed) => {
  if (authed && open.value) {
    close()
  }
})

watch(qrMode, (enabled) => {
  if (!open.value) return
  if (enabled) {
    void startQrLogin()
    return
  }
  clearQrRuntime()
})

onBeforeUnmount(() => {
  clearQrRuntime()
})

function close() {
  clearQrRuntime()
  auth.closeLoginModal()
}

async function retryAuthStatus() {
  statusRetrying.value = true
  try {
    await auth.refreshStatus()
  } finally {
    statusRetrying.value = false
  }
}

type PasswordCredentialLike = Credential & {
  id: string
  password: string
}

async function saveCredential() {
  if (!useAccounts.value) return
  if (typeof window === 'undefined' || !window.isSecureContext) return
  if (!('PasswordCredential' in window) || !navigator.credentials?.store) return

  const username = email.value.trim()
  const pwd = password.value.trim()
  if (!username || !pwd) return

  try {
    const PasswordCredentialCtor = window.PasswordCredential as unknown as new (data: {
      id: string
      password: string
      name?: string
    }) => Credential
    const credential = new PasswordCredentialCtor({
      id: username,
      password: pwd,
      name: username,
    })
    await navigator.credentials.store(credential)
  } catch {
    // 浏览器不支持或用户拒绝保存时静默跳过
  }
}

async function prefillCredential() {
  if (!useAccounts.value) return
  if (typeof window === 'undefined' || !window.isSecureContext) return
  if (!navigator.credentials?.get) return

  try {
    const credential = (await navigator.credentials.get({
      password: true,
      mediation: 'optional',
    } as CredentialRequestOptions)) as PasswordCredentialLike | null
    if (!credential) return
    if (!email.value) email.value = credential.id ?? ''
    if (!password.value) password.value = credential.password ?? ''
  } catch {
    // 某些浏览器会禁用该能力，保持无感降级
  }
}

function onLoginFieldEnter(ev: KeyboardEvent) {
  if (ev.key !== 'Enter' || ev.isComposing || ev.repeat) return
  ev.preventDefault()
  void onSubmit()
}

async function onSubmit() {
  if (qrMode.value) return
  if (!password.value.trim()) {
    showError('请输入密码')
    return
  }
  if (useAccounts.value && !email.value.trim()) {
    showError('请输入邮箱')
    return
  }
  submitting.value = true
  try {
    await auth.loginWithCredentials(email.value, password.value)
    await saveCredential()
    close()
  } catch (e: unknown) {
    showError(e instanceof Error ? e.message : '登录失败')
  } finally {
    submitting.value = false
  }
}

function toggleQrMode() {
  qrMode.value = !qrMode.value
}

function clearQrRuntime() {
  if (qrPollTimer !== null) {
    window.clearInterval(qrPollTimer)
    qrPollTimer = null
  }
  if (qrCountdownTimer !== null) {
    window.clearInterval(qrCountdownTimer)
    qrCountdownTimer = null
  }
  qrPollingRequestInFlight = false
  qrRefreshing = false
  qrLoading.value = false
  qrLoginUrl.value = ''
  qrSessionId.value = ''
  qrExpireAtMs.value = 0
  qrCountdown.value = 0
  qrStage.value = 'waiting_scan'
  qrExpired.value = false
}

async function startQrLogin() {
  clearQrRuntime()
  qrLoading.value = true
  await issueQrSession()
}

async function issueQrSession() {
  try {
    const data = await postAuthQrSession()
    qrSessionId.value = data.sessionId?.trim() ?? ''
    qrLoginUrl.value = data.qrLoginUrl?.trim() ?? ''
    qrExpireAtMs.value = Date.parse(data.expiresAt ?? '')
    qrPollingIntervalMs.value = Math.max(1000, Math.min(data.pollIntervalMs ?? 2000, 5000))
    qrStage.value = 'waiting_scan'
    qrExpired.value = false
    if (!qrSessionId.value || !qrLoginUrl.value || Number.isNaN(qrExpireAtMs.value)) {
      throw new Error('二维码会话参数不完整')
    }
    qrLoading.value = false
    qrRefreshing = false
    startQrCountdown()
    startQrPolling()
  } catch (e: unknown) {
    clearQrRuntime()
    qrMode.value = false
    showError(e instanceof Error ? e.message : '二维码登录暂不可用')
  }
}

function startQrCountdown() {
  if (qrCountdownTimer !== null) {
    window.clearInterval(qrCountdownTimer)
    qrCountdownTimer = null
  }
  updateQrCountdown()
  qrCountdownTimer = window.setInterval(() => {
    updateQrCountdown()
  }, 1000)
}

function updateQrCountdown() {
  const left = Math.ceil((qrExpireAtMs.value - Date.now()) / 1000)
  qrCountdown.value = Math.max(0, left)
  if (qrCountdown.value === 0 && qrMode.value && !qrRefreshing && !qrExpired.value) {
    qrExpired.value = true
    if (qrPollTimer !== null) {
      window.clearInterval(qrPollTimer)
      qrPollTimer = null
    }
  }
}

function startQrPolling() {
  if (qrPollTimer !== null) {
    window.clearInterval(qrPollTimer)
    qrPollTimer = null
  }
  void pollQrSessionStatus()
  qrPollTimer = window.setInterval(() => {
    void pollQrSessionStatus()
  }, qrPollingIntervalMs.value)
}

async function pollQrSessionStatus() {
  if (!qrMode.value || !qrSessionId.value || qrPollingRequestInFlight) return
  qrPollingRequestInFlight = true
  try {
    const status = await fetchAuthQrSessionStatus(qrSessionId.value)
    if (status.status === 'pending') {
      qrStage.value = status.stage === 'waiting_confirm' ? 'waiting_confirm' : 'waiting_scan'
      return
    }
    if (status.status === 'approved') {
      clearQrRuntime()
      await auth.loginWithQrGrant(status)
      close()
      return
    }
    if (status.status === 'expired') {
      qrExpired.value = true
      if (qrPollTimer !== null) {
        window.clearInterval(qrPollTimer)
        qrPollTimer = null
      }
      return
    }
    if (status.status === 'rejected') {
      throw new Error(status.error || '二维码登录已拒绝')
    }
  } catch (e: unknown) {
    clearQrRuntime()
    qrMode.value = false
    showError(e instanceof Error ? e.message : '二维码登录失败')
  } finally {
    qrPollingRequestInFlight = false
  }
}

async function onManualRefreshQr() {
  if (qrLoading.value || !qrExpired.value) return
  qrLoading.value = true
  qrRefreshing = false
  await issueQrSession()
}

function showError(text: string) {
  appMessage.show(text, {
    type: 'error',
    duration: 2600,
  })
}
</script>

<style scoped lang="scss">
.login-content {
  width: 100%;
  height: 100%;
  margin: 0 auto;
  padding:30px 30px 0 30px;
  padding-bottom: 30px;
  box-sizing: border-box;
  /* display: flex;
  align-items: center;
  justify-content: center; */
  
}

:deep(.dialog-body) {
  position: relative;
  overflow: hidden;
}

.panel-box {
  padding: 1.25rem 0 0;
}

.login-panel {
  position: relative;
  min-height: 260px;
}

.disabled-text {
  margin: 0 0 14px;
  font-size: 13px;
  line-height: 1.55;
  color: #475569;
}

.login-input {
  margin: 0 0 14px;
}

.form-actions {
  margin-top: 1.5rem;
}

.login-submit-btn {
  letter-spacing: 0;
}

.qr-login-wrap {
  min-height: 202px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.qr-code-stage {
  position: relative;
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: filter 0.2s ease;

  &.is-expired {
    filter: grayscale(1) brightness(0.9);
  }
}

.qr-placeholder {
  width: 120px;
  height: 120px;
  border: 1px dashed #eee;
  color: #9ca3af;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-refresh-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.8);
  cursor: pointer;
}

.qr-refresh-icon {
  width: 35px;
  height: 35px;
  opacity: 0.9;
}

.qr-login-hint {
  margin: 0;
  font-size: 12px;
  line-height: 1.4;
  color: #9ca3af;
  text-align: center;
}

.qr-login-expire {
  margin: 0;
  font-size: 12px;
  line-height: 1.4;
  color: #9ca3af;
}

.footer-links {
  margin-top: 54px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.footer-link {
  font-size: 12px;
  color: #000000;
  line-height: 1.2;
  white-space: nowrap;
}

.footer-link--muted {
  color: #9ca3af;
}

.footer-links-extra {
  display: flex;
  align-items: center;
}

.footer-divider {
  width: 1px;
  height: 14px;
  background: #eee;
  margin: 0 0.5rem;
  display: inline-block;
}

/* 1. 容器：定位在右下角，并匹配父容器的圆角 */
.qr-corner-toggle {
  --corner-size: 40px; /* 卷角区域大小 */
  --qr-padding: 10px 6px 6px 10px; /* 二维码的留白间距 */

  position: absolute;
  bottom: 0;
  right: 0;
  width: var(--corner-size);
  height: var(--corner-size);
  cursor: pointer;
  z-index: 100;
  outline: none;

  /* 重要：如果你的卡片有圆角，这里要对应，否则会超出 */
  border-bottom-right-radius: 8px;
  overflow: hidden;
  transition: opacity 0.2s ease;
}

/* 2. 二维码图标层：负责显示图片并留白 */
.qr-corner-icon {
  width: 100%;
  height: 100%;
  display: block;
  box-sizing: border-box;

  /* 通过 padding 实现留白效果 */
  padding: var(--qr-padding);
  background-color: #fff;

  /* 只保留右下对角线区域 */
  clip-path: polygon(100% 0, 100% 100%, 0 100%);

  /* 确保二维码图片在留白区域内完整显示 */
  object-fit: contain;
}

/* 3. 卷角效果层：模拟折叠的纸张 */
.qr-corner-toggle::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 2;

  /* 只保留左上对角线区域 */
  clip-path: polygon(0 0, 100% 0, 0 100%);

  /* 
     精细化的渐变：
     - 从纯白到浅灰，模拟纸张背面
     - 在 50% 处增加一个稍深的灰色 (#ccc) 作为折痕
     - 在折痕后增加极短的透明阴影，让它看起来是“浮”在二维码上的
  */
  background: linear-gradient(
    135deg,
    #ffffff 0%,
    #fcfcfc 35%,
    #e8e8e8 48%,
    #d0d0d0 50%,
    rgba(0, 0, 0, 0.1) 52%,
    transparent 60%
  );

  pointer-events: none;
}

/* 4. 悬停状态微调 */
.qr-corner-toggle:hover {
  opacity: 0.9;
}

/* 5. 针对图片本身可能有的底色处理 */
.qr-corner-icon img {
  width: 100%;
  height: 100%;
}

@media (max-width: 719px) {
  .login-content {
    width: 100%;
    padding: 50px 24px 40px;
  }

  .footer-links {
    margin-top: 120px;
  }

  .panel-box {
    padding-top: 40px;
  }
}
</style>
