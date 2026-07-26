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
  heroProductShowcaseClasses,
  heroProductShowcaseMedia,
} from '../hero-product-showcase-config'

describe('home hero product showcase', () => {
  test('hides the three-product montage below the supported viewport width', () => {
    const stageClasses = heroProductShowcaseClasses.stage.split(' ')

    assert.ok(stageClasses.includes('hidden'))
    assert.ok(stageClasses.includes('min-[700px]:block'))
  })

  test('preserves each supplied image aspect ratio while scaling responsively', () => {
    const imageClasses = heroProductShowcaseClasses.image.split(' ')
    const frameClasses = heroProductShowcaseClasses.imageFrame.split(' ')

    assert.ok(imageClasses.includes('w-full'))
    assert.ok(imageClasses.includes('h-auto'))
    assert.ok(imageClasses.includes('object-contain'))
    assert.ok(frameClasses.includes('rounded-[8px]'))
  })

  test('presents desktop, web, and WeChat mini-program surfaces', () => {
    assert.deepEqual(heroProductShowcaseMedia.map((item) => item.id).sort(), [
      'desktop',
      'web',
      'wechat-mini-program',
    ])
  })

  test('links the desktop preview to the PccAgent Microsoft Store page', () => {
    const desktop = heroProductShowcaseMedia.find(
      (item) => item.id === 'desktop'
    )

    assert.equal(
      desktop?.href,
      'https://apps.microsoft.com/detail/9pf5ff13cbhp?hl=zh-CN&gl=CN'
    )
    assert.equal(desktop?.storeIconSrc, '/landing/microsoft-store.png')
  })

  test('keeps the web title visible and the mini-program label above its image', () => {
    const web = heroProductShowcaseMedia.find((item) => item.id === 'web')
    const desktop = heroProductShowcaseMedia.find(
      (item) => item.id === 'desktop'
    )
    const miniProgram = heroProductShowcaseMedia.find(
      (item) => item.id === 'wechat-mini-program'
    )

    assert.ok(web?.positionClassName.includes('-top-[35px]'))
    assert.ok(web?.positionClassName.includes('left-1/2'))
    assert.ok(web?.positionClassName.includes('w-[66%]'))
    assert.ok(desktop?.labelClassName.includes('right-3'))
    assert.ok(miniProgram?.labelClassName.includes('-top-10'))
  })
})
