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
import { describe, expect, it } from 'vitest'

import { centsToYuanInput, formatCny, yuanToCents } from '../lib/price'

describe('yuanToCents', () => {
  it.each([
    ['599', 59900],
    ['599.5', 59950],
    ['0.01', 1],
    [' 12.34 ', 1234],
    ['100000', 10_000_000],
  ])('converts %j yuan into whole fen', (input, expected) => {
    expect(yuanToCents(input)).toBe(expected)
  })

  it.each(['0', '0.00', '100000.01', '1.234', '-5', '1e3', 'abc', ''])(
    'rejects out-of-range or malformed amount %j',
    (input) => {
      expect(yuanToCents(input)).toBeNull()
    }
  )
})

describe('centsToYuanInput', () => {
  it('round-trips stored prices back to the editable yuan value', () => {
    for (const cents of [1, 50, 59900, 59905, 10_000_000]) {
      expect(yuanToCents(centsToYuanInput(cents))).toBe(cents)
    }
  })
})

describe('formatCny', () => {
  it('hides fractional digits for whole-yuan prices and keeps two otherwise', () => {
    expect(formatCny(59900, 'en-US')).toBe('¥599')
    expect(formatCny(59950, 'en-US')).toBe('¥599.50')
  })

  it('stays in CNY regardless of the display locale', () => {
    expect(formatCny(129900, 'zh-CN')).toBe('¥1,299')
  })
})
