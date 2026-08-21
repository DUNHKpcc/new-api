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
  getRankingDisplayTotal,
  parseRankingDisplayConfig,
  serializeRankingDisplayConfig,
} from '../display-config'

describe('ranking display configuration', () => {
  test('preserves valid added tokens for each supported period', () => {
    const config = parseRankingDisplayConfig(
      JSON.stringify({
        version: 1,
        enabled: true,
        periods: {
          today: { adjustments: { 'model-a': 100 } },
          week: { adjustments: { 'model-b': 200 } },
        },
      })
    )

    assert.equal(config.enabled, true)
    assert.equal(config.periods.today?.adjustments['model-a'], 100)
    assert.equal(config.periods.week?.adjustments['model-b'], 200)
  })

  test('drops invalid and zero adjustments before saving', () => {
    const config = parseRankingDisplayConfig(
      JSON.stringify({
        version: 1,
        enabled: true,
        periods: {
          week: {
            adjustments: {
              'model-a': 100,
              'model-zero': 0,
              'model-negative': -1,
            },
          },
        },
      })
    )

    assert.deepEqual(JSON.parse(serializeRankingDisplayConfig(config)), {
      version: 1,
      enabled: true,
      periods: { week: { adjustments: { 'model-a': 100 } } },
    })
  })

  test('previews the formatted total from live and added tokens', () => {
    assert.equal(getRankingDisplayTotal(240_000, 10_000), 250_000)
  })

  test('falls back to disabled configuration for malformed input', () => {
    const config = parseRankingDisplayConfig('{broken')

    assert.equal(config.enabled, false)
    assert.deepEqual(config.periods, {})
  })
})
