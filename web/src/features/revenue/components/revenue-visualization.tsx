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
import { VChart } from '@visactor/react-vchart'
import { ChartSpline, GitFork } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { buildFlowSankeySpec } from '@/features/dashboard/lib/flow'
import { formatMinorCurrency, formatNumber } from '@/lib/format'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import { buildRevenueFlowGraph } from '../lib/analytics'
import type { ExternalRevenueSummary, PlatformRevenueSummary } from '../types'

type RevenueVisualizationProps = {
  currency: string
  platform: PlatformRevenueSummary | undefined
  external: ExternalRevenueSummary | undefined
}

type WavePoint = {
  date: string
  source: string
  amount: number
}

export function RevenueVisualization(props: RevenueVisualizationProps) {
  const { t } = useTranslation()
  const [view, setView] = useState<'flow' | 'wave'>('flow')
  const [metric, setMetric] = useState<'amount' | 'count'>('amount')
  const { resolvedTheme, themeReady } = useChartTheme()
  const flow = useMemo(
    () =>
      buildRevenueFlowGraph(
        props.platform,
        props.external,
        props.currency,
        { platform: t('Platform orders'), external: t('External revenue') },
        metric
      ),
    [metric, props.currency, props.external, props.platform, t]
  )
  const flowSpec = useMemo(
    () =>
      buildFlowSankeySpec(
        flow,
        t('Revenue flow'),
        metric === 'count'
          ? (value) => formatNumber(value)
          : (value) =>
              formatMinorCurrency(Math.round(value).toString(), props.currency),
        {
          quota: metric === 'count' ? t('Orders') : t('Revenue'),
          tokens: t('Orders'),
          requests: t('Orders'),
          share: t('Share'),
        },
        {
          tokens: false,
          requests: metric === 'amount',
        }
      ),
    [flow, metric, props.currency, t]
  )
  const waveData = useMemo(() => {
    const points = new Map<string, WavePoint>()
    const platformTimeline = props.platform?.timeline ?? []
    const externalTimeline = props.external?.timeline ?? []
    const addPoint = (date: string, source: string, amountMinor: string) => {
      const key = `${date}\u0000${source}`
      const amount = Number(BigInt(amountMinor)) / 100
      const current = points.get(key)
      if (current) {
        current.amount += amount
      } else {
        points.set(key, { date, source, amount })
      }
    }
    platformTimeline
      .filter((item) => item.currency === props.currency)
      .forEach((item) =>
        addPoint(item.date, t('Platform orders'), item.amount_minor)
      )
    externalTimeline
      .filter((item) => item.currency === props.currency)
      .forEach((item) =>
        addPoint(item.date, t('External revenue'), item.amount_minor)
      )
    return [...points.values()].sort((left, right) =>
      left.date.localeCompare(right.date)
    )
  }, [props.currency, props.external, props.platform, t])
  const waveSpec = useMemo(
    () => ({
      type: 'area' as const,
      data: [{ id: 'revenue', values: waveData }],
      xField: 'date',
      yField: 'amount',
      seriesField: 'source',
      stack: false,
      point: { visible: true, size: 5 },
      line: { style: { lineWidth: 2.5 } },
      area: { style: { fillOpacity: 0.12 } },
      axes: [
        {
          orient: 'bottom' as const,
          label: { autoRotate: true, autoHide: true },
        },
        {
          orient: 'left' as const,
          label: {
            formatMethod: (value: number) =>
              formatMinorCurrency(
                Math.round(value * 100).toString(),
                props.currency
              ),
          },
        },
      ],
      legends: { visible: true, orient: 'top' as const },
      tooltip: { mark: { content: [{ key: 'source', value: 'amount' }] } },
      animation: false,
    }),
    [props.currency, waveData]
  )
  const hasData = view === 'flow' ? flow.links.length > 0 : waveData.length > 0
  let chartContent: ReactNode = null
  if (!hasData) {
    chartContent = (
      <Empty className='h-full border-0'>
        <EmptyHeader>
          <EmptyTitle>{t('No revenue data')}</EmptyTitle>
          <EmptyDescription>
            {t('Revenue data for the selected month will appear here.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  } else if (themeReady) {
    chartContent = (
      <VChart
        key={`${view}-${metric}-${props.currency}-${resolvedTheme}`}
        spec={{
          ...(view === 'flow' ? flowSpec : waveSpec),
          theme: resolvedTheme === 'dark' ? 'dark' : 'light',
          background: 'transparent',
        }}
        option={VCHART_OPTION}
      />
    )
  }

  return (
    <Card>
      <CardHeader className='flex-row items-center justify-between gap-3'>
        <CardTitle>{t('Revenue analysis')}</CardTitle>
        <div className='flex flex-wrap justify-end gap-2'>
          {view === 'flow' ? (
            <div
              className='border-border/60 bg-muted/30 flex gap-0.5 rounded-md border p-1'
              data-revenue-metric-toggle='true'
            >
              <Button
                size='sm'
                variant={metric === 'amount' ? 'secondary' : 'ghost'}
                onClick={() => setMetric('amount')}
              >
                {t('Amount')}
              </Button>
              <Button
                size='sm'
                variant={metric === 'count' ? 'secondary' : 'ghost'}
                onClick={() => setMetric('count')}
              >
                {t('Orders')}
              </Button>
            </div>
          ) : null}
          <div
            className='border-border/60 bg-muted/30 flex gap-0.5 rounded-md border p-1'
            data-revenue-view-toggle='true'
          >
            <Button
              size='icon-sm'
              variant={view === 'flow' ? 'secondary' : 'ghost'}
              onClick={() => setView('flow')}
              title={t('Flow chart')}
              aria-label={t('Flow chart')}
            >
              <GitFork />
            </Button>
            <Button
              size='icon-sm'
              variant={view === 'wave' ? 'secondary' : 'ghost'}
              onClick={() => setView('wave')}
              title={t('Trend chart')}
              aria-label={t('Trend chart')}
            >
              <ChartSpline />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className='h-[320px] sm:h-[420px]'>{chartContent}</div>
      </CardContent>
    </Card>
  )
}
