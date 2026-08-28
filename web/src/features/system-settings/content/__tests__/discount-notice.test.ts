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

import { CONTENT_SECTION_IDS, getContentSectionMeta } from '../section-registry'

describe('content settings navigation', () => {
  test('exposes a dedicated content settings navigation entry', () => {
    assert.equal(CONTENT_SECTION_IDS.includes('discount-notice'), true)
    assert.equal(
      getContentSectionMeta('discount-notice').titleKey,
      'Discount Notice'
    )
  })

  test('exposes version update details as a dedicated content section', () => {
    assert.equal(CONTENT_SECTION_IDS.includes('version-update'), true)
    assert.equal(
      getContentSectionMeta('version-update').titleKey,
      'Version update details'
    )
  })

  test('exposes global notifications as an independent content section', () => {
    assert.equal(CONTENT_SECTION_IDS.includes('global-notifications'), true)
    assert.equal(
      getContentSectionMeta('global-notifications').titleKey,
      'Global Notifications'
    )
  })
})
