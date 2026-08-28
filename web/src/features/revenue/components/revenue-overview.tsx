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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  BadgeDollarSign,
  CircleDollarSign,
  Landmark,
  ReceiptText,
  TrendingDown,
  TrendingUp,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  TimeRangeFilterDialog,
  type TimeRangeFilterValue,
} from '@/components/time-range-filter-dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { getSavedChartPreferences } from '@/features/dashboard/lib'
import { formatMinorCurrency, formatNumber, formatPercent } from '@/lib/format'
import { getRollingDateRange } from '@/lib/time'

import {
  createRevenueCost,
  getExternalRevenueSummary,
  getPlatformRevenueSummary,
  getRevenueCosts,
  updateRevenueCost,
} from '../api'
import {
  calculateRevenueProfit,
  combineRevenueTotals,
  revenueMonthKey,
  sumMinorAmounts,
} from '../lib/analytics'
import type { RevenueCostMutation } from '../types'
import { RevenueCostEditor } from './revenue-cost-editor'
import { RevenueVisualization } from './revenue-visualization'

type MetricCardProps = {
  label: string
  value: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  loading: boolean
  valueClassName?: string
}

function MetricCard(props: MetricCardProps) {
  const Icon = props.icon
  return (
    <Card size='sm'>
      <CardContent className='flex items-start justify-between gap-3'>
        <div className='min-w-0 space-y-1'>
          <p className='text-muted-foreground text-xs font-medium'>
            {props.label}
          </p>
          {props.loading ? (
            <Skeleton className='h-7 w-28' />
          ) : (
            <p
              className={`truncate text-xl font-semibold tabular-nums ${props.valueClassName ?? ''}`}
            >
              {props.value}
            </p>
          )}
          <p className='text-muted-foreground text-xs'>{props.description}</p>
        </div>
        <div className='bg-muted rounded-md p-2'>
          <Icon className='text-muted-foreground h-4 w-4' />
        </div>
      </CardContent>
    </Card>
  )
}

