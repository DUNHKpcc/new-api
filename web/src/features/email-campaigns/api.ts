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
  EmailCampaign,
  EmailConfig,
  EmailDelivery,
  EmailDraft,
  EmailPage,
  EmailSubscriptions,
  UnsubscribeInfo,
} from './types'

interface Response<T> {
  success: boolean
  message: string
  data: T
}
const base = '/api/email-campaign'
export async function getEmailConfig() {
  return requireServerSuccess(
    (await api.get<Response<EmailConfig>>(`${base}/config`)).data
  ).data
}
export async function updateEmailConfig(rate_per_minute: number) {
  return requireServerSuccess(
    (
      await api.put<Response<EmailConfig>>(`${base}/config`, {
        rate_per_minute,
      })
    ).data
  ).data
}
export async function listEmailCampaigns(p: number, page_size: number) {
  return requireServerSuccess(
    (
      await api.get<Response<EmailPage<EmailCampaign>>>(`${base}/`, {
        params: { p, page_size },
      })
    ).data
  ).data
}
export async function saveEmailDraft(draft: EmailDraft, id?: number) {
  const response = id
    ? await api.put<Response<EmailCampaign>>(`${base}/${id}`, draft)
    : await api.post<Response<EmailCampaign>>(`${base}/`, draft)
  return requireServerSuccess(response.data).data
}
export async function previewEmailCampaign(id: number) {
  return requireServerSuccess(
    (await api.get<Response<{ eligible: number }>>(`${base}/${id}/preview`))
      .data
  ).data
}
export async function queueEmailCampaign(id: number) {
  return requireServerSuccess(
    (await api.post<Response<EmailCampaign>>(`${base}/${id}/queue`)).data
  ).data
}
export async function cancelEmailCampaign(id: number) {
  return requireServerSuccess(
    (await api.post<Response<EmailCampaign>>(`${base}/${id}/cancel`)).data
  ).data
}
export async function listEmailDeliveries(
  id: number,
  p: number,
  page_size: number,
  status: string
) {
  return requireServerSuccess(
    (
      await api.get<Response<EmailPage<EmailDelivery>>>(
        `${base}/${id}/deliveries`,
        { params: { p, page_size, status } }
      )
    ).data
  ).data
}
export async function getEmailSubscriptions() {
  return requireServerSuccess(
    (
      await api.get<Response<EmailSubscriptions>>(
        '/api/user/self/email-subscriptions'
      )
    ).data
  ).data
}
export async function updateEmailSubscriptions(value: EmailSubscriptions) {
  return requireServerSuccess(
    (
      await api.put<Response<EmailSubscriptions>>(
        '/api/user/self/email-subscriptions',
        value
      )
    ).data
  ).data
}
export async function getUnsubscribeInfo(token: string) {
  return requireServerSuccess(
    (
      await api.get<Response<UnsubscribeInfo>>('/api/email/unsubscribe', {
        params: { token },
      })
    ).data
  ).data
}
export async function confirmUnsubscribe(token: string) {
  return requireServerSuccess(
    (
      await api.post<Response<UnsubscribeInfo>>(
        '/api/email/unsubscribe',
        { confirm: true },
        { params: { token } }
      )
    ).data
  ).data
}
