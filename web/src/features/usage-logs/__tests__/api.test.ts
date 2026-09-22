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
import {
  AxiosError,
  AxiosHeaders,
  type InternalAxiosRequestConfig,
} from 'axios'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore, type AuthBundle } from '@/stores/auth-store'

import { fetchWithPrivilegeFallback, getAllLogs } from '../api'

const originalAdapter = api.defaults.adapter

const bundle: AuthBundle = {
  access_token: 'access-token',
  token_type: 'Bearer',
  access_expires_at: Math.floor(Date.now() / 1000) + 600,
  user: { id: 1, username: 'operator', role: 10 },
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

function privilegeError(): AxiosError {
  const config = { headers: new AxiosHeaders() } as InternalAxiosRequestConfig
  return new AxiosError('Forbidden', 'ERR_BAD_REQUEST', undefined, undefined, {
    data: {
      success: false,
      code: 'AUTH_INSUFFICIENT_PRIVILEGE',
    },
    status: 403,
    statusText: 'Forbidden',
    headers: {},
    config,
  })
}

afterEach(() => {
  api.defaults.adapter = originalAdapter
  useAuthStore.getState().auth.reset('idle')
  vi.restoreAllMocks()
})

it('uses the canonical trailing-slash admin logs route', async () => {
  useAuthStore.getState().auth.setBundle(bundle)
  let requestedUrl = ''
  api.defaults.adapter = async (config) => {
    requestedUrl = config.url ?? ''
    return {
      data: { success: true, data: { items: [], total: 0 } },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    }
  }

  await getAllLogs({ p: 1, page_size: 20 })

  expect(requestedUrl).toMatch(/^\/api\/log\/\?/)
})

it('falls back to self logs only after a refreshed role is no longer administrative', async () => {
  useAuthStore.getState().auth.setBundle(bundle)
  const refresh = vi.fn(async () => {
    useAuthStore.getState().auth.setUser({
      ...bundle.user,
      role: 1,
    })
    return { kind: 'authenticated' as const, bundle }
  })
  const adminRequest = vi.fn(async () => {
    throw privilegeError()
  })
  const selfRequest = vi.fn(async () => 'self-data')

  await expect(
    fetchWithPrivilegeFallback(adminRequest, selfRequest, true, refresh)
  ).resolves.toBe('self-data')

  expect(refresh).toHaveBeenCalledOnce()
  expect(selfRequest).toHaveBeenCalledOnce()
})

it('does not downgrade an administrative user after an insufficient-privilege response', async () => {
  useAuthStore.getState().auth.setBundle(bundle)
  const refresh = vi.fn(async () => ({
    kind: 'authenticated' as const,
    bundle,
  }))
  const adminRequest = vi.fn(async () => {
    throw privilegeError()
  })
  const selfRequest = vi.fn(async () => 'self-data')

  await expect(
    fetchWithPrivilegeFallback(adminRequest, selfRequest, true, refresh)
  ).rejects.toMatchObject({ response: { status: 403 } })

  expect(refresh).toHaveBeenCalledOnce()
  expect(selfRequest).not.toHaveBeenCalled()
})

it('keeps ordinary users on the self route without probing an admin route', async () => {
  const refresh = vi.fn()
  const adminRequest = vi.fn(async () => 'admin-data')
  const selfRequest = vi.fn(async () => 'self-data')

  await expect(
    fetchWithPrivilegeFallback(adminRequest, selfRequest, false, refresh)
  ).resolves.toBe('self-data')

  expect(adminRequest).not.toHaveBeenCalled()
  expect(refresh).not.toHaveBeenCalled()
})
