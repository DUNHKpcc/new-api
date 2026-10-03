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
import { getCookie, removeCookie, setCookie } from './cookies'
import {
  CONTENT_LAYOUT_VALUES,
  THEME_COOKIE_KEYS,
  THEME_FONT_VALUES,
  THEME_PRESET_VALUES,
  THEME_RADIUS_VALUES,
  THEME_SCALE_VALUES,
} from './theme-customization'
import { THEME_STORAGE_KEYS } from './theme-storage'

export const THEME_MIGRATION_COOKIE_KEY = 'theme_default_migration'
export const CURRENT_THEME_MIGRATION = 'operator-v1'

const MIGRATION_COOKIE_MAX_AGE = 60 * 60 * 24 * 365 * 10 // 10 years

/**
 * Applies a versioned, one-time reset so existing browsers receive the new
 * system theme without overriding choices made after the migration.
 */
export function migrateThemeCustomizationCookies(): boolean {
  if (getCookie(THEME_MIGRATION_COOKIE_KEY) === CURRENT_THEME_MIGRATION) {
    return false
  }

  for (const cookieName of Object.values(THEME_COOKIE_KEYS)) {
    removeCookie(cookieName)
  }
  setCookie(
    THEME_MIGRATION_COOKIE_KEY,
    CURRENT_THEME_MIGRATION,
    MIGRATION_COOKIE_MAX_AGE
  )
  return true
}

export const THEME_COOKIE_MIGRATION_STORAGE_KEY =
  'newapi:theme:v1:cookie-migration'

/**
 * Preserve choices made after the fork's default-theme migration exactly once
 * per origin. Unmarked legacy cookies still receive the fork's new defaults;
 * current origin preferences always take precedence over shared cookies.
 */
export function migrateThemeCustomizationPreferences(): void {
  if (typeof window === 'undefined') return

  try {
    if (
      window.localStorage.getItem(THEME_COOKIE_MIGRATION_STORAGE_KEY) ===
      CURRENT_THEME_MIGRATION
    ) {
      return
    }

    if (getCookie(THEME_MIGRATION_COOKIE_KEY) === CURRENT_THEME_MIGRATION) {
      const preferences: readonly [string, string, ReadonlySet<string>][] = [
        [
          'vite-ui-theme',
          THEME_STORAGE_KEYS.mode,
          new Set(['light', 'dark', 'system']),
        ],
        [
          THEME_COOKIE_KEYS.preset,
          THEME_STORAGE_KEYS.preset,
          THEME_PRESET_VALUES,
        ],
        [THEME_COOKIE_KEYS.font, THEME_STORAGE_KEYS.font, THEME_FONT_VALUES],
        [
          THEME_COOKIE_KEYS.radius,
          THEME_STORAGE_KEYS.radius,
          THEME_RADIUS_VALUES,
        ],
        [THEME_COOKIE_KEYS.scale, THEME_STORAGE_KEYS.scale, THEME_SCALE_VALUES],
        [
          THEME_COOKIE_KEYS.contentLayout,
          THEME_STORAGE_KEYS.contentLayout,
          CONTENT_LAYOUT_VALUES,
        ],
      ]
      for (const [cookie, storage, allowed] of preferences) {
        const value = getCookie(cookie)
        if (
          value &&
          allowed.has(value) &&
          window.localStorage.getItem(storage) === null
        ) {
          window.localStorage.setItem(storage, value)
        }
      }
    }
    window.localStorage.setItem(
      THEME_COOKIE_MIGRATION_STORAGE_KEY,
      CURRENT_THEME_MIGRATION
    )
    migrateThemeCustomizationCookies()
  } catch {
    // Preserve cookie choices for a later attempt when storage is available.
  }
}
