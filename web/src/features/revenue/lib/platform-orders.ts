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
import {
  getUserSubscriptions,
  getAdminPlans,
} from '@/features/subscriptions/api'
import type {
  PlanRecord,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { getUser } from '@/features/users/api'
import type { User } from '@/features/users/types'
import type { TopupRecord } from '@/features/wallet/types'

export type RevenueUserSummary = Pick<User, 'id' | 'username' | 'display_name'>

export interface PlatformOrderContext {
  users: Record<string, RevenueUserSummary>
  subscriptions: Record<string, UserSubscriptionRecord[]>
  planTitles: Record<string, string>
}

export interface ResolvedOrderSubscription {
  planId?: number
  title: string
  matched: boolean
}

const SUBSCRIPTION_ORDER_PATTERN =
  /^(sub_ref_|subusr|subbalusr|waffo_pancake_sub-)/i
const SUBSCRIPTION_MATCH_WINDOW_SECONDS = 7 * 24 * 60 * 60
const NON_ORDER_SUBSCRIPTION_SOURCES = new Set(['admin', 'pcc_agent_gift'])

function trimmedString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function positiveInteger(value: unknown): number | undefined {
  const number = typeof value === 'number' ? value : Number(value)
  return Number.isInteger(number) && number > 0 ? number : undefined
}

export function isSubscriptionOrderTradeNo(tradeNo: string): boolean {
  return (
    typeof tradeNo === 'string' &&
    SUBSCRIPTION_ORDER_PATTERN.test(tradeNo.trim())
  )
}

function uniquePositiveUserIds(records: TopupRecord[]): number[] {
  return [
    ...new Set(
      records
        .map((record) => positiveInteger(record.user_id))
        .filter((userId): userId is number => userId !== undefined)
    ),
  ]
}

function planTitleMap(records: PlanRecord[]): Record<string, string> {
  return Object.fromEntries(
    records
      .filter(
        (record) =>
          Boolean(record?.plan) &&
          Number.isInteger(record.plan.id) &&
          record.plan.id > 0
      )
      .map((record) => [
        String(record.plan.id),
        trimmedString(record.plan.title) || `#${record.plan.id}`,
      ])
  )
}

/** Fetches only the existing admin resources needed to decorate the visible rows. */
export async function loadPlatformOrderContext(
  records: TopupRecord[]
): Promise<PlatformOrderContext> {
  const userIds = uniquePositiveUserIds(records)
  const subscriptionUserIds = [
    ...new Set(
      records
        .filter(
          (record) =>
            isSubscriptionOrderTradeNo(record.trade_no) ||
            positiveInteger(record.subscription_plan_id ?? record.plan_id) !==
              undefined
        )
        .map((record) => record.user_id)
        .map((userId) => positiveInteger(userId))
        .filter((userId): userId is number => userId !== undefined)
    ),
  ]
  const [userResults, planResult, subscriptionResults] = await Promise.all([
    Promise.all(
      userIds.map(async (userId) => {
        try {
          const response = await getUser(userId)
          return response.success && response.data
            ? { userId, user: response.data }
            : null
        } catch {
          return null
        }
      })
    ),
    subscriptionUserIds.length > 0
      ? getAdminPlans().catch(() => null)
      : Promise.resolve(null),
    Promise.all(
      subscriptionUserIds.map(async (userId) => {
        try {
          const response = await getUserSubscriptions(userId)
          return response.success && Array.isArray(response.data)
            ? { userId, subscriptions: response.data ?? [] }
            : null
        } catch {
          return null
        }
      })
    ),
  ])

  const users = Object.fromEntries(
    userResults
      .filter(
        (result): result is { userId: number; user: User } => result !== null
      )
      .map(({ userId, user }) => [
        String(userId),
        {
          id: user.id,
          username: user.username,
          display_name: user.display_name,
        },
      ])
  )
  const subscriptions = Object.fromEntries(
    subscriptionResults
      .filter(
        (
          result
        ): result is {
          userId: number
          subscriptions: UserSubscriptionRecord[]
        } => result !== null
      )
      .map(({ userId, subscriptions: recordsForUser }) => [
        String(userId),
        recordsForUser,
      ])
  )
  const planTitles =
    planResult?.success && Array.isArray(planResult.data)
      ? planTitleMap(planResult.data ?? [])
      : {}
  return { users, subscriptions, planTitles }
}

export function getOrderUser(
  record: TopupRecord,
  context: PlatformOrderContext | undefined
): RevenueUserSummary | undefined {
  const user = context?.users[String(record.user_id)]
  if (user) return user
  const username = trimmedString(record.username)
  const displayName = trimmedString(record.display_name)
  if (username || displayName) {
    return {
      id: record.user_id,
      username,
      display_name: displayName,
    }
  }
  return undefined
}

function subscriptionTimestamp(record: UserSubscriptionRecord): number {
  const createdAt = Number(record.subscription.created_at)
  if (Number.isFinite(createdAt) && createdAt > 0) return createdAt
  const startTime = Number(record.subscription.start_time)
  return Number.isFinite(startTime) && startTime > 0 ? startTime : 0
}

export function resolveOrderSubscription(
  record: TopupRecord,
  context: PlatformOrderContext | undefined
): ResolvedOrderSubscription | undefined {
  const explicitTitle = trimmedString(record.subscription_plan_title)
  const explicitPlanId = positiveInteger(
    record.subscription_plan_id ?? record.plan_id
  )
  if (explicitTitle) {
    return {
      planId: explicitPlanId,
      title: explicitTitle,
      matched: true,
    }
  }

  if (explicitPlanId) {
    return {
      planId: explicitPlanId,
      title:
        context?.planTitles[String(explicitPlanId)] ?? `#${explicitPlanId}`,
      matched: true,
    }
  }

  if (!isSubscriptionOrderTradeNo(record.trade_no)) return undefined
  const subscriptions = (
    context?.subscriptions[String(record.user_id)] ?? []
  ).filter((recordItem) => {
    const source = trimmedString(recordItem.subscription.source).toLowerCase()
    return !source || !NON_ORDER_SUBSCRIPTION_SOURCES.has(source)
  })
  const targetTimestamp = record.complete_time || record.create_time
  let candidate: UserSubscriptionRecord | undefined
  if (!candidate && subscriptions.length > 0) {
    const ranked = subscriptions
      .map((recordItem) => ({
        recordItem,
        distance: Math.abs(subscriptionTimestamp(recordItem) - targetTimestamp),
      }))
      .sort(
        (left, right) =>
          left.distance - right.distance ||
          right.recordItem.subscription.id - left.recordItem.subscription.id
      )
    if (
      ranked[0] &&
      (ranked[0].distance <= SUBSCRIPTION_MATCH_WINDOW_SECONDS ||
        subscriptions.length === 1)
    ) {
      candidate = ranked[0].recordItem
    }
  }

  const planId = explicitPlanId ?? candidate?.subscription.plan_id
  const title = planId ? context?.planTitles[String(planId)] : undefined
  if (candidate || planId) {
    return {
      planId,
      title: title || (planId ? `#${planId}` : 'Subscription order'),
      matched: Boolean(candidate),
    }
  }
  return { title: 'Subscription order', matched: false }
}
