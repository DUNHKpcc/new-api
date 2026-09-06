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
import type { SubscriptionPlan } from '@/features/subscriptions/types'

import { QUOTA_TYPE_VALUES } from '../constants'
import type { PricingModel } from '../types'
import {
  getDynamicPricingTiers,
  hasDynamicRequestRules,
  isDynamicPricingModel,
} from './dynamic-price'
import { getDisplayGroupRatio, isTokenBasedModel } from './model-helpers'
import { getTokenPricesPerMillion } from './price'

export const SUBSCRIPTION_ESTIMATE_INPUT_TOKENS = 1_000
export const SUBSCRIPTION_ESTIMATE_OUTPUT_TOKENS = 4_000
export const SUBSCRIPTION_ESTIMATE_CACHE_HIT_RATE = 0.99
export const SUBSCRIPTION_ESTIMATE_DISPLAY_MULTIPLIER = 2.5
export const MAX_SUBSCRIPTION_DISPLAY_MODELS = 3

const DAYS_PER_MONTH = 30
const DAYS_PER_YEAR = 365
const HOURS_PER_DAY = 24
const SECONDS_PER_DAY = 86_400
const MONTHS_PER_YEAR = 12

export type SubscriptionEstimateReason =
  | 'ok'
  | 'unlimited'
  | 'free'
  | 'request'
  | 'dynamic'
  | 'unavailable'

export type SubscriptionTokenEstimate = {
  tokens: number | null
  costPerSampleUsd: number | null
  reason: SubscriptionEstimateReason
}

type TokenPricesPerMillion = {
  input: number
  output: number
  cache: number
}

function positiveFiniteNumber(value: unknown, fallback: number): number {
  const number = Number(value)
  return Number.isFinite(number) && number > 0 ? number : fallback
}

function nonNegativeFiniteNumber(value: unknown): number | null {
  const number = Number(value)
  return Number.isFinite(number) && number >= 0 ? number : null
}

function getPlanDurationDays(plan: Partial<SubscriptionPlan>): number {
  const unit = plan.duration_unit || 'month'
  const value = positiveFiniteNumber(plan.duration_value, 1)

  switch (unit) {
    case 'year':
      return value * DAYS_PER_YEAR
    case 'month':
      return value * DAYS_PER_MONTH
    case 'day':
      return value
    case 'hour':
      return value / HOURS_PER_DAY
    case 'custom':
      return (
        positiveFiniteNumber(plan.custom_seconds, SECONDS_PER_DAY) /
        SECONDS_PER_DAY
      )
    default:
      return DAYS_PER_MONTH
  }
}

function getPlanDurationMonths(plan: Partial<SubscriptionPlan>): number | null {
  const unit = plan.duration_unit || 'month'
  const value = positiveFiniteNumber(plan.duration_value, 1)

  if (unit === 'year') return value * MONTHS_PER_YEAR
  if (unit === 'month') return value
  return null
}

/**
 * Return the number of quota allocations a plan can receive during its
 * validity period. Subscription plans use calendar-friendly 30/365 day
 * estimates so a one-month daily plan follows the displayed 30-day policy.
 */
export function getSubscriptionResetCount(
  plan: Partial<SubscriptionPlan>
): number {
  const resetPeriod = plan.quota_reset_period || 'never'
  if (resetPeriod === 'never') return 1

  if (resetPeriod === 'monthly') {
    const durationMonths = getPlanDurationMonths(plan)
    if (durationMonths !== null) {
      return Math.max(1, Math.ceil(durationMonths))
    }
    return Math.max(1, Math.ceil(getPlanDurationDays(plan) / DAYS_PER_MONTH))
  }

  if (resetPeriod === 'custom') {
    const resetSeconds = positiveFiniteNumber(
      plan.quota_reset_custom_seconds,
      SECONDS_PER_DAY
    )
    const durationSeconds = getPlanDurationDays(plan) * SECONDS_PER_DAY
    return Math.max(1, Math.ceil(durationSeconds / resetSeconds))
  }

  const durationDays = getPlanDurationDays(plan)
  const divisor = resetPeriod === 'weekly' ? 7 : 1
  return Math.max(1, Math.ceil(durationDays / divisor))
}

/**
 * Convert the raw per-reset quota stored by the backend to the total USD
 * equivalent available across the plan's validity period.
 */
export function getSubscriptionTotalBudgetUsd(
  plan: Partial<SubscriptionPlan>,
  quotaPerUnit: number
): number | null {
  const rawAmount = nonNegativeFiniteNumber(plan.total_amount)
  const unit = positiveFiniteNumber(quotaPerUnit, 0)
  if (rawAmount === null || rawAmount <= 0 || unit <= 0) return null

  const budget = (rawAmount / unit) * getSubscriptionResetCount(plan)
  return Number.isFinite(budget) && budget > 0 ? budget : null
}

function conditionsMatch(
  conditions: Array<{ var: 'p' | 'c' | 'len'; op: string; value: number }>,
  values: Record<'p' | 'c' | 'len', number>
): boolean {
  return conditions.every((condition) => {
    const actual = values[condition.var]
    if (!Number.isFinite(actual) || !Number.isFinite(condition.value)) {
      return false
    }

    switch (condition.op) {
      case '<':
        return actual < condition.value
      case '<=':
        return actual <= condition.value
      case '>':
        return actual > condition.value
      case '>=':
        return actual >= condition.value
      default:
        return false
    }
  })
}

