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
    assert.ok(tokens(appHeaderLayoutClasses.bar).includes('sm:px-4'))
    assert.ok(tokens(appHeaderLayoutClasses.sidebarTrigger).includes('size-10'))
    assert.ok(tokens(appHeaderLayoutClasses.brand.mark).includes('size-8'))
    assert.ok(tokens(appHeaderLayoutClasses.brand.name).includes('text-xl'))
    assert.ok(tokens(appHeaderLayoutClasses.search).includes('h-10'))
  })

  test('keeps enlarged utilities stable without forcing the search width', () => {
    const actions = tokens(appHeaderLayoutClasses.actions)
    const utilities = tokens(appHeaderLayoutClasses.utilities)

    assert.ok(actions.includes('min-w-0'))
    assert.ok(utilities.includes('shrink-0'))
    assert.ok(utilities.includes('[&>button]:size-10'))
    assert.ok(utilities.includes('[&_[data-slot=avatar]]:size-7'))
    assert.equal(
      appHeaderLayoutClasses.search.includes('[&_button]:size-10'),
      false
    )
  })

  test('gives console navigation full-height touch targets', () => {
    assert.ok(tokens(appHeaderLayoutClasses.topNav.desktop).includes('h-10'))
    assert.ok(tokens(appHeaderLayoutClasses.topNav.link).includes('h-10'))
    assert.ok(tokens(appHeaderLayoutClasses.topNav.link).includes('px-3'))
  })
})
