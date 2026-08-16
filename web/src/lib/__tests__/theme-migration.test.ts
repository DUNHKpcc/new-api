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
import { after, beforeEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import { getCookie, setCookie } from '../cookies'
import { THEME_COOKIE_KEYS } from '../theme-customization'
import {
  CURRENT_THEME_MIGRATION,
  migrateThemeCustomizationCookies,
  THEME_MIGRATION_COOKIE_KEY,
} from '../theme-migration'

const domWindow = new Window({ url: 'http://localhost/' })
const originalDocumentDescriptor = Object.getOwnPropertyDescriptor(
  globalThis,
  'document'
)

Object.defineProperty(globalThis, 'document', {
  configurable: true,
  value: domWindow.document,
})

describe('theme customization migration', () => {
  beforeEach(() => {
    domWindow.document.cookie = `${THEME_MIGRATION_COOKIE_KEY}=; path=/; max-age=0`
    for (const cookieName of Object.values(THEME_COOKIE_KEYS)) {
      domWindow.document.cookie = `${cookieName}=; path=/; max-age=0`
    }
  })

  after(() => {
    if (originalDocumentDescriptor) {
      Object.defineProperty(globalThis, 'document', originalDocumentDescriptor)
    } else {
      delete (globalThis as { document?: Document }).document
    }
    domWindow.close()
  })

  test('resets existing theme cookies to the new system default once', () => {
    setCookie(THEME_COOKIE_KEYS.preset, 'anthropic')
    setCookie(THEME_COOKIE_KEYS.font, 'serif')
    setCookie(THEME_COOKIE_KEYS.radius, 'xl')
    setCookie(THEME_COOKIE_KEYS.scale, 'lg')
    setCookie(THEME_COOKIE_KEYS.contentLayout, 'centered')

    assert.equal(migrateThemeCustomizationCookies(), true)

    for (const cookieName of Object.values(THEME_COOKIE_KEYS)) {
      assert.ok(!getCookie(cookieName))
    }
    assert.equal(getCookie(THEME_MIGRATION_COOKIE_KEY), CURRENT_THEME_MIGRATION)
  })

  test('preserves theme choices made after the migration', () => {
    migrateThemeCustomizationCookies()
    setCookie(THEME_COOKIE_KEYS.preset, 'rose-garden')

    assert.equal(migrateThemeCustomizationCookies(), false)
    assert.equal(getCookie(THEME_COOKIE_KEYS.preset), 'rose-garden')
  })
})
