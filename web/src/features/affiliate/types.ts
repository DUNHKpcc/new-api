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
export type AffiliateOverview = {
  invite: { code: string; link: string; count: number }
  signup_rewards: {
    enabled: boolean
    inviter_reward_quota: string
    invitee_reward_quota: string
    available_quota: string
    lifetime_quota: string
  }
  program: {
    enabled: boolean
    access: 'inherit' | 'allow' | 'deny'
    status: 'inactive' | 'active' | 'suspended'
    activated_at: number
    eligible: boolean
    qualification_source: string
    verified_amount_minor: string
    qualification_threshold_minor: string
    remaining_amount_minor: string
    currency: string
    commission_rate_bps: number
    commission_wait_days: number
    commission_debt_quota: string
    config_version: number
  }
  commissions: {
    referred_paid_amount_minor: string
    pending_quota: string
    available_quota: string
    transferred_quota: string
    lifetime_quota: string
    reversed_quota: string
  }
}

export type AffiliateCommission = {
  id: string
  referred_user: string
  paid_amount_minor: string
  paid_currency: string
  purchased_quota: string
  commission_rate_bps: number
  commission_amount_minor: string
  reward_quota: string
  debt_offset_quota: string
  status: 'pending' | 'available' | 'transferred' | 'reversed'
  available_at: number
  transferred_at: number
  reversed_at: number
  reverse_reason?: string
  created_at: number
}

export type AffiliateCommissionPage = {
  items: AffiliateCommission[]
  page: number
  page_size: number
  total: number
}

export type AffiliateApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
