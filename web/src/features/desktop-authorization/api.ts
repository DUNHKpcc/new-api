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

import { api } from '@/lib/api'

import type {
  DesktopAuthorizationDecision,
  DesktopAuthorizationRequestView,
  DesktopGrantResponse,
} from './types'

export async function getDesktopAuthorizationRequest(
  requestToken: string
): Promise<DesktopAuthorizationRequestView> {
  const response = await api.get(
    `/api/desktop/oauth/authorization-requests/${encodeURIComponent(requestToken)}`,
    { skipBusinessError: true, skipErrorHandler: true }
  )
  return response.data as DesktopAuthorizationRequestView
}

export async function decideDesktopAuthorization(
  requestToken: string,
  decision: 'allow' | 'deny'
): Promise<DesktopAuthorizationDecision> {
  const response = await api.post(
    '/api/desktop/oauth/authorize',
    { request_token: requestToken, decision },
    { skipBusinessError: true, skipErrorHandler: true }
  )
  return response.data as DesktopAuthorizationDecision
}

export async function getDesktopGrants(): Promise<DesktopGrantResponse> {
  const response = await api.get('/api/user/desktop-grants')
  return response.data as DesktopGrantResponse
}

export async function revokeDesktopGrant(
  publicID: string
): Promise<DesktopGrantResponse> {
  const response = await api.delete(
    `/api/user/desktop-grants/${encodeURIComponent(publicID)}`
  )
  return response.data as DesktopGrantResponse
}
