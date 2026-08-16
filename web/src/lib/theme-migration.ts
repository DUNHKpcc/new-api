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
import { THEME_COOKIE_KEYS } from './theme-customization'

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
