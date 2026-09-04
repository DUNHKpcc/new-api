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
import { Activity, AlertCircle, CheckCircle2, HeartPulse } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getStatusLevel,
  type StatusLevel,
  usePerformanceHealthData,
} from '@/features/performance-metrics/hooks/use-performance-health'
import {
  formatUptimePct,
  getSuccessRateDotClass,
  getSuccessRateTextClass,
} from '@/features/performance-metrics/lib/format'
import { tileSuccessRates } from '@/features/performance-metrics/lib/status-series'
import type { PerfModelSummary } from '@/features/performance-metrics/types'
import { cn } from '@/lib/utils'

import { PanelWrapper } from '../ui/panel-wrapper'

const MAX_STATUS_SEGMENTS = 72

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

export function SystemStatusPanel() {
  const { t } = useTranslation()
  const { models, loading, hasData } = usePerformanceHealthData()

  return (
    <PanelWrapper
      title={
        <span className='flex items-center gap-2'>
          <IconBadge tone='info' size='sm'>
            <Activity />
          </IconBadge>
          {t('System status')}
        </span>
      }
      description={t('Performance metrics for the last 24 hours')}
      height='h-72'
      contentClassName='p-0'
    >
      <ScrollArea className='h-72' data-system-status-panel='true'>
        <div className='min-w-0'>
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
        </div>
      </ScrollArea>
    </PanelWrapper>
  )
}

function formatRequestCount(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0'
  return new Intl.NumberFormat().format(Math.round(value))
}

function getStatusRates(model: PerfModelSummary): number[] {
  const recentRates = (model.recent_success_rates ?? []).filter((rate) =>
    Number.isFinite(rate)
  )
  if (recentRates.length > 0) {
    return tileSuccessRates(recentRates, MAX_STATUS_SEGMENTS)
  }
  if (!Number.isFinite(model.success_rate)) return []
  return tileSuccessRates([model.success_rate], MAX_STATUS_SEGMENTS)
}

function getStatusSegments(model: PerfModelSummary) {
  const seen = new Map<number, number>()
  return getStatusRates(model).map((rate) => {
    const occurrence = seen.get(rate) ?? 0
    seen.set(rate, occurrence + 1)
    return { rate, key: `${rate}-${occurrence}` }
  })
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
        className='mt-3 flex h-8 min-w-0 items-stretch gap-0.5 overflow-hidden sm:gap-1'
        data-performance-status-bar='true'
        role='img'
        aria-label={`${model.model_name} ${t('Success rate')} ${formatUptimePct(model.success_rate)}`}
      >
        {segments.map(({ rate, key }) => (
          <span
            key={`${model.model_name}-${key}`}
            data-performance-status-segment='true'
            title={`${t('Success rate')}: ${formatUptimePct(rate)}`}
            className={cn(
              'min-w-0 flex-1 rounded-sm',
              getSuccessRateDotClass(rate)
            )}
          />
        ))}
      </div>
    </div>
  )
}
