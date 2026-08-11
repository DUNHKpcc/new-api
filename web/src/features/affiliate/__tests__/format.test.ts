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

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import {
  formatAffiliateMinor,
  formatAffiliateQuota,
  formatAffiliateRate,
} from '../lib'

describe('affiliate amount formatting', () => {
  test('converts internal quota units to the configured user currency', () => {
    useSystemConfigStore.setState((state) => ({
      config: {
        ...state.config,
        currency: { ...DEFAULT_CURRENCY_CONFIG },
      },
    }))

    assert.equal(formatAffiliateQuota('500000'), '$1')
    assert.equal(formatAffiliateQuota('200000'), '$0.4')
  })

  test('formats minor currency units without numeric conversion', () => {
    assert.equal(
      formatAffiliateMinor('900719925474099312345', 'CNY'),
      'CNY 9,007,199,254,740,993,123.45'
    )
  })

  test('shows commission basis points as a user-facing percentage', () => {
    assert.equal(formatAffiliateRate(10), '0.10%')
    assert.equal(formatAffiliateRate(1000), '10.00%')
  })
})
