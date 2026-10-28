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
const MAX_PRICE_CENTS = 10_000_000

/**
 * Guardian services are always priced in CNY regardless of the site's quota
 * display currency, so the shared currency helpers (which follow the site
 * configuration) do not apply here.
 */
export function formatCny(cents: number, locale: Intl.LocalesArgument): string {
  if (!Number.isFinite(cents)) return '-'
  const hasFraction = cents % 100 !== 0
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: 'CNY',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: hasFraction ? 2 : 0,
    maximumFractionDigits: 2,
  }).format(cents / 100)
}

/** Parses a yuan amount typed by an administrator into whole fen. */
export function yuanToCents(value: string): number | null {
  const match = /^(\d{1,6})(?:\.(\d{1,2}))?$/.exec(value.trim())
  if (!match) return null
  const cents = Number(match[1]) * 100 + Number((match[2] ?? '').padEnd(2, '0'))
  if (cents < 1 || cents > MAX_PRICE_CENTS) return null
  return cents
}

export function centsToYuanInput(cents: number): string {
  const whole = Math.trunc(cents / 100)
  const fraction = cents % 100
  return fraction === 0
    ? String(whole)
    : `${whole}.${String(fraction).padStart(2, '0')}`
}