function getDynamicTokenPrices(
  model: PricingModel,
  selectedGroup?: string
): TokenPricesPerMillion | null {
  if (!isDynamicPricingModel(model) || hasDynamicRequestRules(model)) {
    return null
  }

  const tiers = getDynamicPricingTiers(model)
  const uncachedInputTokens =
    SUBSCRIPTION_ESTIMATE_INPUT_TOKENS *
    (1 - SUBSCRIPTION_ESTIMATE_CACHE_HIT_RATE)
  const tier = tiers.find((candidate) => {
    if (!candidate.conditions.length) return true
    return conditionsMatch(candidate.conditions, {
      p: uncachedInputTokens,
      c: SUBSCRIPTION_ESTIMATE_OUTPUT_TOKENS,
      len: SUBSCRIPTION_ESTIMATE_INPUT_TOKENS,
    })
  })
  if (!tier) return null

  const groupRatio = getDisplayGroupRatio(model, selectedGroup)
  const input = Number(tier.input_unit_cost) * groupRatio
  const output = Number(tier.output_unit_cost) * groupRatio
  const expression = model.billing_expr || ''
  const includesCacheRead = /\bcr\s*\*/.test(expression)
  const cache = includesCacheRead
    ? Number(tier.cache_read_unit_cost) * groupRatio
    : input

  if (![input, output, cache].every((value) => Number.isFinite(value))) {
    return null
  }
  if ([input, output, cache].some((value) => value < 0)) return null

  return { input, output, cache }
}

function getModelTokenPrices(
  model: PricingModel,
  selectedGroup?: string
): TokenPricesPerMillion | null {
  if (isDynamicPricingModel(model)) {
    return getDynamicTokenPrices(model, selectedGroup)
  }

  const prices = getTokenPricesPerMillion(model, selectedGroup)
  if (!prices) return null

  const input = nonNegativeFiniteNumber(prices.input)
  const output = nonNegativeFiniteNumber(prices.output)
  const cache = nonNegativeFiniteNumber(prices.cache)
  if (input === null || output === null || cache === null) return null

  return { input, output, cache }
}

/**
 * Estimate total tokens for a transparent, fixed sample workload. The sample
 * keeps input/output/cache assumptions stable so cards can compare models
 * without touching the payment or settlement paths.
 */
export function estimateSubscriptionTokens(
  plan: Partial<SubscriptionPlan>,
  model: PricingModel,
  quotaPerUnit: number,
  selectedGroup?: string
): SubscriptionTokenEstimate {
  if (Number(plan.total_amount || 0) <= 0) {
    return { tokens: null, costPerSampleUsd: null, reason: 'unlimited' }
  }

  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return { tokens: null, costPerSampleUsd: null, reason: 'request' }
  }

  const budgetUsd = getSubscriptionTotalBudgetUsd(plan, quotaPerUnit)
  if (budgetUsd === null) {
    return { tokens: null, costPerSampleUsd: null, reason: 'unavailable' }
  }

  const prices = getModelTokenPrices(model, selectedGroup)
  if (prices === null) {
    return {
      tokens: null,
      costPerSampleUsd: null,
      reason: isDynamicPricingModel(model) ? 'dynamic' : 'unavailable',
    }
  }

  const inputTokens = SUBSCRIPTION_ESTIMATE_INPUT_TOKENS
  const outputTokens = SUBSCRIPTION_ESTIMATE_OUTPUT_TOKENS
  const cacheHitTokens = inputTokens * SUBSCRIPTION_ESTIMATE_CACHE_HIT_RATE
  const regularInputTokens = inputTokens - cacheHitTokens
  const costPerSampleUsd =
    (regularInputTokens * prices.input +
      cacheHitTokens * prices.cache +
      outputTokens * prices.output) /
    1_000_000

  if (!Number.isFinite(costPerSampleUsd) || costPerSampleUsd < 0) {
    return { tokens: null, costPerSampleUsd: null, reason: 'unavailable' }
  }
  if (costPerSampleUsd === 0) {
    return { tokens: null, costPerSampleUsd: 0, reason: 'free' }
  }

  const tokens = Math.floor(
    (budgetUsd / costPerSampleUsd) *
      (inputTokens + outputTokens) *
      SUBSCRIPTION_ESTIMATE_DISPLAY_MULTIPLIER
  )
  if (!Number.isFinite(tokens) || tokens < 0) {
    return { tokens: null, costPerSampleUsd, reason: 'unavailable' }
  }

  return { tokens, costPerSampleUsd, reason: 'ok' }
}

export function parseSubscriptionDisplayModels(raw: unknown): string[] {
  let parsed: unknown = raw

  if (typeof raw === 'string') {
    if (!raw.trim()) return []
    try {
      parsed = JSON.parse(raw)
    } catch {
      return []
    }
  }

  if (!Array.isArray(parsed)) return []

  const names: string[] = []
  for (const value of parsed) {
    if (typeof value !== 'string') continue
    const name = value.trim()
    if (!name || names.includes(name)) continue
    names.push(name)
    if (names.length >= MAX_SUBSCRIPTION_DISPLAY_MODELS) break
  }
  return names
}

export function serializeSubscriptionDisplayModels(models: string[]): string {
  return JSON.stringify(parseSubscriptionDisplayModels(models))
}

export function resolveSubscriptionDisplayModels(
  configured: unknown,
  models: PricingModel[]
): Array<{ name: string; model: PricingModel | null }> {
  const configuredNames = parseSubscriptionDisplayModels(configured)
  const names = configuredNames.length
    ? configuredNames
    : models
        .filter((model) => isTokenBasedModel(model))
        .slice(0, MAX_SUBSCRIPTION_DISPLAY_MODELS)
        .map((model) => model.model_name)

  return names.map((name) => ({
    name,
    model: models.find((model) => model.model_name === name) || null,
  }))
}

export function formatEstimatedTokenCount(tokens: number): string {
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(tokens)
}
