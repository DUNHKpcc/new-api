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

import { tileSuccessRates } from '../status-series'

describe('performance status series', () => {
  test('tiles three recent rates across the full status bar', () => {
    assert.deepEqual(
      tileSuccessRates([100, 92.5, 98], 8),
      [100, 92.5, 98, 100, 92.5, 98, 100, 92.5]
    )
  })

  test('repeats a single recent rate when only one bucket is available', () => {
    assert.deepEqual(tileSuccessRates([99.5], 4), [99.5, 99.5, 99.5, 99.5])
  })

  test('ignores invalid rates and returns no segments without data', () => {
    assert.deepEqual(tileSuccessRates([Number.NaN, 95], 3), [95, 95, 95])
    assert.deepEqual(
      tileSuccessRates([Number.NaN, Number.POSITIVE_INFINITY], 3),
      []
    )
  })
})
