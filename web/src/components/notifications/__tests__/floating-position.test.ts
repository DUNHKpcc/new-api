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
  clampNotificationBottomOffset,
  floatingNotificationPosition,
  resolveNotificationBottomOffset,
} from '../floating-position'

describe('floating notification position', () => {
  test('keeps the notification inside the viewport', () => {
    assert.equal(clampNotificationBottomOffset(-20, 64, 844), 12)
    assert.equal(clampNotificationBottomOffset(900, 64, 844), 768)
  })

  test('keeps the preferred position when no floating control overlaps', () => {
    const offset = resolveNotificationBottomOffset({
      preferredOffset: 12,
      widgetHeight: 64,
      viewportHeight: 844,
      blockers: [],
    })

    assert.equal(offset, 12)
  })

  test('moves above a bottom-right floating control', () => {
    const offset = resolveNotificationBottomOffset({
      preferredOffset: 12,
      widgetHeight: 64,
      viewportHeight: 844,
      blockers: [{ top: 768, bottom: 832 }],
    })

    assert.equal(offset, 844 - 768 + floatingNotificationPosition.collisionGap)
  })

  test('moves above a stack of floating controls', () => {
    const offset = resolveNotificationBottomOffset({
      preferredOffset: 12,
      widgetHeight: 64,
      viewportHeight: 844,
      blockers: [
        { top: 768, bottom: 832 },
        { top: 680, bottom: 744 },
      ],
    })

    assert.equal(offset, 844 - 680 + floatingNotificationPosition.collisionGap)
  })
})
