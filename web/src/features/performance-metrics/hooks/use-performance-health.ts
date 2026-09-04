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
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getPerfMetricsSummary } from '../api'
import type { PerfModelSummary } from '../types'

export const PERFORMANCE_WINDOW_HOURS = 24
export const TOP_MODEL_LIMIT = 5

function expandPreviewRates(pattern: number[]): number[] {
  return pattern.flatMap((rate) => [rate, rate, rate, rate, rate, rate])
}

// Design-review fixture. It is reachable only from a development URL with
// `?perfPreview` and is excluded from production builds by the DEV guard.
const PREVIEW_MODELS: PerfModelSummary[] = [
  {
    model_name: 'gpt-5.4',
    avg_latency_ms: 820,
    success_rate: 100,
    avg_tps: 68,
    recent_success_rates: expandPreviewRates([
      100, 100, 100, 100, 99.8, 100, 100, 100, 100, 99.9, 100, 100,
    ]),
    request_count: 12400,
  },
  {
    model_name: 'claude-sonnet-4',
    avg_latency_ms: 1100,
    success_rate: 98.9,
    avg_tps: 55,
    recent_success_rates: expandPreviewRates([
      100, 98, 99, 97, 100, 98, 99, 99, 100, 98, 99, 99,
    ]),
    request_count: 9800,
  },
  {
    model_name: 'gemini-2.5-pro',
    avg_latency_ms: 1360,
    success_rate: 94.2,
    avg_tps: 47,
    recent_success_rates: expandPreviewRates([
      98, 92, 92, 95, 94, 93, 96, 92, 95, 94, 93, 95,
    ]),
    request_count: 7600,
  },
  {
    model_name: 'qwen-max',
    avg_latency_ms: 1520,
    success_rate: 83.5,
    avg_tps: 39,
    recent_success_rates: expandPreviewRates([
      88, 80, 82, 85, 81, 79, 84, 83, 86, 80, 82, 83,
    ]),
    request_count: 5400,
  },
  {
    model_name: 'deepseek-v3',
    avg_latency_ms: 980,
    success_rate: 99.8,
    avg_tps: 62,
    recent_success_rates: expandPreviewRates([
      100, 100, 99, 100, 99, 100, 100, 100, 99, 100, 100, 100,
    ]),
    request_count: 4200,
  },
]

export type StatusLevel = 'operational' | 'minor' | 'degraded' | 'unknown'

export type PerformanceHealthSummary = {
  level: StatusLevel
  successRate: number
  avgLatencyMs: number
  avgTps: number
}

export function getStatusLevel(rate: number): StatusLevel {
  if (!Number.isFinite(rate)) return 'unknown'
  if (rate < 95) return 'degraded'
  if (rate < 99) return 'minor'
  return 'operational'
}

function getPreviewMode(): boolean {
  return (
    import.meta.env.DEV &&
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).has('perfPreview')
  )
}

function buildSummary(models: PerfModelSummary[]): PerformanceHealthSummary {
  let totalRequests = 0
  let successfulRequests = 0
  let latencyTotal = 0
  let latencyWeight = 0
  let throughputTotal = 0
  let throughputWeight = 0

  for (const model of models) {
    const requests = Number(model.request_count)
    const successRate = Number(model.success_rate)
    if (!Number.isFinite(requests) || requests <= 0) continue
    if (!Number.isFinite(successRate)) continue
    totalRequests += requests
    successfulRequests += requests * Math.min(100, Math.max(0, successRate))

    const latency = Number(model.avg_latency_ms)
    if (Number.isFinite(latency) && latency > 0) {
      latencyTotal += latency * requests
      latencyWeight += requests
    }

    const throughput = Number(model.avg_tps)
    if (Number.isFinite(throughput) && throughput > 0) {
      throughputTotal += throughput * requests
      throughputWeight += requests
    }
  }

  const successRate =
    totalRequests > 0 ? successfulRequests / totalRequests : Number.NaN

  const level = getStatusLevel(successRate)

  return {
    level,
    successRate,
    avgLatencyMs: latencyWeight > 0 ? latencyTotal / latencyWeight : Number.NaN,
    avgTps:
      throughputWeight > 0 ? throughputTotal / throughputWeight : Number.NaN,
  }
}

export function usePerformanceHealthData(): {
  models: PerfModelSummary[]
  summary: PerformanceHealthSummary
  loading: boolean
  hasData: boolean
  previewMode: boolean
} {
  const previewMode = getPreviewMode()
  const metricsQuery = useQuery({
    queryKey: ['perf-metrics-summary', PERFORMANCE_WINDOW_HOURS],
    queryFn: () => getPerfMetricsSummary(PERFORMANCE_WINDOW_HOURS),
    staleTime: 60 * 1000,
    retry: false,
    enabled: !previewMode,
  })

  const models = useMemo(
    () =>
      [
        ...(previewMode
          ? PREVIEW_MODELS
          : (metricsQuery.data?.data.models ?? [])),
      ]
        .sort((a, b) => (b.request_count ?? 0) - (a.request_count ?? 0))
        .slice(0, TOP_MODEL_LIMIT),
    [metricsQuery.data, previewMode]
  )

  const summary = useMemo(() => buildSummary(models), [models])

  return {
    models,
    summary,
    loading: metricsQuery.isLoading,
    hasData: models.length > 0,
    previewMode,
  }
}
