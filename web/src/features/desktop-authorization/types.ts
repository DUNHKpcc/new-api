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
export interface DesktopAuthorizationRequestView {
  client_id: string
  client_name: string
  device_name: string
  platform: string
  app_version: string
  scopes: string[]
  allowed_models: string[]
  expires_at: number
  token_ttl: number
  wechat_verification_required: boolean
  wechat_verified: boolean
}

export interface DesktopAuthorizationDecision {
  redirect_uri: string
}

export interface DesktopGrant {
  public_id: string
  client_id: string
  device_name: string
  platform: string
  app_version: string
  scopes: string
  status: 'active' | 'revoked' | 'expired'
  created_time: number
  last_used_time: number
  expired_time: number
  revoked_time: number
  revoke_reason?: string
}

export interface DesktopGrantResponse {
  success: boolean
  message?: string
  data?: DesktopGrant[]
}
