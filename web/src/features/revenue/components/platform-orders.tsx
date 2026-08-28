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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, Loader2, Search } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge } from '@/components/status-badge'
import {
  TimeRangeFilterDialog,
  type TimeRangeFilterValue,
} from '@/components/time-range-filter-dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getSavedChartPreferences } from '@/features/dashboard/lib'
import {
  completeOrder,
  getAllBillingHistory,
  isApiSuccess,
} from '@/features/wallet/api'
import {
  formatTimestamp,
  getPaymentMethodName,
  getStatusConfig,
} from '@/features/wallet/lib/billing'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { getRollingDateRange } from '@/lib/time'

import {
  getOrderUser,
  isSubscriptionOrderTradeNo,
  loadPlatformOrderContext,
  resolveOrderSubscription,
} from '../lib/platform-orders'

export function PlatformOrders() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
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
  const [confirmTradeNo, setConfirmTradeNo] = useState<string | null>(null)
  const [completing, setCompleting] = useState(false)
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
  const query = useQuery({
    queryKey: [
      'revenue',
      'platform-orders',
      page,
      keyword,
      range.startTime,
      range.endTime,
    ],
    queryFn: () =>
      getAllBillingHistory(page, 20, keyword, {
        startTime: range.startTime,
        endTime: range.endTime,
      }),
    placeholderData: keepPreviousData,
  })
  const queryItems = query.data?.data?.items
  const records = useMemo(() => queryItems ?? [], [queryItems])
  const total = query.data?.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / 20))
  const contextUserIds = useMemo(
    () => [...new Set(records.map((record) => record.user_id))].sort(),
    [records]
  )
  const contextSubscriptionUserIds = useMemo(
    () =>
      [
        ...new Set(
          records
            .filter((record) => isSubscriptionOrderTradeNo(record.trade_no))
            .map((record) => record.user_id)
        ),
      ].sort(),
    [records]
  )
  const orderContextQuery = useQuery({
    queryKey: [
      'revenue',
      'platform-order-context',
      contextUserIds,
      contextSubscriptionUserIds,
    ],
    queryFn: () => loadPlatformOrderContext(records),
    enabled: records.length > 0,
    staleTime: 5 * 60 * 1000,
  })
  const orderContext = orderContextQuery.data

  const handleConfirmComplete = async () => {
    const tradeNo = confirmTradeNo
    if (!tradeNo) return

    setCompleting(true)
    try {
      const response = await completeOrder({ trade_no: tradeNo })
      if (isApiSuccess(response)) {
        toast.success(t('Order completed successfully'))
        setConfirmTradeNo(null)
        await query.refetch()
      } else {
        toast.error(response.message || t('Failed to complete order'))
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setCompleting(false)
    }
  }

  let tableContent: ReactNode
  if (query.isLoading) {
    tableContent = (
      <div className='space-y-2 p-4'>
        {['one', 'two', 'three', 'four', 'five', 'six'].map((key) => (
          <Skeleton key={key} className='h-12 w-full' />
        ))}
      </div>
    )
  } else if (query.isError) {
    tableContent = (
      <div className='text-destructive p-8 text-center text-sm'>
        {t('Failed to load platform orders')}
      </div>
    )
  } else if (records.length === 0) {
    tableContent = (
      <div className='text-muted-foreground p-12 text-center text-sm'>
        {t('No billing records found')}
      </div>
    )
  } else {
    tableContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Order number')}</TableHead>
            <TableHead>{t('User ID')}</TableHead>
            <TableHead>{t('Username')}</TableHead>
            <TableHead>{t('Subscription')}</TableHead>
            <TableHead>{t('Payment Method')}</TableHead>
            <TableHead>{t('Amount')}</TableHead>
            <TableHead>{t('Payment')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Completed at')}</TableHead>
            <TableHead>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {records.map((record) => {
            const statusConfig = getStatusConfig(record.status)
            const user = getOrderUser(record, orderContext)
            const subscription = resolveOrderSubscription(record, orderContext)
            const username = user?.username?.trim()
            const displayName = user?.display_name?.trim()
            const subscriptionTitle =
              subscription?.title === 'Subscription order'
                ? t('Subscription order')
                : subscription?.title
            return (
              <TableRow key={record.id}>
                <TableCell>
                  <div className='max-w-56 truncate font-mono text-xs'>
                    {record.trade_no}
                  </div>
                </TableCell>
                <TableCell>{record.user_id}</TableCell>
                <TableCell>
                  <div className='min-w-32'>
                    <div className='max-w-44 truncate font-medium'>
                      {username || displayName || `#${record.user_id}`}
                    </div>
                    {username && displayName && username !== displayName ? (
                      <div className='text-muted-foreground max-w-44 truncate text-xs'>
                        {displayName}
                      </div>
                    ) : null}
                    <div className='text-muted-foreground text-xs'>
                      #{record.user_id}
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  {subscriptionTitle ? (
                    <div className='min-w-36'>
                      <div className='max-w-52 truncate font-medium'>
                        {subscriptionTitle}
                      </div>
                      {subscription?.planId ? (
                        <div className='text-muted-foreground text-xs'>
                          #{subscription.planId}
                        </div>
                      ) : null}
                    </div>
                  ) : (
                    '-'
                  )}
                </TableCell>
                <TableCell>
                  {getPaymentMethodName(record.payment_method, t)}
                </TableCell>
                <TableCell className='font-semibold'>
                  {formatCurrencyFromUSD(record.amount, {
                    digitsLarge: 2,
                    digitsSmall: 2,
                    abbreviate: false,
                  })}
                </TableCell>
                <TableCell className='font-semibold'>
                  {formatNumber(record.money)}
                </TableCell>
                <TableCell>
                  <StatusBadge
                    label={t(statusConfig.label)}
                    variant={statusConfig.variant}
                    showDot
                  />
                </TableCell>
                <TableCell>
                  {record.complete_time
                    ? formatTimestamp(record.complete_time)
                    : '-'}
                </TableCell>
                <TableCell>
                  {record.status === 'pending' ? (
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      onClick={() => setConfirmTradeNo(record.trade_no)}
                      disabled={completing}
                    >
                      {completing && confirmTradeNo === record.trade_no ? (
                        <Loader2 className='animate-spin' />
                      ) : null}
                      {t('Complete Order')}
                    </Button>
                  ) : (
                    '-'
                  )}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <form
          className='relative min-w-64 flex-1 sm:max-w-xl'
          onSubmit={(event) => {
            event.preventDefault()
            setPage(1)
            setKeyword(searchInput.trim())
          }}
        >
          <Search className='text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
          <Input
            className='pl-9'
            value={searchInput}
            onChange={(event) => setSearchInput(event.currentTarget.value)}
            placeholder={t('Search by order number...')}
          />
        </form>
        <TimeRangeFilterDialog
          currentFilters={filters}
          defaultTimeRangeDays={chartPreferences.defaultTimeRangeDays}
          defaultTimeGranularity={chartPreferences.defaultTimeGranularity}
          onFilterChange={(nextFilters) => {
            setPage(1)
            setFilters(nextFilters)
          }}
          onReset={() => {
            const { start, end } = getRollingDateRange(
              chartPreferences.defaultTimeRangeDays
            )
            setPage(1)
            setFilters({
              start_timestamp: start,
              end_timestamp: end,
              time_granularity: chartPreferences.defaultTimeGranularity,
            })
          }}
          titleKey='Platform orders'
          descriptionKey='Filter platform orders by time range.'
          showGranularity={false}
        />
      </div>

      <Card className='gap-0 py-0'>
        <CardContent className='px-0'>{tableContent}</CardContent>
      </Card>

      <div className='flex items-center justify-between gap-3'>
        <p className='text-muted-foreground text-xs'>
          {t('{{count}} orders', { count: total })}
        </p>
        <div className='flex items-center gap-2'>
          <Button
            size='icon-sm'
            variant='outline'
            disabled={page <= 1}
            onClick={() => setPage((value) => value - 1)}
            aria-label={t('Previous page')}
          >
            <ChevronLeft />
          </Button>
          <span className='text-muted-foreground min-w-16 text-center text-xs'>
            {page} / {totalPages}
          </span>
          <Button
            size='icon-sm'
            variant='outline'
            disabled={page >= totalPages}
            onClick={() => setPage((value) => value + 1)}
            aria-label={t('Next page')}
          >
            <ChevronRight />
          </Button>
        </div>
      </div>

      <AlertDialog
        open={!!confirmTradeNo}
        onOpenChange={(open) => !open && setConfirmTradeNo(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Complete Order')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'Are you sure you want to manually complete this order? The user will be credited with the corresponding quota.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={completing}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmComplete}
              disabled={completing}
            >
              {completing ? t('Processing...') : t('Confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
