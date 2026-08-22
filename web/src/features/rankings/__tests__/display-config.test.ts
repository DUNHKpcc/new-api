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
  MAX_RANKING_DISPLAY_DATES,
  parseRankingDisplayConfig,
  serializeRankingDisplayConfig,
} from '../display-config'

describe('ranking display configuration', () => {
  test('preserves valid date-based added tokens', () => {
    const config = parseRankingDisplayConfig(
      JSON.stringify({
        version: 1,
        enabled: true,
        daily_records: {
          '2026-08-22': { adjustments: { 'model-a': 100 } },
          '2026-08-21': { adjustments: { 'model-b': 200 } },
        },
      })
    )

    assert.equal(config.enabled, true)
    assert.equal(
      config.daily_records['2026-08-22']?.adjustments['model-a'],
      100
    )
    assert.equal(
      config.daily_records['2026-08-21']?.adjustments['model-b'],
      200
    )
  })

  test('drops invalid and zero date adjustments before saving', () => {
    const config = parseRankingDisplayConfig(
      JSON.stringify({
        version: 1,
        enabled: true,
        daily_records: {
          '2026-08-22': {
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
      periods: {},
      daily_records: { '2026-08-22': { adjustments: { 'model-a': 100 } } },
    })
  })

  test('preserves legacy period adjustments', () => {
    const config = parseRankingDisplayConfig(
      JSON.stringify({
        version: 1,
        enabled: true,
        periods: {
          week: { adjustments: { 'model-a': 200 } },
        },
      })
    )

    assert.equal(config.periods.week?.adjustments['model-a'], 200)
    assert.deepEqual(JSON.parse(serializeRankingDisplayConfig(config)), {
      version: 1,
      enabled: true,
      periods: { week: { adjustments: { 'model-a': 200 } } },
      daily_records: {},
    })
  })

  test('keeps the newest daily records when the retention limit is exceeded', () => {
    const daily_records: Record<
      string,
      { adjustments: { 'model-a': number } }
    > = {}
    for (let index = 0; index <= MAX_RANKING_DISPLAY_DATES; index += 1) {
      const date = new Date(Date.UTC(2020, 0, index + 1))
        .toISOString()
        .slice(0, 10)
      daily_records[date] = { adjustments: { 'model-a': index + 1 } }
    }

    const config = parseRankingDisplayConfig(
      JSON.stringify({ version: 1, enabled: true, daily_records })
    )
    const newestDate = Object.keys(daily_records).sort().at(-1)

    assert.equal(
      Object.keys(config.daily_records).length,
      MAX_RANKING_DISPLAY_DATES
    )
    assert.ok(newestDate)
    assert.equal(
      config.daily_records[newestDate]?.adjustments['model-a'],
      MAX_RANKING_DISPLAY_DATES + 1
    )

    const serialized = JSON.parse(serializeRankingDisplayConfig(config))
    assert.equal(
      Object.keys(serialized.daily_records).length,
      MAX_RANKING_DISPLAY_DATES
    )
    assert.ok(serialized.daily_records[newestDate])
  })

  test('previews the formatted total from live and added tokens', () => {
    assert.equal(getRankingDisplayTotal(240_000, 10_000), 250_000)
  })

  test('falls back to disabled configuration for malformed input', () => {
    const config = parseRankingDisplayConfig('{broken')

    assert.equal(config.enabled, false)
    assert.deepEqual(config.periods, {})
    assert.deepEqual(config.daily_records, {})
  })
})
