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

import { getOptionInvalidationQueryKeys } from '../use-update-option'

describe('system option cache invalidation', () => {
  test('refreshes status after announcement content or visibility changes', () => {
    assert.deepEqual(
      getOptionInvalidationQueryKeys('console_setting.announcements'),
      ['system-options', 'status']
    )
    assert.deepEqual(
      getOptionInvalidationQueryKeys('console_setting.announcements_enabled'),
      ['system-options', 'status']
    )
  })

  test('refreshes the dedicated notice query after a system notice changes', () => {
    assert.deepEqual(getOptionInvalidationQueryKeys('Notice'), [
      'system-options',
      'notice',
    ])
  })

  test('refreshes status after a discount notice changes', () => {
    assert.deepEqual(getOptionInvalidationQueryKeys('DiscountNotice'), [
      'system-options',
      'status',
    ])
  })

  test('refreshes public pricing cards after estimate models change', () => {
    assert.deepEqual(
      getOptionInvalidationQueryKeys('SubscriptionDisplayModels'),
      ['system-options', 'status']
    )
  })

  test('refreshes public ranking data after calibration changes', () => {
    assert.deepEqual(getOptionInvalidationQueryKeys('RankingDisplayConfig'), [
      'system-options',
      'rankings',
      'rankings-live',
    ])
  })
})
