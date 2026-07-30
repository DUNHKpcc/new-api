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

import { globalNotificationCenterLayout } from '../layout'

describe('global notification center layout', () => {
  test('keeps a larger square notification entry at the bottom right', () => {
    const rootClasses = globalNotificationCenterLayout.root.split(' ')
    const triggerClasses = globalNotificationCenterLayout.trigger.split(' ')
    const triggerIconClasses =
      globalNotificationCenterLayout.triggerIcon.split(' ')

    assert.ok(rootClasses.includes('fixed'))
    assert.ok(rootClasses.includes('right-3'))
    assert.ok(rootClasses.includes('sm:right-4'))
    assert.ok(rootClasses.includes('z-[60]'))
    assert.ok(triggerClasses.includes('size-16'))
    assert.ok(triggerClasses.includes('flex-col'))
    assert.ok(triggerClasses.includes('touch-none'))
    assert.ok(triggerClasses.includes('cursor-grab'))
    assert.ok(triggerIconClasses.includes('size-7'))
  })

  test('bounds the unread preview and expanded feed on small screens', () => {
    const panelClasses = globalNotificationCenterLayout.panel.split(' ')
    const previewClasses = globalNotificationCenterLayout.preview.split(' ')
    const feedClasses = globalNotificationCenterLayout.feed.split(' ')

    assert.ok(panelClasses.includes('w-[min(22rem,calc(100vw-5rem))]'))
    assert.ok(previewClasses.includes('w-[min(18rem,calc(100vw-5rem))]'))
    assert.ok(feedClasses.includes('max-h-[min(52svh,24rem)]'))
    assert.ok(feedClasses.includes('overflow-y-auto'))
  })

  test('expands uniformly from the compact button origin', () => {
    assert.equal(globalNotificationCenterLayout.motionOrigin, 'bottom right')
    assert.equal(globalNotificationCenterLayout.motion.initial.scale, 0.86)
    assert.equal(globalNotificationCenterLayout.motion.animate.scale, 1)
  })

  test('finishes the exiting surface before mounting the next surface', () => {
    assert.equal(globalNotificationCenterLayout.presenceMode, 'wait')
  })
})
