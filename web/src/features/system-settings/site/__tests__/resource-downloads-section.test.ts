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
  isHttpDownloadUrl,
  parseResourceDownloadItems,
} from '@/features/resource-downloads/lib/resource-downloads'

import { SITE_SECTION_IDS, getSiteSectionMeta } from '../section-registry'

describe('resource download settings', () => {
  test('exposes a dedicated site settings navigation entry', () => {
    assert.equal(SITE_SECTION_IDS.includes('resource-downloads'), true)
    assert.equal(
      getSiteSectionMeta('resource-downloads').titleKey,
      'Resource Downloads'
    )
  })

  test('keeps only complete resource records when reading stored settings', () => {
    const parsed = parseResourceDownloadItems(
      JSON.stringify([
        {
          id: 'valid',
          name: 'Valid resource',
          description: '',
          url: 'https://example.com/download',
          thumbnail: 'data:image/webp;base64,UklGRg==',
        },
        { id: 'missing-fields' },
      ])
    )

    assert.equal(parsed.length, 1)
    assert.equal(parsed[0]?.id, 'valid')
  })

  test('accepts only credential-free HTTP download URLs', () => {
    assert.equal(isHttpDownloadUrl('https://example.com/download'), true)
    assert.equal(isHttpDownloadUrl('javascript:alert(1)'), false)
    assert.equal(isHttpDownloadUrl('https://user:pass@example.com/file'), false)
  })
})
