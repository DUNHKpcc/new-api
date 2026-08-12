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
  PCC_AGENT_AUTOPLAY_OPTIONS,
  PCC_AGENT_SECTION_CLASSES,
  PCC_AGENT_SLIDE_DIMENSIONS,
  PCC_AGENT_SLIDE_SOURCES,
  PCC_AGENT_STORE_URL,
} from '../pcc-agent-config'

describe('PccAgent home section layout', () => {
  test('automatically advances every three seconds and keeps manual controls active', () => {
    assert.deepEqual(PCC_AGENT_AUTOPLAY_OPTIONS, {
      delay: 3000,
      playOnInit: true,
      stopOnFocusIn: true,
      stopOnInteraction: false,
      stopOnMouseEnter: true,
    })
  })

  test('uses three uniformly generated product screenshots in the carousel', () => {
    assert.deepEqual(PCC_AGENT_SLIDE_SOURCES, [
      '/landing/pcc-agent-marketplace-generated.webp',
      '/landing/pcc-agent-workspace-generated.webp',
      '/landing/pcc-agent-analytics-generated.webp',
    ])

    assert.deepEqual(PCC_AGENT_SLIDE_DIMENSIONS, {
      width: 1536,
      height: 1024,
    })
  })

  test('preserves image proportions across responsive carousel widths', () => {
    const containerClasses = PCC_AGENT_SECTION_CLASSES.container.split(' ')
    const mediaClasses = PCC_AGENT_SECTION_CLASSES.media.split(' ')
    const imageClasses = PCC_AGENT_SECTION_CLASSES.image.split(' ')

    assert.ok(containerClasses.includes('max-w-6xl'))
    assert.ok(mediaClasses.includes('aspect-[3/2]'))
    assert.ok(mediaClasses.includes('w-full'))
    assert.ok(mediaClasses.includes('overflow-hidden'))
    assert.ok(mediaClasses.includes('rounded-lg'))
    assert.ok(imageClasses.includes('h-full'))
    assert.ok(imageClasses.includes('w-full'))
    assert.ok(imageClasses.includes('rounded-[inherit]'))
    assert.ok(imageClasses.includes('object-contain'))
  })

  test('links the section download action to the official store listing', () => {
    assert.equal(
      PCC_AGENT_STORE_URL,
      'https://apps.microsoft.com/detail/9pf5ff13cbhp?hl=zh-CN&gl=CN'
    )
  })
})
