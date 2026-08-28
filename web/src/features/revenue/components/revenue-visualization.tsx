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
import { AreaChart, BarChart3, GitFork } from 'lucide-react'
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
import { getDashboardChartColors } from '@/features/dashboard/lib/charts'
import { buildFlowSankeySpec } from '@/features/dashboard/lib/flow'
import { formatMinorCurrency, formatNumber } from '@/lib/format'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import { buildRevenueFlowGraph, buildRevenueTimeline } from '../lib/analytics'
import type {
  ExternalRevenueSummary,
  PlatformRevenueSummary,
  RevenueChartMetric,
  RevenueChartView,
} from '../types'

type RevenueVisualizationProps = {
  currency: string
  platform: PlatformRevenueSummary | undefined
  external: ExternalRevenueSummary | undefined
}

function formatTrendValue(
  datum: Record<string, unknown>,
  metric: RevenueChartMetric,
  currency: string
): string {
  if (metric === 'count') return formatNumber(Number(datum.count) || 0)
  const amountMinor = datum.amountMinor
  return formatMinorCurrency(
    typeof amountMinor === 'string' ? amountMinor : '0',
    currency
  )
}

function formatTrendAxisValue(
  value: number | string,
  metric: RevenueChartMetric,
  currency: string
): string {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) return '--'
  if (metric === 'count') return formatNumber(numericValue)
  return formatMinorCurrency(String(Math.round(numericValue * 100)), currency)
}

export function RevenueVisualization(props: RevenueVisualizationProps) {
  const { t } = useTranslation()
  const [view, setView] = useState<RevenueChartView>('bar')
  const [metric, setMetric] = useState<RevenueChartMetric>('amount')
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
  const trendData = useMemo(() => {
    const sourceLabels = {
      platform: t('Platform orders'),
      external: t('External revenue'),
    }
    return buildRevenueTimeline(
      props.platform,
      props.external,
      props.currency,
      metric
    ).map((item) => ({
      ...item,
      Time: item.date,
      Source: sourceLabels[item.source],
      Value: item.value,
    }))
  }, [metric, props.currency, props.external, props.platform, t])
  const trendSpecs = useMemo(() => {
    const sourceDomain = [t('Platform orders'), t('External revenue')]
    const color = {
      type: 'ordinal',
      domain: sourceDomain,
      range: getDashboardChartColors(sourceDomain.length),
    }
    const axes = [
      {
        orient: 'bottom' as const,
        label: { autoRotate: true, autoHide: true, autoLimit: true },
        tick: { visible: false },
      },
      {
        orient: 'left' as const,
        label: {
          formatMethod: (value: number | string) =>
            formatTrendAxisValue(value, metric, props.currency),
        },
        grid: { visible: true },
      },
    ]
    const tooltip = {
      mark: {
        content: [
          {
            key: (datum: Record<string, unknown>) => String(datum.Source ?? ''),
            value: (datum: Record<string, unknown>) =>
              formatTrendValue(datum, metric, props.currency),
          },
        ],
      },
      dimension: {
        content: [
          {
            key: (datum: Record<string, unknown>) => String(datum.Source ?? ''),
            value: (datum: Record<string, unknown>) =>
              formatTrendValue(datum, metric, props.currency),
          },
        ],
      },
    }
    return {
      bar: {
        type: 'bar' as const,
        data: [{ id: 'revenue-bar', values: trendData }],
        xField: 'Time',
        yField: 'Value',
        seriesField: 'Source',
        stack: true,
        legends: {
          visible: true,
          selectMode: 'single',
          orient: 'top' as const,
        },
        color,
        axes,
        bar: { state: { hover: { stroke: '#000', lineWidth: 1 } } },
        tooltip,
        background: { fill: 'transparent' },
        animation: true,
      },
      area: {
        type: 'area' as const,
        data: [{ id: 'revenue-area', values: trendData }],
        xField: 'Time',
        yField: 'Value',
        seriesField: 'Source',
        stack: false,
        legends: {
          visible: true,
          selectMode: 'single',
          orient: 'top' as const,
        },
        color,
        axes,
        area: {
          style: { fillOpacity: 0.08, curveType: 'monotone' },
        },
        line: { style: { lineWidth: 2, curveType: 'monotone' } },
        point: { visible: false },
        tooltip,
        background: { fill: 'transparent' },
        animation: true,
      },
    }
  }, [metric, props.currency, t, trendData])
  const hasTrendData = trendData.some((item) => item.value > 0)
  const hasFlowData = flow.links.some((link) => link.value > 0)
  const hasData = view === 'flow' ? hasFlowData : hasTrendData
  const chartSpec = view === 'flow' ? flowSpec : trendSpecs[view]
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
          ...chartSpec,
          theme: resolvedTheme === 'dark' ? 'dark' : 'light',
          background: 'transparent',
        }}
        option={VCHART_OPTION}
      />
    )
  }

  return (
    <Card>
      <CardHeader className='grid-cols-1 items-start gap-3 sm:grid-cols-[1fr_auto] sm:items-center'>
        <CardTitle>{t('Revenue analysis')}</CardTitle>
        <div className='flex w-full flex-wrap justify-end gap-2 sm:w-auto'>
          <div
            className='border-border/60 bg-muted/30 flex gap-0.5 rounded-md border p-1'
            data-revenue-metric-toggle='true'
          >
            <Button
              size='sm'
              variant={metric === 'amount' ? 'secondary' : 'ghost'}
              onClick={() => setMetric('amount')}
              aria-pressed={metric === 'amount'}
            >
              {t('Amount')}
            </Button>
            <Button
              size='sm'
              variant={metric === 'count' ? 'secondary' : 'ghost'}
              onClick={() => setMetric('count')}
              aria-pressed={metric === 'count'}
            >
              {t('Orders')}
            </Button>
          </div>
          <div
            className='border-border/60 bg-muted/30 flex gap-0.5 rounded-md border p-1'
            data-revenue-view-toggle='true'
          >
            <Button
              size='icon-sm'
              variant={view === 'bar' ? 'secondary' : 'ghost'}
              onClick={() => setView('bar')}
              aria-pressed={view === 'bar'}
              title={t('Bar Chart')}
              aria-label={t('Bar Chart')}
            >
              <BarChart3 />
            </Button>
            <Button
              size='icon-sm'
              variant={view === 'area' ? 'secondary' : 'ghost'}
              onClick={() => setView('area')}
              aria-pressed={view === 'area'}
              title={t('Area Chart')}
              aria-label={t('Area Chart')}
            >
              <AreaChart />
            </Button>
            <Button
              size='icon-sm'
              variant={view === 'flow' ? 'secondary' : 'ghost'}
              onClick={() => setView('flow')}
              aria-pressed={view === 'flow'}
              title={t('Flow chart')}
              aria-label={t('Flow chart')}
            >
              <GitFork />
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
