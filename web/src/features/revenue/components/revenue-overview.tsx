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
  BadgeDollarSign,
  CircleDollarSign,
  Landmark,
  ReceiptText,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

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
import { formatMinorCurrency, formatNumber } from '@/lib/format'

import { getExternalRevenueSummary, getPlatformRevenueSummary } from '../api'
import { combineRevenueTotals, revenueMonthRange } from '../lib/analytics'
import { RevenueMonthSelector } from './revenue-month-selector'
import { RevenueVisualization } from './revenue-visualization'

type MetricCardProps = {
  label: string
  value: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  loading: boolean
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
            <p className='truncate text-xl font-semibold tabular-nums'>
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
  const [month, setMonth] = useState(
    () => new Date(new Date().getFullYear(), new Date().getMonth(), 1)
  )
  const range = useMemo(() => revenueMonthRange(month), [month])
  const platformQuery = useQuery({
    queryKey: ['revenue', 'platform-summary', range.startTime, range.endTime],
    queryFn: () => getPlatformRevenueSummary(range.startTime, range.endTime),
  })
  const externalQuery = useQuery({
    queryKey: ['revenue', 'external-summary', range.startTime, range.endTime],
    queryFn: () => getExternalRevenueSummary(range.startTime, range.endTime),
  })
  const platform = platformQuery.data?.data
  const external = externalQuery.data?.data
  const totals = useMemo(
    () => combineRevenueTotals(platform, external),
    [external, platform]
  )
  const [currency, setCurrency] = useState('CNY')
  useEffect(() => {
    if (totals.length === 0) return
    if (!totals.some((item) => item.currency === currency)) {
      setCurrency(totals[0].currency)
    }
  }, [currency, totals])
  const selected = totals.find((item) => item.currency === currency) ?? {
    currency,
    totalAmountMinor: '0',
    platformAmountMinor: '0',
    externalAmountMinor: '0',
    platformCount: 0,
    externalCount: 0,
  }
  const loading = platformQuery.isLoading || externalQuery.isLoading

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <RevenueMonthSelector month={month} onMonthChange={setMonth} />
        <Select
          value={currency}
          onValueChange={(value) => value && setCurrency(value)}
        >
          <SelectTrigger className='w-28' aria-label={t('Currency')}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(totals.length > 0 ? totals : [{ currency: 'CNY' }]).map(
              (item) => (
                <SelectItem key={item.currency} value={item.currency}>
                  {item.currency}
                </SelectItem>
              )
            )}
          </SelectContent>
        </Select>
      </div>

      {platformQuery.isError || externalQuery.isError ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {t('Failed to load revenue data')}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        <MetricCard
          label={t('Total revenue')}
          value={formatMinorCurrency(selected.totalAmountMinor, currency)}
          description={t('Platform and external revenue')}
          icon={CircleDollarSign}
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
