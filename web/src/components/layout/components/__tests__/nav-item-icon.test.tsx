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

import { BadgePercent, Wallet } from 'lucide-react'
import { renderToStaticMarkup } from 'react-dom/server'

import { NavItemIcon } from '../nav-item-icon'

describe('sidebar navigation item icon', () => {
  test('overlays an accessible promotion badge without resizing the icon', () => {
    const markup = renderToStaticMarkup(
      <NavItemIcon
        icon={Wallet}
        badge={{ icon: BadgePercent, label: 'Discount' }}
      />
    )

    assert.match(markup, /relative flex size-4 shrink-0/)
    assert.match(markup, /absolute -top-1 -right-1/)
    assert.match(markup, /rounded-full/)
    assert.match(markup, /<span class="sr-only">Discount<\/span>/)
  })

  test('does not render an overlay when no promotion is configured', () => {
    const markup = renderToStaticMarkup(<NavItemIcon icon={Wallet} />)

    assert.doesNotMatch(markup, /absolute -top-1 -right-1/)
    assert.doesNotMatch(markup, /sr-only/)
  })
})
