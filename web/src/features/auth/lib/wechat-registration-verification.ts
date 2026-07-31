/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const VERIFICATION_STORAGE_KEY = 'wechat-registration-verification'
const RETURN_TO_STORAGE_KEY = 'wechat-registration-return-to'

export const WECHAT_REGISTRATION_RESET_CODES = new Set([
  'WECHAT_REGISTRATION_VERIFICATION_INVALID',
  'WECHAT_REGISTRATION_IDENTITY_BOUND',
])

export type WeChatRegistrationVerification = {
  token: string
  expiresAt: number
}

export type WeChatRegistrationVerificationState =
  | 'not-required'
  | 'unavailable'
  | 'pending'
  | 'verified'

type WeChatRegistrationVerificationResponse = {
  action: 'wechat_registration_verified'
  verification_token: string
  expires_at: number
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

export function parseWeChatRegistrationVerificationResponse(
  value: unknown
): WeChatRegistrationVerificationResponse | null {
  if (!isRecord(value)) return null
  if (value.action !== 'wechat_registration_verified') return null
  if (
    typeof value.verification_token !== 'string' ||
    !value.verification_token
  ) {
    return null
  }
  if (typeof value.expires_at !== 'number' || value.expires_at <= 0) {
    return null
  }
  return {
    action: 'wechat_registration_verified',
    verification_token: value.verification_token,
    expires_at: value.expires_at,
  }
}

export function resolveWeChatRegistrationVerificationState(
  required: boolean,
  available: boolean,
  verification: WeChatRegistrationVerification | null
): WeChatRegistrationVerificationState {
  if (!required) return 'not-required'
  if (!available) return 'unavailable'
  if (verification) return 'verified'
  return 'pending'
}

export function getWeChatRegistrationVerification(
  now = Date.now()
): WeChatRegistrationVerification | null {
  if (typeof window === 'undefined') return null
  try {
    const raw = window.sessionStorage.getItem(VERIFICATION_STORAGE_KEY)
    if (!raw) return null
    const parsed: unknown = JSON.parse(raw)
    if (!isRecord(parsed)) {
      clearWeChatRegistrationVerification()
      return null
    }
    const token = parsed.token
    const expiresAt = parsed.expiresAt
    if (
      typeof token !== 'string' ||
      !token ||
      typeof expiresAt !== 'number' ||
      expiresAt * 1000 <= now
    ) {
      clearWeChatRegistrationVerification()
      return null
    }
    return { token, expiresAt }
  } catch {
    clearWeChatRegistrationVerification()
    return null
  }
}

export function saveWeChatRegistrationVerification(
  verification: WeChatRegistrationVerification
): void {
  if (typeof window === 'undefined') return
  try {
    window.sessionStorage.setItem(
      VERIFICATION_STORAGE_KEY,
      JSON.stringify(verification)
    )
  } catch {
    /* Browser storage can be unavailable in restricted contexts. */
  }
}

export function clearWeChatRegistrationVerification(): void {
  if (typeof window === 'undefined') return
  try {
    window.sessionStorage.removeItem(VERIFICATION_STORAGE_KEY)
  } catch {
    /* Browser storage can be unavailable in restricted contexts. */
  }
}

export function saveWeChatRegistrationReturnTo(returnTo?: string): void {
  if (typeof window === 'undefined') return
  try {
    if (returnTo) {
      window.sessionStorage.setItem(RETURN_TO_STORAGE_KEY, returnTo)
    } else {
      window.sessionStorage.removeItem(RETURN_TO_STORAGE_KEY)
    }
  } catch {
    /* Browser storage can be unavailable in restricted contexts. */
  }
}

export function consumeWeChatRegistrationReturnTo(): string {
  if (typeof window === 'undefined') return ''
  try {
    const returnTo =
      window.sessionStorage.getItem(RETURN_TO_STORAGE_KEY)?.trim() ?? ''
    window.sessionStorage.removeItem(RETURN_TO_STORAGE_KEY)
    return returnTo
  } catch {
    return ''
  }
}
