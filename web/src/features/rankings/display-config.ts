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

export type RankingDisplayPeriodConfig = {
  adjustments: Record<string, number>
}

export type RankingDisplayConfig = {
  version: typeof RANKING_DISPLAY_CONFIG_VERSION
  enabled: boolean
  periods: Partial<Record<RankingPeriod, RankingDisplayPeriodConfig>>
}

export function createRankingDisplayConfig(): RankingDisplayConfig {
  return {
    version: RANKING_DISPLAY_CONFIG_VERSION,
    enabled: false,
    periods: {},
  }
}

export function parseRankingDisplayConfig(raw: string): RankingDisplayConfig {
  if (!raw.trim()) return createRankingDisplayConfig()

  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (
      parsed.version !== RANKING_DISPLAY_CONFIG_VERSION ||
      typeof parsed.enabled !== 'boolean' ||
      !isRecord(parsed.periods)
    ) {
      return createRankingDisplayConfig()
    }

    const config = createRankingDisplayConfig()
    config.enabled = parsed.enabled
    for (const period of ['today', 'week', 'month', 'year'] as const) {
      const periodValue = parsed.periods[period]
      if (!isRecord(periodValue) || !isRecord(periodValue.adjustments)) continue

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

  return JSON.stringify({
    version: RANKING_DISPLAY_CONFIG_VERSION,
    enabled: config.enabled,
    periods,
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
