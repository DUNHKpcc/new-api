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

import { renderToStaticMarkup } from 'react-dom/server'

import { SidebarProvider } from '../sidebar'

describe('sidebar layout variables', () => {
  test('allows a theme preset to resize expanded and collapsed navigation', () => {
    const markup = renderToStaticMarkup(
      <SidebarProvider>
        <div>Content</div>
      </SidebarProvider>
    )

    assert.match(markup, /--sidebar-width:var\(--app-sidebar-width, 13rem\)/)
    assert.match(
      markup,
      /--sidebar-width-icon:var\(--app-sidebar-width-icon, 2\.75rem\)/
    )
  })
})
