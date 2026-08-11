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
import {
  formatMinorCurrency,
  formatQuota,
  isPositiveIntegerString,
} from '@/lib/format'

export { formatMinorCurrency as formatAffiliateMinor }

export function formatAffiliateRate(bps: number): string {
  return `${(bps / 100).toFixed(2)}%`
}

export function formatAffiliateQuota(quota: string | number): string {
  const numericQuota = typeof quota === 'number' ? quota : Number(quota)
  if (!Number.isSafeInteger(numericQuota)) return '-'
  return formatQuota(numericQuota)
}

export function createAffiliateIdempotencyKey(): string {
  return crypto.randomUUID()
}

export function hasPositiveAffiliateInteger(value: string): boolean {
  return isPositiveIntegerString(value)
}

export function absoluteAffiliateLink(link: string): string {
  if (typeof window === 'undefined') return link
  return new URL(link, window.location.origin).toString()
}
