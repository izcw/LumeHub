import { defineStore } from 'pinia'
import { computed, nextTick, ref } from 'vue'
import axios from 'axios'
import { parseApiErrorMessage } from '@/utils/apiError'
import { http } from '@/api/http'
import {
  fetchAuthMe,
  fetchAuthStatus,
  patchAuthMe,
  postAuthLogin,
  postAuthMeAvatar,
  deleteAuthMeAvatar,
  type AuthQrSessionStatus,
} from '@/api/authApi'
import type { PatchAuthMeBody } from '@/api/authApi'
import type { ApiAccountPublic } from '@/api/types'

const LS_TOKEN = 'lumehub_session_token'
const LS_TOKEN_EXP = 'lumehub_token_expires_at'

function readStoredSessionToken(): string {
  if (typeof localStorage === 'undefined') return ''
  return localStorage.getItem(LS_TOKEN)?.trim() ?? ''
}

export const useAuthStore = defineStore('auth', () => {
  const authenticated = ref(false)
  const authConfigured = ref(false)
  /** 是否已成功拉取过 /api/auth/status（与网络失败区分） */
  const authStatusLoaded = ref(false)
  /** 首次登录状态校验尚未结束（刷新后避免误判未登录） */
  const authBootstrapping = ref(true)
  /** 拉取登录状态时网络/服务端错误 */
  const authStatusError = ref('')
  /** 多账号模式（accounts.json 有用户） */
  const useAccounts = ref(false)
  const currentUser = ref<ApiAccountPublic | null>(null)

  /** 来自 GET /api/auth/status + accounts.json */
  const guestAvatarUrl = ref('')
  const loggedInFallbackAvatarUrl = ref('')

  /** 头像更新后用于破坏浏览器缓存 */
  const avatarRevision = ref(0)

  const loginModalOpen = ref(false)
  const loginModalMessage = ref('')

  const profileModalOpen = ref(false)
  /** 打开资料弹窗时希望切换到的标签（在弹窗内消费一次后清空） */
  const pendingProfileTab = ref<'me' | 'nav' | 'security' | 'accounts' | 'recycle' | null>(null)

  function clearClientSession() {
    localStorage.removeItem(LS_TOKEN)
    localStorage.removeItem(LS_TOKEN_EXP)
    delete http.defaults.headers.common['Authorization']
    currentUser.value = null
  }

  function isTokenExpiredClient(): boolean {
    if (typeof localStorage === 'undefined') return false
    const raw = localStorage.getItem(LS_TOKEN_EXP)?.trim()
    if (!raw) return false
    const ms = Date.parse(raw)
    if (Number.isNaN(ms)) return false
    return Date.now() >= ms
  }

  function hasStoredSession(): boolean {
    if (isTokenExpiredClient()) return false
    return readStoredSessionToken() !== ''
  }

  function applyStoredToken() {
    if (isTokenExpiredClient()) {
      clearClientSession()
      return
    }
    const t = localStorage.getItem(LS_TOKEN)?.trim()
    if (t) {
      http.defaults.headers.common['Authorization'] = `Bearer ${t}`
    } else {
      delete http.defaults.headers.common['Authorization']
    }
  }

  function applyLoginPayload(data: {
    token?: string
    expiresAt?: string
    user?: ApiAccountPublic
  }) {
    if (data.token) {
      localStorage.setItem(LS_TOKEN, data.token)
      if (data.expiresAt) {
        localStorage.setItem(LS_TOKEN_EXP, data.expiresAt)
      } else {
        localStorage.removeItem(LS_TOKEN_EXP)
      }
      http.defaults.headers.common['Authorization'] = `Bearer ${data.token}`
    }
    if (data.user) {
      currentUser.value = data.user
    }
  }

  let statusInflight: Promise<void> | null = null

  async function refreshStatus() {
    if (statusInflight) return statusInflight
    statusInflight = (async () => {
      authBootstrapping.value = true
      applyStoredToken()
      authStatusError.value = ''
      try {
        const data = await fetchAuthStatus()
        authStatusLoaded.value = true
        authConfigured.value = !!data.authConfigured
        useAccounts.value = !!data.useAccounts
        if (!data.authConfigured) {
          authenticated.value = false
          currentUser.value = null
          guestAvatarUrl.value = ''
          loggedInFallbackAvatarUrl.value = ''
          return
        }
        authenticated.value = !!data.authenticated
        guestAvatarUrl.value = (data.guestAvatarUrl ?? '').trim()
        loggedInFallbackAvatarUrl.value = (data.loggedInFallbackAvatarUrl ?? '').trim()
        if (authenticated.value) {
          await fetchMe()
        } else {
          currentUser.value = null
        }
      } catch (e) {
        authStatusLoaded.value = false
        authStatusError.value =
          axios.isAxiosError(e) && !e.response
            ? '无法连接服务器，请确认后端已在 5353 端口运行。'
            : '登录状态加载失败，请稍后重试。'
        useAccounts.value = false
        authenticated.value = false
        currentUser.value = null
        guestAvatarUrl.value = ''
        loggedInFallbackAvatarUrl.value = ''
      } finally {
        authBootstrapping.value = false
      }
    })().finally(() => {
      statusInflight = null
    })
    return statusInflight
  }

  async function fetchMe() {
    try {
      const data = await fetchAuthMe()
      if (data.user) {
        currentUser.value = data.user
      } else {
        currentUser.value = null
      }
      return data
    } catch {
      currentUser.value = null
      return null
    }
  }

  function openLoginModal(message?: string) {
    if (authenticated.value) return
    loginModalMessage.value = message?.trim() || '请登录后继续'
    profileModalOpen.value = false
    loginModalOpen.value = true
    if (authBootstrapping.value || !authStatusLoaded.value || authStatusError.value) {
      void refreshStatus()
    }
  }

  function closeLoginModal() {
    loginModalOpen.value = false
  }

  async function tryRecoverSessionFromToken() {
    if (authenticated.value || !hasStoredSession()) return
    applyStoredToken()
    const data = await fetchMe()
    if (data?.user) {
      authenticated.value = true
    }
  }

  async function openProfileModal(
    initialTab?: 'me' | 'nav' | 'security' | 'accounts' | 'recycle',
  ) {
    pendingProfileTab.value = initialTab ?? null
    if (authenticated.value) {
      loginModalOpen.value = false
      profileModalOpen.value = false
      await nextTick()
      profileModalOpen.value = true
      return
    }
    if (authBootstrapping.value || !authStatusLoaded.value) {
      await refreshStatus()
    }
    if (!authenticated.value) {
      await tryRecoverSessionFromToken()
    }
    if (!authenticated.value) {
      openLoginModal('请先登录后再打开账户设置')
      return
    }
    loginModalOpen.value = false
    profileModalOpen.value = false
    await nextTick()
    profileModalOpen.value = true
  }

  function takePendingProfileTab(): 'me' | 'nav' | 'security' | 'accounts' | 'recycle' | null {
    const t = pendingProfileTab.value
    pendingProfileTab.value = null
    return t
  }

  function closeProfileModal() {
    profileModalOpen.value = false
    pendingProfileTab.value = null
  }

  function withAvatarCacheBust(url: string): string {
    if (!url || avatarRevision.value <= 0) return url
    const sep = url.includes('?') ? '&' : '?'
    return `${url}${sep}v=${avatarRevision.value}`
  }

  function bumpAvatarRevision() {
    avatarRevision.value += 1
  }

  /** 仅为明确需要 token 的地址附加 access_token（默认不污染 /resource 短链） */
  function appendAccessToResourceUrl(url: string): string {
    if (!url.startsWith('/api/avatar/')) return url
    const t =
      (localStorage.getItem(LS_TOKEN)?.trim() ||
        (typeof http.defaults.headers.common['Authorization'] === 'string'
          ? http.defaults.headers.common['Authorization'].replace(/^Bearer\s+/i, '').trim()
          : '')) ?? ''
    if (!t) return url
    const sep = url.includes('?') ? '&' : '?'
    return `${url}${sep}access_token=${encodeURIComponent(t)}`
  }

  async function loginWithCredentials(email: string, password: string) {
    try {
      const trimmedEmail = email.trim()
      const body =
        useAccounts.value || trimmedEmail
          ? { email: trimmedEmail, password }
          : { password }
      const data = await postAuthLogin(body)
      applyLoginPayload(data)
      await refreshStatus()
      if (authConfigured.value && !authenticated.value) {
        throw new Error(data.error || '登录失败')
      }
    } catch (e) {
      if (axios.isAxiosError(e)) {
        const body = e.response?.data as { error?: string } | undefined
        const msg = body?.error?.trim()
        if (msg) throw new Error(msg)
        if (e.response?.status === 401) {
          throw new Error('邮箱或密码错误')
        }
        if (e.response?.status === 429) {
          throw new Error('登录失败次数过多，请稍后再试')
        }
      }
      throw e
    }
  }

  async function loginWithQrGrant(grant: AuthQrSessionStatus) {
    if (!grant.token) {
      throw new Error(grant.error || '二维码登录令牌缺失')
    }
    applyLoginPayload(grant)
    await refreshStatus()
    if (authConfigured.value && !authenticated.value) {
      throw new Error(grant.error || '二维码登录失败')
    }
  }

  async function updateProfile(patch: PatchAuthMeBody) {
    try {
      const updated = await patchAuthMe(patch)
      currentUser.value = updated
      return updated
    } catch (e) {
      throw new Error(parseApiErrorMessage(e, '保存失败'))
    }
  }

  async function uploadAvatar(file: File) {
    try {
      const updated = await postAuthMeAvatar(file)
      currentUser.value = updated
      bumpAvatarRevision()
      return updated
    } catch (e) {
      throw new Error(parseApiErrorMessage(e, '头像上传失败'))
    }
  }

  async function removeAvatar() {
    try {
      const updated = await deleteAuthMeAvatar()
      currentUser.value = updated
      bumpAvatarRevision()
      return updated
    } catch (e) {
      throw new Error(parseApiErrorMessage(e, '头像移除失败'))
    }
  }

  async function logout() {
    try {
      await http.post('/api/auth/logout')
    } catch {
      /* 忽略 */
    }
    clearClientSession()
    await refreshStatus()
  }

  const needsLoginForCurrentNav = computed(() => authConfigured.value && !authenticated.value)

  const canAccessPrivate = computed(() => !authConfigured.value || authenticated.value)

  const showsLoggedInNav = computed(
    () => authenticated.value || (authBootstrapping.value && hasStoredSession()),
  )

  function guestAvatarSrc(): string {
    const g = guestAvatarUrl.value.trim()
    if (!g) return ''
    if (g.startsWith('http://') || g.startsWith('https://')) return g
    const path = g.startsWith('/') ? g : `/${g}`
    return appendAccessToResourceUrl(path)
  }

  function resolvedAvatarUrl(): string {
    const u = currentUser.value?.avatar?.trim()
    if (u) {
      if (u.startsWith('http://') || u.startsWith('https://')) return withAvatarCacheBust(u)
      const path = u.startsWith('/') ? u : `/${u}`
      return withAvatarCacheBust(appendAccessToResourceUrl(path))
    }
    const fb = loggedInFallbackAvatarUrl.value.trim()
    if (fb) {
      if (fb.startsWith('http://') || fb.startsWith('https://')) return fb
      const path = fb.startsWith('/') ? fb : `/${fb}`
      return appendAccessToResourceUrl(path)
    }
    return ''
  }

  applyStoredToken()
  void refreshStatus()

  return {
    authenticated,
    authConfigured,
    authStatusLoaded,
    authStatusError,
    authBootstrapping,
    useAccounts,
    currentUser,
    guestAvatarUrl,
    loggedInFallbackAvatarUrl,
    loginModalOpen,
    loginModalMessage,
    profileModalOpen,
    needsLoginForCurrentNav,
    canAccessPrivate,
    showsLoggedInNav,
    hasStoredSession,
    refreshStatus,
    fetchMe,
    openLoginModal,
    closeLoginModal,
    openProfileModal,
    takePendingProfileTab,
    closeProfileModal,
    loginWithCredentials,
    loginWithQrGrant,
    updateProfile,
    uploadAvatar,
    removeAvatar,
    logout,
    appendAccessToResourceUrl,
    applyStoredToken,
    resolvedAvatarUrl,
    guestAvatarSrc,
  }
})
