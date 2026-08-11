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
  AffiliateApiResponse,
  AffiliateCommissionPage,
  AffiliateOverview,
} from './types'

export async function getAffiliateOverview() {
  const response = await api.get<AffiliateApiResponse<AffiliateOverview>>(
    '/api/user/affiliate/overview'
  )
  return response.data
}

export async function activateAffiliate() {
  const response = await api.post<AffiliateApiResponse<unknown>>(
    '/api/user/affiliate/activate'
  )
  return response.data
}

export async function getAffiliateCommissions(page: number, pageSize: number) {
  const response = await api.get<AffiliateApiResponse<AffiliateCommissionPage>>(
    '/api/user/affiliate/commissions',
    { params: { page, page_size: pageSize } }
  )
  return response.data
}

export async function transferAffiliateInviteRewards(
  quota: string,
  idempotencyKey: string
) {
  const response = await api.post<AffiliateApiResponse<unknown>>(
    '/api/user/affiliate/invite-rewards/transfer',
    { quota, idempotency_key: idempotencyKey }
  )
  return response.data
}

export async function transferAffiliateCommissions(idempotencyKey: string) {
  const response = await api.post<AffiliateApiResponse<unknown>>(
    '/api/user/affiliate/commissions/transfer',
    { idempotency_key: idempotencyKey }
  )
  return response.data
}
