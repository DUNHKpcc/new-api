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
import type { UserSubscriptionRecord } from '../types'

export const PCC_AGENT_GIFT_SOURCE = 'pcc_agent_gift'

type SubscriptionActionSource = Pick<
  UserSubscriptionRecord['subscription'],
  'source' | 'status' | 'end_time'
>

export function isUserSubscriptionActive(
  subscription: Pick<
    UserSubscriptionRecord['subscription'],
    'status' | 'end_time'
  >,
  now: number
): boolean {
  const isExpired = subscription.end_time > 0 && subscription.end_time < now
  return subscription.status === 'active' && !isExpired
}

export function isPccAgentGiftSubscription(
  subscription: Pick<UserSubscriptionRecord['subscription'], 'source'>
): boolean {
  return subscription.source === PCC_AGENT_GIFT_SOURCE
}

export function getUserSubscriptionActionPolicy(
  subscription: SubscriptionActionSource,
  now: number
) {
  const isActive = isUserSubscriptionActive(subscription, now)
  const isPccAgentGift = isPccAgentGiftSubscription(subscription)

  return {
    isActive,
    showReset: !isPccAgentGift,
    canReset: isActive && !isPccAgentGift,
    canInvalidate: isActive,
    canDelete: !isPccAgentGift,
  }
}
