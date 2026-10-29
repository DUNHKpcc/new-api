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
  AdminGuardianOrder,
  GuardianCatalog,
  GuardianOrder,
  GuardianOrderCheckout,
  GuardianOrderRequest,
  GuardianOrderStatus,
  GuardianOrderUpdate,
  GuardianPage,
  GuardianService,
  GuardianServicePayload,
} from './types'

interface Response<T> {
  success: boolean
  message: string
  data: T
}

const base = '/api/guardian'

export async function getGuardianCatalog() {
  return requireServerSuccess(
    (await api.get<Response<GuardianCatalog>>(`${base}/services`)).data
  ).data
}

export async function createGuardianOrder(request: GuardianOrderRequest) {
  return requireServerSuccess(
    (await api.post<Response<GuardianOrderCheckout>>(`${base}/orders`, request))
      .data
  ).data
}

export async function listMyGuardianOrders(p: number, page_size: number) {
  return requireServerSuccess(
    (
      await api.get<Response<GuardianPage<GuardianOrder>>>(
        `${base}/orders/self`,
        { params: { p, page_size } }
      )
    ).data
  ).data
}

export async function payGuardianOrder(id: number) {
  return requireServerSuccess(
    (
      await api.post<Response<GuardianOrderCheckout>>(
        `${base}/orders/${id}/pay`
      )
    ).data
  ).data
}

export async function cancelGuardianOrder(id: number) {
  return requireServerSuccess(
    (await api.post<Response<GuardianOrder>>(`${base}/orders/${id}/cancel`))
      .data
  ).data
}

export async function listAdminGuardianServices() {
  return requireServerSuccess(
    (await api.get<Response<GuardianService[]>>(`${base}/admin/services`)).data
  ).data
}

export async function saveGuardianService(
  payload: GuardianServicePayload,
  id?: number
) {
  const response = id
    ? await api.put<Response<GuardianService>>(
        `${base}/admin/services/${id}`,
        payload
      )
    : await api.post<Response<GuardianService>>(
        `${base}/admin/services`,
        payload
      )
  return requireServerSuccess(response.data).data
}

export async function listAdminGuardianOrders(params: {
  p: number
  page_size: number
  status: GuardianOrderStatus | ''
  keyword: string
}) {
  return requireServerSuccess(
    (
      await api.get<Response<GuardianPage<AdminGuardianOrder>>>(
        `${base}/admin/orders`,
        { params }
      )
    ).data
  ).data
}

export async function updateAdminGuardianOrder(
  id: number,
  update: GuardianOrderUpdate
) {
  return requireServerSuccess(
    (
      await api.put<Response<AdminGuardianOrder>>(
        `${base}/admin/orders/${id}`,
        update
      )
    ).data
  ).data
}
