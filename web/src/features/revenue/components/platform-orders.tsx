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
import { ChevronLeft, ChevronRight, Search } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
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
import { getAllBillingHistory } from '@/features/wallet/api'
import {
  formatTimestamp,
  getPaymentMethodName,
  getStatusConfig,
} from '@/features/wallet/lib/billing'
import { formatNumber } from '@/lib/format'

import { revenueMonthRange } from '../lib/analytics'
import { RevenueMonthSelector } from './revenue-month-selector'

export function PlatformOrders() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [month, setMonth] = useState(
    () => new Date(new Date().getFullYear(), new Date().getMonth(), 1)
  )
  const range = useMemo(() => revenueMonthRange(month), [month])
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
  const records = query.data?.data?.items ?? []
  const total = query.data?.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / 20))
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
            <TableHead>{t('Payment Method')}</TableHead>
            <TableHead>{t('Payment')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Completed at')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {records.map((record) => {
            const statusConfig = getStatusConfig(record.status)
            return (
              <TableRow key={record.id}>
                <TableCell>
                  <div className='max-w-56 truncate font-mono text-xs'>
                    {record.trade_no}
                  </div>
                </TableCell>
                <TableCell>{record.user_id}</TableCell>
                <TableCell>
                  {getPaymentMethodName(record.payment_method, t)}
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
        <RevenueMonthSelector
          month={month}
          onMonthChange={(value) => {
            setPage(1)
            setMonth(value)
          }}
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
    </div>
  )
}
