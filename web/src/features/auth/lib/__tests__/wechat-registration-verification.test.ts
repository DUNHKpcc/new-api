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

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getWeChatRegistrationVerification,
  parseWeChatRegistrationVerificationResponse,
  resolveWeChatRegistrationVerificationState,
  saveWeChatRegistrationVerification,
} from '../wechat-registration-verification'

describe('WeChat registration verification', () => {
  test('accepts only complete verification callback payloads', () => {
    assert.deepEqual(
      parseWeChatRegistrationVerificationResponse({
        action: 'wechat_registration_verified',
        verification_token: 'proof-token',
        expires_at: 100,
      }),
      {
        action: 'wechat_registration_verified',
        verification_token: 'proof-token',
        expires_at: 100,
      }
    )
    assert.equal(
      parseWeChatRegistrationVerificationResponse({
        action: 'wechat_registration_verified',
        expires_at: 100,
      }),
      null
    )
    assert.equal(
      parseWeChatRegistrationVerificationResponse({
        action: 'login',
        verification_token: 'proof-token',
        expires_at: 100,
      }),
      null
    )
  })

  test('keeps required registration blocked until WeChat is available and verified', () => {
    assert.equal(
      resolveWeChatRegistrationVerificationState(false, false, null),
      'not-required'
    )
    assert.equal(
      resolveWeChatRegistrationVerificationState(true, false, null),
      'unavailable'
    )
    assert.equal(
      resolveWeChatRegistrationVerificationState(true, true, null),
      'pending'
    )
    assert.equal(
      resolveWeChatRegistrationVerificationState(true, true, {
        token: 'proof-token',
        expiresAt: 100,
      }),
      'verified'
    )
  })

  test('drops expired verification proofs from session storage', () => {
    const values = new Map<string, string>()
    const originalWindow = globalThis.window
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: {
        sessionStorage: {
          getItem: (key: string) => values.get(key) ?? null,
          setItem: (key: string, value: string) => values.set(key, value),
          removeItem: (key: string) => values.delete(key),
        },
      },
    })

    try {
      saveWeChatRegistrationVerification({
        token: 'proof-token',
        expiresAt: 120,
      })
      assert.deepEqual(getWeChatRegistrationVerification(100_000), {
        token: 'proof-token',
        expiresAt: 120,
      })
      assert.equal(getWeChatRegistrationVerification(120_000), null)
      assert.equal(values.size, 0)
    } finally {
      Object.defineProperty(globalThis, 'window', {
        configurable: true,
        value: originalWindow,
      })
    }
  })
})
