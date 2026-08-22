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
import type { RankingPeriod } from './types'

export const RANKING_DISPLAY_CONFIG_VERSION = 1
export const MAX_RANKING_ADDED_TOKENS = Number.MAX_SAFE_INTEGER
export const MAX_RANKING_DISPLAY_DATES = 3660
const RANKING_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/

export type RankingDisplayPeriodConfig = {
  adjustments: Record<string, number>
}

export type RankingDisplayDayConfig = {
  adjustments: Record<string, number>
}

export type RankingDisplayConfig = {
  version: typeof RANKING_DISPLAY_CONFIG_VERSION
  enabled: boolean
  periods: Partial<Record<RankingPeriod, RankingDisplayPeriodConfig>>
  daily_records: Record<string, RankingDisplayDayConfig>
}

export function createRankingDisplayConfig(): RankingDisplayConfig {
  return {
    version: RANKING_DISPLAY_CONFIG_VERSION,
    enabled: false,
    periods: {},
    daily_records: {},
  }
}

export function parseRankingDisplayConfig(raw: string): RankingDisplayConfig {
  if (!raw.trim()) return createRankingDisplayConfig()

  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (
      parsed.version !== RANKING_DISPLAY_CONFIG_VERSION ||
      typeof parsed.enabled !== 'boolean' ||
      (parsed.periods !== undefined && !isRecord(parsed.periods)) ||
      (parsed.daily_records !== undefined && !isRecord(parsed.daily_records))
    ) {
      return createRankingDisplayConfig()
    }

    const config = createRankingDisplayConfig()
    config.enabled = parsed.enabled
    if (isRecord(parsed.periods)) {
      for (const period of ['today', 'week', 'month', 'year'] as const) {
        const periodValue = parsed.periods[period]
        if (!isRecord(periodValue) || !isRecord(periodValue.adjustments)) {
          continue
        }

        const adjustments: Record<string, number> = {}
        for (const [modelName, addedTokens] of Object.entries(
          periodValue.adjustments
        )) {
          if (
            modelName.trim() !== modelName ||
            modelName.length === 0 ||
            modelName.length > 64 ||
            typeof addedTokens !== 'number' ||
            !Number.isSafeInteger(addedTokens) ||
            addedTokens < 0
          ) {
            continue
          }
          adjustments[modelName] = addedTokens
        }
        if (Object.keys(adjustments).length > 0) {
          config.periods[period] = { adjustments }
        }
      }
    }
    if (isRecord(parsed.daily_records)) {
      for (const [date, dateValue] of selectLatestDailyRecordEntries(
        parsed.daily_records
      )) {
        if (!isRecord(dateValue) || !isRecord(dateValue.adjustments)) {
          continue
        }

        const adjustments: Record<string, number> = {}
        for (const [modelName, addedTokens] of Object.entries(
          dateValue.adjustments
        )) {
          if (
            modelName.trim() !== modelName ||
            modelName.length === 0 ||
            modelName.length > 64 ||
            typeof addedTokens !== 'number' ||
            !Number.isSafeInteger(addedTokens) ||
            addedTokens < 0
          ) {
            continue
          }
          adjustments[modelName] = addedTokens
        }
        if (Object.keys(adjustments).length > 0) {
          config.daily_records[date] = { adjustments }
        }
      }
    }
    return config
  } catch {
    return createRankingDisplayConfig()
  }
}

export function serializeRankingDisplayConfig(
  config: RankingDisplayConfig
): string {
  const periods: RankingDisplayConfig['periods'] = {}
  for (const period of ['today', 'week', 'month', 'year'] as const) {
    const periodConfig = config.periods[period]
    if (!periodConfig) continue

    const adjustments = Object.fromEntries(
      Object.entries(periodConfig.adjustments).filter(
        ([, addedTokens]) =>
          Number.isSafeInteger(addedTokens) && addedTokens > 0
      )
    )
    if (Object.keys(adjustments).length > 0) {
      periods[period] = { adjustments }
    }
  }

  const dailyRecords: RankingDisplayConfig['daily_records'] = {}
  for (const [date, dateConfig] of selectLatestDailyRecordEntries(
    config.daily_records
  )) {
    const adjustments = Object.fromEntries(
      Object.entries(dateConfig.adjustments).filter(
        ([, addedTokens]) =>
          Number.isSafeInteger(addedTokens) && addedTokens > 0
      )
    )
    if (Object.keys(adjustments).length > 0) {
      dailyRecords[date] = { adjustments }
    }
  }

  return JSON.stringify({
    version: RANKING_DISPLAY_CONFIG_VERSION,
    enabled: config.enabled,
    periods,
    daily_records: dailyRecords,
  })
}

export function getRankingDisplayTotal(
  liveTokens: number,
  addedTokens: number
): number {
  return liveTokens + addedTokens
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function selectLatestDailyRecordEntries<T>(
  records: Record<string, T>
): Array<[string, T]> {
  return Object.entries(records)
    .filter(([date]) => RANKING_DATE_PATTERN.test(date))
    .sort(([left], [right]) => right.localeCompare(left))
    .slice(0, MAX_RANKING_DISPLAY_DATES)
    .sort(([left], [right]) => left.localeCompare(right))
}
