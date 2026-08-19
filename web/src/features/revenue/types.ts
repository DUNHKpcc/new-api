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
import type { ApiResponse, TopupRecord } from '@/features/wallet/types'

export type RevenueSource = 'xianyu' | 'wechat' | 'alipay' | 'other'
export type ExternalRevenueStatus = 'active' | 'voided'

export interface ExternalRevenueRecord {
  id: number
  source: RevenueSource
  source_label: string
  external_order_no: string | null
  amount_minor: string
  currency: string
  occurred_at: number
  note: string
  status: ExternalRevenueStatus
  version: number
  created_by: number
  updated_by: number
  voided_by: number
  create_time: number
  update_time: number
  void_time: number
}

export interface ExternalRevenueMutation {
  source: RevenueSource
  source_label: string
  external_order_no: string
  amount_minor: string
  currency: string
  occurred_at: number
  note: string
  version?: number
}

export interface RevenueTotal {
  currency: string
  amount_minor: string
  count: number
}

export interface ExternalRevenueSourceTotal extends RevenueTotal {
  source: RevenueSource
  source_label: string
}

export interface ExternalRevenueTimelinePoint extends RevenueTotal {
  date: string
  source: RevenueSource
  source_label: string
}

export interface ExternalRevenueSummary {
  totals: RevenueTotal[] | null
  by_source: ExternalRevenueSourceTotal[] | null
  timeline: ExternalRevenueTimelinePoint[] | null
  start_time: number
  end_time: number
  timezone_offset: number
}

export interface PlatformRevenueTotal extends RevenueTotal {
  verified_count: number
}

export interface PlatformRevenueMethodTotal extends RevenueTotal {
  payment_method: string
}

export interface PlatformRevenueTimelinePoint extends RevenueTotal {
  date: string
  payment_method: string
}

export interface PlatformRevenueSummary {
  totals: PlatformRevenueTotal[] | null
  by_payment_method: PlatformRevenueMethodTotal[] | null
  timeline: PlatformRevenueTimelinePoint[] | null
  default_currency: string
  start_time: number
  end_time: number
  timezone_offset: number
  normalization_basis: string
}

export interface ExternalRevenuePage {
  items: ExternalRevenueRecord[]
  total: number
  page: number
  page_size: number
}

export type ExternalRevenuePageResponse = ApiResponse<ExternalRevenuePage>
export type ExternalRevenueRecordResponse = ApiResponse<ExternalRevenueRecord>
export type ExternalRevenueSummaryResponse = ApiResponse<ExternalRevenueSummary>
export type PlatformRevenueSummaryResponse = ApiResponse<PlatformRevenueSummary>

export interface PlatformOrderPage {
  items: TopupRecord[]
  total: number
}

export interface RevenueMonthRange {
  startTime: number
  endTime: number
  label: string
}

export interface CombinedRevenueTotal {
  currency: string
  platformAmountMinor: string
  externalAmountMinor: string
  totalAmountMinor: string
  platformCount: number
  externalCount: number
}
