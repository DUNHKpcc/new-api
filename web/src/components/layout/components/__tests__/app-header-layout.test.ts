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

import { appHeaderLayoutClasses } from '../app-header-layout'

function tokens(classes: string) {
  return classes.split(' ')
}

describe('authenticated app header layout', () => {
  test('matches the enlarged public topbar hierarchy in the console', () => {
    assert.ok(
      tokens(appHeaderLayoutClasses.root).includes(
        'h-[var(--app-header-height,3.75rem)]'
      )
    )
    assert.ok(tokens(appHeaderLayoutClasses.root).includes('border-b'))
    assert.ok(tokens(appHeaderLayoutClasses.root).includes('bg-background/90'))
    assert.ok(tokens(appHeaderLayoutClasses.bar).includes('mx-auto'))
    assert.ok(tokens(appHeaderLayoutClasses.bar).includes('max-w-7xl'))
    assert.ok(tokens(appHeaderLayoutClasses.bar).includes('px-4'))
    assert.ok(tokens(appHeaderLayoutClasses.bar).includes('md:px-6'))
    assert.ok(tokens(appHeaderLayoutClasses.bar).includes('h-full'))
    assert.ok(tokens(appHeaderLayoutClasses.sidebarTrigger).includes('size-10'))
    assert.ok(tokens(appHeaderLayoutClasses.brand.mark).includes('size-8'))
    assert.ok(tokens(appHeaderLayoutClasses.brand.name).includes('text-xl'))
    assert.ok(tokens(appHeaderLayoutClasses.search).includes('size-10'))
    assert.equal(
      tokens(appHeaderLayoutClasses.brand.link).includes('px-2'),
      false
    )
  })

  test('keeps enlarged utilities stable without forcing the search width', () => {
    const actions = tokens(appHeaderLayoutClasses.actions)
    const utilities = tokens(appHeaderLayoutClasses.utilities)

    assert.ok(actions.includes('min-w-0'))
    assert.ok(!actions.includes('ms-auto'))
    assert.ok(utilities.includes('shrink-0'))
    assert.ok(utilities.includes('[&>button]:size-10'))
    assert.ok(utilities.includes('[&_[data-slot=avatar]]:size-7'))
    assert.equal(
      appHeaderLayoutClasses.search.includes('[&_button]:size-10'),
      false
    )
  })

  test('matches the public header three-column alignment', () => {
    const bar = tokens(appHeaderLayoutClasses.bar)
    const left = tokens(appHeaderLayoutClasses.left)
    const nav = tokens(appHeaderLayoutClasses.nav)
    const actions = tokens(appHeaderLayoutClasses.actions)

    assert.ok(bar.includes('grid'))
    assert.ok(bar.includes('grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]'))
    assert.ok(bar.includes('items-center'))
    assert.ok(left.includes('col-start-1'))
    assert.ok(left.includes('row-start-1'))
    assert.ok(nav.includes('col-start-2'))
    assert.ok(nav.includes('row-start-1'))
    assert.ok(nav.includes('justify-self-center'))
    assert.ok(actions.includes('col-start-3'))
    assert.ok(actions.includes('row-start-1'))
    assert.ok(actions.includes('justify-self-end'))
    assert.ok(nav.includes('hidden'))
    assert.ok(nav.includes('lg:flex'))
  })

  test('matches public navigation sizing and active indicator', () => {
    assert.ok(tokens(appHeaderLayoutClasses.topNav.desktop).includes('h-full'))
    assert.ok(tokens(appHeaderLayoutClasses.topNav.desktop).includes('lg:flex'))
    assert.ok(
      tokens(appHeaderLayoutClasses.topNav.compact).includes('lg:hidden')
    )
    assert.ok(tokens(appHeaderLayoutClasses.topNav.link).includes('h-full'))
    assert.ok(tokens(appHeaderLayoutClasses.topNav.link).includes('px-2.5'))
    assert.ok(
      tokens(appHeaderLayoutClasses.topNav.linkActive).includes(
        'after:bg-primary'
      )
    )
  })
})
