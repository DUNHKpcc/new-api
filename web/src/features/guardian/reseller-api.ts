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
import { requireServerSuccess } from '@/lib/server-error-message'

import type {
  GuardianCatalog,
  GuardianService,
  GuardianBillingPeriod,
  GuardianOrderStatus,
} from './types'

export interface GuardianReseller {
  user_id: number
  status: 'active' | 'suspended'
  code: string
  created_by: number
  created_at: number
  updated_at: number
}

interface Response<T> {
  success: boolean
  message: string
  data: T
}

export async function getGuardianReseller(userId?: number) {
  const path = userId
    ? `/api/guardian/admin/resellers/${userId}`
    : '/api/guardian/reseller'
  return requireServerSuccess(
    (await api.get<Response<GuardianReseller | null>>(path)).data
  ).data
}

export async function setGuardianResellerAccess(
  userId: number,
  request: { status: 'active' | 'suspended'; reason: string }
) {
  return requireServerSuccess(
    (
      await api.put<Response<GuardianReseller>>(
        `/api/guardian/admin/resellers/${userId}`,
        request
      )
    ).data
  ).data
}

const base = '/api/guardian'
export async function resellerRead<T>(path: string) {
  return requireServerSuccess((await api.get<Response<T>>(base + path)).data)
    .data
}
export async function resellerWrite<T = unknown>(
  path: string,
  body: unknown,
  method: 'put' | 'post' = 'put',
  proof?: string
) {
  return requireServerSuccess(
    (
      await api[method]<Response<T>>(
        base + path,
        body,
        proof ? { headers: { 'X-Security-Proof': proof } } : undefined
      )
    ).data
  ).data
}
export const getResellerCatalog = (code: string) =>
  resellerRead<GuardianCatalog>(`/resellers/${encodeURIComponent(code)}`)
export interface ResellerOffer {
  id: number
  service_id: number
  granted: boolean
  listed: boolean
  supply_cents: number
  quarter_supply_cents: number
  annual_supply_cents: number
  max_supply_cents: number
  ultra_supply_cents?: number
  retail_cents: number
  quarter_retail_cents: number
  annual_retail_cents: number
  max_retail_cents: number
  ultra_retail_cents?: number
}
export interface ResellerOffers {
  offers: ResellerOffer[]
  services: GuardianService[]
  setting: { fee_bps: number } | null
}
export interface ResellerOrder {
  order_id: number
  reseller_id: number
  trade_no: string
  service_title: string
  billing_period: GuardianBillingPeriod
  amount_cents: number
  supply_cents: number
  estimated_fee_cents: number
  actual_fee_cents: number
  fee_confirmed_at: number
  paid_at: number
  status: GuardianOrderStatus
  refund_state: string
  refund_cents: number
  delivered_at: number
}
export interface Withdrawal {
  id: number
  reseller_id: number
  amount_cents: number
  fee_cents: number
  account: string
  payee_name: string
  status: string
  reference: string | null
  note: string
}
export interface ResellerBalance {
  held_cents: number
  pending_cents: number
  available_cents: number
  reserved_cents: number
  paid_cents: number
  debt_cents: number
}
export interface CustomerService {
  id: number
  service_id: number
  initial_order_id: number
  title: string
  reseller_id: number
}
export interface RenewalQuote {
  service: GuardianService
  reseller_code: string
  direct_confirmation_required: boolean
}
export interface ResellerPage<T> {
  items: T[]
  total: number
}

export interface GuardianResellerSummary {
  user_id: number
  username: string
  display_name: string
  status: 'active' | 'suspended'
}
