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
  ExternalRevenueMutation,
  ExternalRevenuePageResponse,
  ExternalRevenueRecordResponse,
  ExternalRevenueSummaryResponse,
  PlatformRevenueSummaryResponse,
} from './types'

type ExternalRevenueListParams = {
  page: number
  pageSize: number
  keyword?: string
  source?: string
  status?: string
}

function requireSuccess<T extends { success?: boolean; message?: string }>(
  response: T
): T {
  if (!response.success) {
    throw new Error(response.message || 'Request failed')
  }
  return response
}

export async function getExternalRevenue(
  params: ExternalRevenueListParams
): Promise<ExternalRevenuePageResponse> {
  const search = new URLSearchParams({
    p: params.page.toString(),
    page_size: params.pageSize.toString(),
  })
  if (params.keyword) search.set('keyword', params.keyword)
  if (params.source) search.set('source', params.source)
  if (params.status) search.set('status', params.status)
  const response = await api.get(
    `/api/admin/external-revenue?${search.toString()}`
  )
  return requireSuccess(response.data as ExternalRevenuePageResponse)
}

export async function createExternalRevenue(
  payload: ExternalRevenueMutation
): Promise<ExternalRevenueRecordResponse> {
  const response = await api.post('/api/admin/external-revenue', payload)
  return requireSuccess(response.data as ExternalRevenueRecordResponse)
}

export async function updateExternalRevenue(
  id: number,
  payload: ExternalRevenueMutation
): Promise<ExternalRevenueRecordResponse> {
  const response = await api.put(`/api/admin/external-revenue/${id}`, payload)
  return requireSuccess(response.data as ExternalRevenueRecordResponse)
}

export async function voidExternalRevenue(
  id: number,
  version: number
): Promise<void> {
  const response = await api.post(`/api/admin/external-revenue/${id}/void`, {
    version,
  })
  requireSuccess(response.data as { success?: boolean; message?: string })
}

function summarySearch(startTime: number, endTime: number): string {
  return new URLSearchParams({
    start_time: startTime.toString(),
    end_time: endTime.toString(),
    timezone_offset: (-new Date().getTimezoneOffset()).toString(),
  }).toString()
}

export async function getExternalRevenueSummary(
  startTime: number,
  endTime: number
): Promise<ExternalRevenueSummaryResponse> {
  const response = await api.get(
    `/api/admin/external-revenue/summary?${summarySearch(startTime, endTime)}`
  )
  return requireSuccess(response.data as ExternalRevenueSummaryResponse)
}

export async function getPlatformRevenueSummary(
  startTime: number,
  endTime: number
): Promise<PlatformRevenueSummaryResponse> {
  const response = await api.get(
    `/api/admin/revenue/platform-summary?${summarySearch(startTime, endTime)}`
  )
  return requireSuccess(response.data as PlatformRevenueSummaryResponse)
}
