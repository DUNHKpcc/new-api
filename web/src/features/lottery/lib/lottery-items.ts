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
import type { LotteryItem } from '../types'

export const MAX_LOTTERY_ITEMS = 6

export function parseLotteryItems(value: unknown): LotteryItem[] {
  if (!value) return []

  try {
    const parsed =
      typeof value === 'string' ? (JSON.parse(value) as unknown) : value
    if (!Array.isArray(parsed)) return []

    return parsed.filter((item): item is LotteryItem => {
      if (!item || typeof item !== 'object') return false
      const record = item as Record<string, unknown>
      return (
        typeof record.id === 'string' &&
        typeof record.title === 'string' &&
        typeof record.content === 'string' &&
        typeof record.winnerInfo === 'string' &&
        typeof record.image === 'string' &&
        typeof record.publishDate === 'string'
      )
    })
  } catch {
    return []
  }
}

export function toDateTimeLocalValue(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}
