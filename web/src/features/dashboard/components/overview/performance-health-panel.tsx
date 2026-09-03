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
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Gauge,
  HeartPulse,
  Timer,
} from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { getPerfMetricsSummary } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
  getSuccessRateDotClass,
  getSuccessRateTextClass,
} from '@/features/performance-metrics/lib/format'
import type { PerfModelSummary } from '@/features/performance-metrics/types'
import { cn } from '@/lib/utils'

import { DASHBOARD_PANEL_CLASS_NAME } from '../ui/panel-surface'

const PERFORMANCE_WINDOW_HOURS = 24
const TOP_MODEL_LIMIT = 5
const MAX_STATUS_SEGMENTS = 72

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

type StatusLevel = 'operational' | 'minor' | 'degraded' | 'unknown'

function getStatusLevel(rate: number): StatusLevel {
  if (!Number.isFinite(rate)) return 'unknown'
  if (rate < 95) return 'degraded'
  if (rate < 99) return 'minor'
  return 'operational'
}

function getStatusIcon(level: StatusLevel) {
  if (level === 'operational') return CheckCircle2
  if (level === 'minor') return Activity
  if (level === 'degraded') return AlertCircle
  return HeartPulse
}

function getStatusTone(level: StatusLevel): IconBadgeTone {
  if (level === 'operational') return 'success'
  if (level === 'minor') return 'warning'
  if (level === 'degraded') return 'destructive'
  return 'neutral'
}

function getStatusBackground(level: StatusLevel): string {
  if (level === 'operational') return 'bg-success/10'
  if (level === 'minor') return 'bg-warning/10'
  if (level === 'degraded') return 'bg-destructive/10'
  return 'bg-muted/60'
}

function formatRequestCount(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0'
  return new Intl.NumberFormat().format(Math.round(value))
}

function getStatusRates(model: PerfModelSummary): number[] {
  const recentRates = (model.recent_success_rates ?? []).filter((rate) =>
    Number.isFinite(rate)
  )
  return recentRates.length > 0
    ? recentRates.slice(-MAX_STATUS_SEGMENTS)
    : [model.success_rate]
}

function getStatusSegments(model: PerfModelSummary) {
  const seen = new Map<number, number>()
  return getStatusRates(model).map((rate) => {
    const occurrence = seen.get(rate) ?? 0
    seen.set(rate, occurrence + 1)
    return { rate, key: `${rate}-${occurrence}` }
  })
}

function getStatusCopy(
  level: StatusLevel,
  t: (key: string) => string,
  loading: boolean
): { label: string; description: string } {
  if (level === 'unknown') {
    return {
      label: loading ? t('Loading') : t('No performance data available'),
      description: t('No performance data available'),
    }
  }
  if (level === 'operational') {
    return {
      label: t('All systems operational'),
      description: t('No current issues detected in the leading models.'),
    }
  }
  if (level === 'minor') {
    return {
      label: t('Minor blips in the leading models'),
      description: t(
        'Some recent calls were unsuccessful, but service remains available.'
      ),
    }
  }
  return {
    label: t('Degraded performance recently'),
    description: t(
      'Recent model calls show a lower than expected success rate.'
    ),
  }
}

