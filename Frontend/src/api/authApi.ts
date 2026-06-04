import { http } from '@/api/http'
import type { ApiAccountPublic, ApiAuthMeResponse, ApiAuthStatus } from '@/api/types'

export async function fetchAuthStatus(): Promise<ApiAuthStatus> {
  const { data } = await http.get<ApiAuthStatus>('/api/auth/status')
  return data
}

export async function fetchAuthMe(): Promise<ApiAuthMeResponse> {
  const { data } = await http.get<ApiAuthMeResponse>('/api/auth/me')
  return data
}

export async function postAuthLogin(body: { email?: string; password: string }) {
  const { data } = await http.post<{
    authenticated?: boolean
    authConfigured?: boolean
    token?: string
    expiresAt?: string
    user?: ApiAccountPublic
    error?: string
  }>('/api/auth/login', body)
  return data
}

export type AuthQrSessionResponse = {
  sessionId: string
  qrLoginUrl: string
  expiresAt: string
  pollIntervalMs?: number
}

export type AuthQrSessionStatus = {
  status: 'pending' | 'approved' | 'expired' | 'rejected'
  stage?: 'waiting_scan' | 'waiting_confirm'
  token?: string
  expiresAt?: string
  user?: ApiAccountPublic
  error?: string
}

export async function postAuthQrSession() {
  const { data } = await http.post<AuthQrSessionResponse>('/api/auth/qr/session')
  return data
}

export async function fetchAuthQrSessionStatus(sessionId: string) {
  const { data } = await http.get<AuthQrSessionStatus>(
    `/api/auth/qr/session/${encodeURIComponent(sessionId)}`,
  )
  return data
}

export type PasskeyRegisterOptions = {
  challenge: string
  rp: { id: string; name: string }
  user: { id: string; name: string; displayName: string }
  pubKeyCredParams: Array<{ type: 'public-key'; alg: number }>
  timeoutMs?: number
  attestation?: 'none' | 'direct' | 'indirect'
  authenticatorSelection?: {
    userVerification?: 'required' | 'preferred' | 'discouraged'
    residentKey?: 'required' | 'preferred' | 'discouraged'
  }
}

export async function postPasskeyRegisterOptions(label?: string) {
  const { data } = await http.post<PasskeyRegisterOptions>('/api/auth/passkey/register/options', {
    label: label?.trim() || '',
  })
  return data
}

export async function postPasskeyRegisterVerify(body: {
  credentialId: string
  publicKey: string
  algorithm: number
  clientDataJSON: string
  authenticatorData: string
  transports?: string[]
  label?: string
}) {
  const { data } = await http.post<{ ok?: boolean; credentialId?: string }>(
    '/api/auth/passkey/register/verify',
    body,
  )
  return data
}

export type PasskeyItem = {
  id: string
  label?: string
  displayId: string
  algorithm: number
  signCount: number
  transports?: string[]
  createdAt?: string
  lastUsedAt?: string
}

export async function fetchPasskeyList() {
  const { data } = await http.get<{ items?: PasskeyItem[] }>('/api/auth/passkey/list')
  return data.items ?? []
}

export type PatchAuthMeBody = {
  displayName?: string
  avatar?: string
  email?: string
  newPassword?: string
  currentPassword?: string
}

export async function patchAuthMe(body: PatchAuthMeBody) {
  const { data } = await http.patch<ApiAccountPublic>('/api/auth/me', body)
  return data
}

export async function postAuthMeAvatar(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  const { data } = await http.post<ApiAccountPublic>('/api/auth/me/avatar', fd, {
    maxBodyLength: Infinity,
    maxContentLength: Infinity,
  })
  return data
}

export async function deleteAuthMeAvatar() {
  const { data } = await http.delete<ApiAccountPublic>('/api/auth/me/avatar')
  return data
}
