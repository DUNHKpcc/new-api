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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { PricingModel } from '../../types'
import {
  estimateSubscriptionTokens,
  getSubscriptionResetCount,
  getSubscriptionTotalBudgetUsd,
  parseSubscriptionDisplayModels,
  resolveSubscriptionDisplayModels,
  serializeSubscriptionDisplayModels,
} from '../subscription-estimate'

const dailyTeamPlan = {
  total_amount: 250_000_000,
  duration_unit: 'month' as const,
  duration_value: 1,
  quota_reset_period: 'daily' as const,
}

const cachedTokenModel: PricingModel = {
  id: 1,
  model_name: 'cached-model',
  quota_type: 0,
  model_ratio: 0.625,
  completion_ratio: 8,
  cache_ratio: 0.1,
  enable_groups: ['default'],
  group_ratio: { default: 1 },
}

describe('subscription token estimates', () => {
  test('multiplies a daily monthly plan by thirty reset periods', () => {
    assert.equal(getSubscriptionResetCount(dailyTeamPlan), 30)
    assert.equal(getSubscriptionTotalBudgetUsd(dailyTeamPlan, 500_000), 15_000)
  })

  test('uses input, output, and cache-hit prices for the token estimate', () => {
    const result = estimateSubscriptionTokens(
      dailyTeamPlan,
      cachedTokenModel,
      500_000
    )

    assert.equal(result.reason, 'ok')
    assert.equal(result.tokens, 4_671_587_405)
  })

  test('normalizes configured display models to three unique names', () => {
    const parsed = parseSubscriptionDisplayModels(
      JSON.stringify(['gpt-5', 'gpt-5', 42, 'claude', 'gemini', 'extra'])
    )

    assert.deepEqual(parsed, ['gpt-5', 'claude', 'gemini'])
    assert.equal(
      serializeSubscriptionDisplayModels(['gpt-5', 'gpt-5', 'claude']),
      '["gpt-5","claude"]'
    )
  })

  test('falls back to the first three token-priced models when unset', () => {
    const models = [
      { ...cachedTokenModel, model_name: 'request-model', quota_type: 1 },
      { ...cachedTokenModel, model_name: 'first-model' },
      { ...cachedTokenModel, model_name: 'second-model' },
      { ...cachedTokenModel, model_name: 'third-model' },
      { ...cachedTokenModel, model_name: 'fourth-model' },
    ]

    assert.deepEqual(
      resolveSubscriptionDisplayModels('', models).map((entry) => entry.name),
      ['first-model', 'second-model', 'third-model']
    )
  })

  test('does not convert per-request plans or unlimited plans to token counts', () => {
    const requestResult = estimateSubscriptionTokens(
      dailyTeamPlan,
      { ...cachedTokenModel, quota_type: 1 },
      500_000
    )
    const unlimitedResult = estimateSubscriptionTokens(
      { ...dailyTeamPlan, total_amount: 0 },
      cachedTokenModel,
      500_000
    )

    assert.equal(requestResult.reason, 'request')
    assert.equal(unlimitedResult.reason, 'unlimited')
  })
})
