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

import { DASHBOARD_PANEL_CLASS_NAME } from '../panel-surface'

describe('dashboard panel surface', () => {
  test('keeps one translucent glass treatment across dashboard panels', () => {
    const classes = DASHBOARD_PANEL_CLASS_NAME.split(' ')

    assert.ok(classes.includes('app-glass-surface'))
    assert.ok(classes.includes('overflow-hidden'))
    assert.ok(classes.includes('rounded-2xl'))
    assert.ok(classes.includes('border'))
    assert.equal(classes.includes('rounded-lg'), false)
  })
})
