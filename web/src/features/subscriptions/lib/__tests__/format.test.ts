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

import type { TFunction } from 'i18next'

import { formatPlanSummary, formatSubscriptionPrice } from '../format'

const translate = ((key: string) => key) as TFunction

describe('subscription plan formatting', () => {
  test('uses a CNY symbol without converting the configured plan amount', () => {
    assert.equal(formatSubscriptionPrice(299), '¥299.00')
    assert.equal(formatSubscriptionPrice('16.66'), '¥16.66')
    assert.equal(formatSubscriptionPrice('invalid'), '¥0.00')
  })

  test('labels unlimited quota and includes reset and validity details', () => {
    const summary = formatPlanSummary(
      {
        total_amount: 0,
        quota_reset_period: 'daily',
        duration_unit: 'month',
        duration_value: 1,
      },
      translate
    )

    assert.equal(
      summary,
      'Total Quota: Unlimited | Quota Reset: Daily | Validity Period: 1 months'
    )
  })
})
