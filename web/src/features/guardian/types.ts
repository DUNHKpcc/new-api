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
export type GuardianServiceRoute = 'any' | 'local' | 'vm' | 'vps'
export type GuardianOrderRoute = 'local' | 'vm' | 'vps' | 'unsure'
export type GuardianOs = 'macos' | 'windows' | 'linux' | 'other'
export type GuardianContactType = 'telegram' | 'qq' | 'wechat' | 'email'
export type GuardianOrderStatus =
  | 'pending_payment'
  | 'paid'
  | 'scheduled'
  | 'in_progress'
  | 'delivered'
  | 'expired'
  | 'cancelled'
  | 'refunded'

export interface GuardianService {
  id: number
  title: string
  subtitle: string
  description: string
  features: string[]
  /** CNY in fen (1/100 yuan). */
  price_cents: number
  delivery_note: string
  badge: string
  route: GuardianServiceRoute
  enabled: boolean
  sort_order: number
  created_at: number
  updated_at: number
}

export type GuardianServicePayload = Omit<
  GuardianService,
  'id' | 'created_at' | 'updated_at'
>

export interface GuardianPayMethod {
  type: string
  name: string
}

export interface GuardianCatalog {
  services: GuardianService[]
  pay_methods: GuardianPayMethod[]
  purchase_enabled: boolean
}

export interface GuardianOrderRequest {
  service_id: number
  payment_method: string
  route: GuardianOrderRoute
  os: GuardianOs
  contact_type: GuardianContactType
  contact: string
  note: string
}

export interface GuardianOrderCheckout {
  trade_no: string
  pay_url: string
  pay_params: Record<string, string>
}

export interface GuardianOrder {
  id: number
  trade_no: string
  service_id: number
  service_title: string
  amount_cents: number
  payment_method: string
  status: GuardianOrderStatus
  route: GuardianOrderRoute
  os: GuardianOs
  contact_type: GuardianContactType
  contact: string
  note: string
  admin_note: string
  created_at: number
  paid_at: number
  delivered_at: number
  updated_at: number
}

export interface AdminGuardianOrder extends GuardianOrder {
  user_id: number
  username: string
}

export interface GuardianPage<T> {
  items: T[]
  total: number
}

export interface GuardianOrderUpdate {
  status: GuardianOrderStatus
  admin_note: string
}