export function RevenueOverview() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const chartPreferences = useMemo(() => getSavedChartPreferences(), [])
  const [filters, setFilters] = useState<TimeRangeFilterValue>(() => {
    const { start, end } = getRollingDateRange(
      chartPreferences.defaultTimeRangeDays
    )
    return {
      start_timestamp: start,
      end_timestamp: end,
      time_granularity: chartPreferences.defaultTimeGranularity,
    }
  })
  const range = useMemo(
    () => ({
      startTime: Math.floor(
        (filters.start_timestamp ?? new Date()).getTime() / 1000
      ),
      endTime: Math.floor(
        (filters.end_timestamp ?? new Date()).getTime() / 1000
      ),
    }),
    [filters.end_timestamp, filters.start_timestamp]
  )
  const platformQuery = useQuery({
    queryKey: ['revenue', 'platform-summary', range.startTime, range.endTime],
    queryFn: () => getPlatformRevenueSummary(range.startTime, range.endTime),
  })
  const externalQuery = useQuery({
    queryKey: ['revenue', 'external-summary', range.startTime, range.endTime],
    queryFn: () => getExternalRevenueSummary(range.startTime, range.endTime),
  })
  const monthKey = revenueMonthKey(filters.start_timestamp ?? new Date())
  const costsQuery = useQuery({
    queryKey: ['revenue', 'costs', monthKey],
    queryFn: () => getRevenueCosts({ month: monthKey, page: 1, pageSize: 100 }),
  })
  const platform = platformQuery.data?.data
  const external = externalQuery.data?.data
  const totals = useMemo(
    () => combineRevenueTotals(platform, external),
    [external, platform]
  )
  const costItems = costsQuery.data?.data?.items
  const availableCurrencies = useMemo(() => {
    const currencies = new Set([
      ...totals.map((item) => item.currency),
      ...(costItems ?? []).map((item) => item.currency),
    ])
    return [...currencies].sort()
  }, [costItems, totals])
  const [currency, setCurrency] = useState('CNY')
  useEffect(() => {
    if (availableCurrencies.length === 0) return
    if (!availableCurrencies.includes(currency)) {
      setCurrency(availableCurrencies[0])
    }
  }, [availableCurrencies, currency])
  const selected = totals.find((item) => item.currency === currency) ?? {
    currency,
    totalAmountMinor: '0',
    platformAmountMinor: '0',
    externalAmountMinor: '0',
    platformCount: 0,
    externalCount: 0,
  }
  const costRecords = useMemo(
    () => (costItems ?? []).filter((item) => item.currency === currency),
    [costItems, currency]
  )
  const costMinor = useMemo(
    () => sumMinorAmounts(costRecords.map((item) => item.amount_minor)),
    [costRecords]
  )
  const createCostMutation = useMutation({
    mutationFn: createRevenueCost,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['revenue', 'costs'] })
    },
  })
  const updateCostMutation = useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: number
      payload: RevenueCostMutation
    }) => updateRevenueCost(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['revenue', 'costs'] })
    },
  })
  const profit = useMemo(
    () => calculateRevenueProfit(selected.totalAmountMinor, costMinor),
    [costMinor, selected.totalAmountMinor]
  )
  const netIsNegative = profit.netAmountMinor.startsWith('-')
  const loading = platformQuery.isLoading || externalQuery.isLoading

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <TimeRangeFilterDialog
          currentFilters={filters}
          defaultTimeRangeDays={chartPreferences.defaultTimeRangeDays}
          defaultTimeGranularity={chartPreferences.defaultTimeGranularity}
          onFilterChange={setFilters}
          onReset={() => {
            const { start, end } = getRollingDateRange(
              chartPreferences.defaultTimeRangeDays
            )
            setFilters({
              start_timestamp: start,
              end_timestamp: end,
              time_granularity: chartPreferences.defaultTimeGranularity,
            })
          }}
          titleKey='Revenue management'
          descriptionKey='Filter revenue data by time range.'
          showGranularity={false}
        />
        <Select
          value={currency}
          onValueChange={(value) => value && setCurrency(value)}
        >
          <SelectTrigger className='w-28' aria-label={t('Currency')}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(availableCurrencies.length > 0
              ? availableCurrencies
              : ['CNY']
            ).map((item) => (
              <SelectItem key={item} value={item}>
                {item}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {platformQuery.isError || externalQuery.isError || costsQuery.isError ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {costsQuery.isError
              ? t('Failed to load cost records')
              : t('Failed to load revenue data')}
          </AlertDescription>
        </Alert>
      ) : null}

      <RevenueCostEditor
        month={monthKey}
        currency={currency}
        records={costRecords}
        totalAmountMinor={costMinor}
        loading={costsQuery.isLoading}
        error={costsQuery.isError}
        onSaved={async (payload, record) => {
          if (record) {
            await updateCostMutation.mutateAsync({ id: record.id, payload })
            return
          }
          await createCostMutation.mutateAsync(payload)
        }}
      />

      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-7'>
        <MetricCard
          label={t('Total revenue')}
          value={formatMinorCurrency(selected.totalAmountMinor, currency)}
          description={t('Platform and external revenue')}
          icon={CircleDollarSign}
          loading={loading}
        />
        <MetricCard
          label={t('Total cost')}
          value={formatMinorCurrency(profit.costAmountMinor, currency)}
          description={t('Recorded costs by category')}
          icon={ReceiptText}
          loading={costsQuery.isLoading}
        />
        <MetricCard
          label={t('Net income')}
          value={formatMinorCurrency(profit.netAmountMinor, currency)}
          description={t('Total revenue minus total cost')}
          icon={netIsNegative ? TrendingDown : TrendingUp}
          valueClassName={
            netIsNegative
              ? 'text-destructive'
              : 'text-emerald-600 dark:text-emerald-400'
          }
          loading={loading}
        />
        <MetricCard
          label={t('Profit margin')}
          value={
            profit.marginPercent === null
              ? '-'
              : formatPercent(profit.marginPercent)
          }
          description={t('Net income as a share of revenue')}
          icon={TrendingUp}
          loading={loading}
        />
        <MetricCard
          label={t('Platform revenue')}
          value={formatMinorCurrency(selected.platformAmountMinor, currency)}
          description={t('{{count}} successful orders', {
            count: formatNumber(selected.platformCount),
          })}
          icon={Landmark}
          loading={loading}
        />
        <MetricCard
          label={t('External revenue')}
          value={formatMinorCurrency(selected.externalAmountMinor, currency)}
          description={t('{{count}} active records', {
            count: formatNumber(selected.externalCount),
          })}
          icon={BadgeDollarSign}
          loading={loading}
        />
        <MetricCard
          label={t('Revenue entries')}
          value={formatNumber(selected.platformCount + selected.externalCount)}
          description={t('Successful orders plus active records')}
          icon={ReceiptText}
          loading={loading}
        />
      </div>

      <RevenueVisualization
        currency={currency}
        platform={platform}
        external={external}
      />

      <Alert>
        <AlertDescription>
          {t(
            'External revenue participates in reports only. It never changes platform orders, settlements, quotas, or user balances.'
          )}
        </AlertDescription>
      </Alert>
    </div>
  )
}