export function PerformanceHealthPanel() {
  const { t } = useTranslation()
  const previewMode =
    import.meta.env.DEV &&
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).has('perfPreview')
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

  const summary = useMemo(() => {
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
      avgLatencyMs:
        latencyWeight > 0 ? latencyTotal / latencyWeight : Number.NaN,
      avgTps:
        throughputWeight > 0 ? throughputTotal / throughputWeight : Number.NaN,
    }
  }, [models])

  const loading = metricsQuery.isLoading
  const hasData = models.length > 0
  const statusCopy = getStatusCopy(summary.level, t, loading)
  const StatusIcon = getStatusIcon(summary.level)
  const statusTone = getStatusTone(summary.level)

  return (
    <section className={cn(DASHBOARD_PANEL_CLASS_NAME, 'overflow-hidden')}>
      <div className='flex items-center gap-2 border-b px-4 py-3 sm:px-5'>
        <IconBadge tone='success' size='sm'>
          <HeartPulse />
        </IconBadge>
        <h3 className='text-sm font-semibold'>{t('Performance health')}</h3>
        <span className='text-muted-foreground ml-auto text-xs'>
          {t('Performance metrics for the last 24 hours')}
        </span>
      </div>

      <div className='grid grid-cols-1 gap-2 border-b p-4 sm:grid-cols-3 sm:p-5'>
        <MetricCell
          icon={HeartPulse}
          label={t('Success rate')}
          value={formatUptimePct(summary.successRate)}
          loading={loading}
          valueClassName={getSuccessRateTextClass(summary.successRate)}
          tone='success'
        />
        <MetricCell
          icon={Timer}
          label={t('Average latency')}
          value={formatLatency(summary.avgLatencyMs)}
          loading={loading}
          tone='warning'
        />
        <MetricCell
          icon={Gauge}
          label={t('Throughput')}
          value={formatThroughput(summary.avgTps)}
          loading={loading}
          tone='info'
        />
      </div>

      <div
        className={cn(
          'flex flex-col gap-1 border-b px-4 py-3 sm:px-5',
          getStatusBackground(summary.level)
        )}
      >
        <div className='flex items-center gap-2'>
          <IconBadge tone={statusTone} size='sm'>
            <StatusIcon />
          </IconBadge>
          <h3 className='text-base font-semibold'>{t('System status')}</h3>
          {previewMode && (
            <span className='text-muted-foreground text-[11px] font-medium uppercase'>
              {t('Preview')}
            </span>
          )}
          {!loading && hasData && (
            <span
              className={cn(
                'ml-auto font-mono text-sm font-semibold tabular-nums',
                getSuccessRateTextClass(summary.successRate)
              )}
            >
              {formatUptimePct(summary.successRate)}
            </span>
          )}
        </div>
        <p className='text-sm font-medium'>
          {loading ? t('Loading') : statusCopy.label}
        </p>
        <p className='text-muted-foreground text-xs'>
          {statusCopy.description}
        </p>
      </div>

      <div className='border-b px-4 py-3 sm:px-5'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='info' size='xs'>
              <HeartPulse />
            </IconBadge>
            <span className='text-sm font-semibold'>
              {t('Model call status')}
            </span>
          </div>
          <span className='text-muted-foreground text-xs'>
            {t('Top {{count}} models by usage in the last 24 hours', {
              count: TOP_MODEL_LIMIT,
            })}
          </span>
        </div>
      </div>

      {loading && (
        <div className='divide-y'>
          {['one', 'two', 'three', 'four', 'five'].map((key) => (
            <div key={key} className='space-y-3 px-4 py-4 sm:px-5'>
              <div className='flex items-center justify-between gap-3'>
                <Skeleton className='h-4 w-44' />
                <Skeleton className='h-4 w-16' />
              </div>
              <Skeleton className='h-3 w-full' />
            </div>
          ))}
        </div>
      )}
      {!loading && !hasData && (
        <div className='text-muted-foreground px-4 py-8 text-center text-sm sm:px-5'>
          {t('No performance data available')}
        </div>
      )}
      {!loading && hasData && (
        <div className='divide-y'>
          {models.map((model) => (
            <ModelStatusRow key={model.model_name} model={model} />
          ))}
        </div>
      )}
    </section>
  )
}

function ModelStatusRow(props: { model: PerfModelSummary }) {
  const { t } = useTranslation()
  const model = props.model
  const segments = getStatusSegments(model)
  const level = getStatusLevel(model.success_rate)
  const StatusIcon = getStatusIcon(level)
  const iconTone = getStatusTone(level)

  return (
    <div className='px-4 py-4 sm:px-5'>
      <div className='flex flex-wrap items-center gap-x-3 gap-y-1.5'>
        <IconBadge tone={iconTone} size='xs'>
          <StatusIcon />
        </IconBadge>
        <span className='min-w-0 flex-1 truncate font-mono text-sm font-semibold'>
          {model.model_name}
        </span>
        <span className='text-muted-foreground text-xs tabular-nums'>
          {formatRequestCount(Number(model.request_count))} {t('Requests')}
        </span>
        <span
          className={cn(
            'font-mono text-sm font-semibold tabular-nums',
            getSuccessRateTextClass(model.success_rate)
          )}
        >
          {formatUptimePct(model.success_rate)}
        </span>
      </div>

      <div
        className='mt-3 flex h-8 items-stretch gap-0.5 overflow-hidden sm:gap-1'
        role='img'
        aria-label={`${model.model_name} ${t('Success rate')} ${formatUptimePct(model.success_rate)}`}
      >
        {segments.map(({ rate, key }) => (
          <span
            key={`${model.model_name}-${key}`}
            title={`${t('Success rate')}: ${formatUptimePct(rate)}`}
            className={cn(
              'w-1.5 shrink-0 rounded-sm sm:w-2',
              getSuccessRateDotClass(rate)
            )}
          />
        ))}
      </div>
    </div>
  )
}

function MetricCell(props: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  loading: boolean
  valueClassName?: string
  tone: IconBadgeTone
}) {
  const Icon = props.icon

  return (
    <div className={cn('px-3 py-3 sm:px-4', 'bg-muted/40')}>
      <div className='text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium'>
        <IconBadge tone={props.tone} size='xs'>
          <Icon />
        </IconBadge>
        <span className='truncate'>{props.label}</span>
      </div>
      {props.loading ? (
        <Skeleton className='mt-1.5 h-5 w-16 rounded' />
      ) : (
        <div
          className={cn(
            'mt-1.5 font-mono text-sm font-semibold tabular-nums',
            props.valueClassName
          )}
        >
          {props.value}
        </div>
      )}
    </div>
  )
}
