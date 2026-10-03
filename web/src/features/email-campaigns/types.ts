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
export type EmailCategory = 'promotion' | 'platform'
export type DeliveryStatus =
  | 'pending'
  | 'sending'
  | 'retry'
  | 'sent'
  | 'failed'
  | 'skipped'
  | 'uncertain'
  | 'cancelled'
export type CampaignStatus =
  | 'draft'
  | 'queued'
  | 'sending'
  | 'completed'
  | 'cancelled'
export interface EmailDraft {
  subject: string
  body: string
  category: EmailCategory
  group: string
}
export interface EmailCampaign extends EmailDraft {
  id: number
  status: CampaignStatus
  created_by: number
  created_at: number
  updated_at: number
  delivery_counts: Record<DeliveryStatus | 'total', number>
}
export interface EmailDelivery {
  id: number
  campaign_id: number
  user_id: number
  email: string
  status: DeliveryStatus
  attempts: number
  next_attempt_at: number
  last_error: string
  sent_at: number
  created_at: number
  updated_at: number
}
export interface EmailConfig {
  rate_per_minute: number
  smtp_configured: boolean
  server_address_ready: boolean
}
export interface EmailSubscriptions {
  promotion: boolean
  platform: boolean
}
export interface UnsubscribeInfo {
  category: EmailCategory
  unsubscribed: boolean
}
export interface EmailPage<T> {
  items: T[]
  total: number
  p: number
  page_size: number
}
