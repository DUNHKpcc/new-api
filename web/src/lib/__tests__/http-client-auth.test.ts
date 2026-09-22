/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { AxiosHeaders } from 'axios'
import { afterEach, expect, it, vi } from 'vitest'

import { useAuthStore, type AuthBundle } from '@/stores/auth-store'

import { prepareAuthenticatedRequest } from '../http-client'

const bundle: AuthBundle = {
  access_token: 'current-token',
  token_type: 'Bearer',
  access_expires_at: Math.floor(Date.now() / 1000) + 30,
  user: { id: 1, username: 'admin', role: 10 },
  session: {
    sid: 'session-a',
    current: true,
    login_method: 'password',
    ip: '127.0.0.1',
    user_agent: 'test',
    created_at: 1,
    last_active_at: 1,
    expires_at: 2,
  },
}

afterEach(() => {
  useAuthStore.getState().auth.reset('idle')
  vi.restoreAllMocks()
})

it('refreshes an access token before an ordinary request when it is near expiry', async () => {
  useAuthStore.getState().auth.setBundle(bundle)
  const refreshHeaders = vi.fn(async () => ({
    Authorization: 'Bearer refreshed-token',
  }))
  const config = {
    headers: new AxiosHeaders(),
    url: '/api/log/',
  }

  await prepareAuthenticatedRequest(config, refreshHeaders)

  expect(refreshHeaders).toHaveBeenCalledOnce()
  expect(config.headers.get('Authorization')).toBe('Bearer refreshed-token')
})

it('keeps the current token without refreshing when it has sufficient lifetime', async () => {
  useAuthStore.getState().auth.setBundle({
    ...bundle,
    access_expires_at: Math.floor(Date.now() / 1000) + 120,
  })
  const refreshHeaders = vi.fn(async () => ({
    Authorization: 'Bearer refreshed-token',
  }))
  const config = {
    headers: new AxiosHeaders(),
    url: '/api/log/',
  }

  await prepareAuthenticatedRequest(config, refreshHeaders)

  expect(refreshHeaders).not.toHaveBeenCalled()
  expect(config.headers.get('Authorization')).toBe('Bearer current-token')
})

it('does not proactively refresh requests that explicitly skip auth refresh', async () => {
  useAuthStore.getState().auth.setBundle(bundle)
  const refreshHeaders = vi.fn(async () => ({
    Authorization: 'Bearer refreshed-token',
  }))
  const config = {
    headers: new AxiosHeaders(),
    url: '/api/auth/login',
    skipAuthRefresh: true,
  }

  await prepareAuthenticatedRequest(config, refreshHeaders)

  expect(refreshHeaders).not.toHaveBeenCalled()
  expect(config.headers.get('Authorization')).toBe('Bearer current-token')
})
