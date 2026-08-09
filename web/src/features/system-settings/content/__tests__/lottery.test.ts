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

import { parseLotteryItems } from '@/features/lottery/lib/lottery-items'

import { CONTENT_SECTION_IDS, getContentSectionMeta } from '../section-registry'

describe('lottery content settings', () => {
  test('exposes a dedicated console content entry', () => {
    assert.equal(CONTENT_SECTION_IDS.includes('lottery'), true)
    assert.equal(getContentSectionMeta('lottery').titleKey, 'Lottery')
  })

  test('keeps only complete lottery records when reading stored settings', () => {
    const parsed = parseLotteryItems([
      {
        id: 'summer-draw',
        title: 'Summer draw',
        content: 'Join now',
        winnerInfo: '',
        image: 'data:image/webp;base64,UklGRg==',
        publishDate: '2026-08-09T08:00:00Z',
      },
      { id: 'missing-fields' },
    ])

    assert.equal(parsed.length, 1)
    assert.equal(parsed[0]?.id, 'summer-draw')
  })
})
