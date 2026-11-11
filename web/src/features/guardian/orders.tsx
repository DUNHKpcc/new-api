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
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DatePicker } from '@/components/date-picker'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
  type CarouselApi,
} from '@/components/ui/carousel'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import { listMyGuardianOrders, type GuardianOrderFilter } from './api'
import { GuardianCustomerServices } from './components/customer-services'
import { GuardianShell } from './components/guardian-shell'
import { OrderCard } from './components/order-card'
import { GUARDIAN_QUERY_KEYS } from './constants'
import type { GuardianOrderStatus } from './types'

const PAGE_SIZE = 10
const PENDING_REFRESH_MS = 15000
/** Orders whose payment or queue place can change while the page is open. */
const WATCHED_STATUSES = new Set<GuardianOrderStatus>([
  'pending_payment',
  'paid',
  'scheduled',
  'in_progress',
])

export type GuardianPayResult = 'success' | 'pending' | 'fail'

function PayResultBanner(props: { result: GuardianPayResult }) {
  const { t } = useTranslation()
  const messages: Record<GuardianPayResult, string> = {
    success: t(
      'Payment received. We will reach out through the contact you left to schedule.'
    ),
    pending: t(
      'Payment is still being confirmed. This page refreshes on its own.'
    ),
    fail: t(
      'We could not verify this payment. If you were charged, contact support with your order number.'
    ),
  }
  return (
    <Alert
      variant={props.result === 'fail' ? 'destructive' : 'default'}
      className='mt-8'
    >
      <AlertDescription>{messages[props.result]}</AlertDescription>
    </Alert>
  )
}

export function GuardianOrders(props: { payResult?: GuardianPayResult }) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [page, setPage] = useState(1)
  const [orderNumber, setOrderNumber] = useState('')
  const [startDate, setStartDate] = useState<Date>()
  const [endDate, setEndDate] = useState<Date>()
  const [filter, setFilter] = useState<GuardianOrderFilter>({})
  const invalidRange = !!(startDate && endDate && startDate > endDate)
  const filtered = !!(
    filter.order_number ||
    filter.created_after ||
    filter.created_before
  )
  const [carousel, setCarousel] = useState<CarouselApi>()
  const [selected, setSelected] = useState(0)
  useEffect(() => {
    if (!carousel) return
    const syncSelection = () => setSelected(carousel.selectedScrollSnap())
    syncSelection()
    carousel.on('select', syncSelection)
    carousel.on('reInit', syncSelection)
    return () => {
      carousel.off('select', syncSelection)
      carousel.off('reInit', syncSelection)
    }
  }, [carousel])
  const orders = useQuery({
    queryKey: [...GUARDIAN_QUERY_KEYS.myOrders, page, filter],
    queryFn: () => listMyGuardianOrders(page, PAGE_SIZE, filter),
    refetchInterval: (query) =>
      query.state.data?.items.some((order) =>
        WATCHED_STATUSES.has(order.status)
      )
        ? PENDING_REFRESH_MS
        : false,
  })
  const total = orders.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <GuardianShell>
      <div className='mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:py-10'>
        <p className='g-eyebrow'>{t('Hands-on services')}</p>
        <h1 className='g-display mt-4 text-[2.5rem] sm:text-5xl'>
          {t('My orders')}
        </h1>
        <p className='g-lede mt-4'>
          {t(
            'Track payment and delivery for the services you booked. Messages from support appear on each order.'
          )}
        </p>

        <GuardianCustomerServices />
        {props.payResult && <PayResultBanner result={props.payResult} />}

        <form
          className='mt-8 grid gap-3 sm:grid-cols-2'
          onSubmit={(event) => {
            event.preventDefault()
            if (invalidRange) return
            setPage(1)
            setSelected(0)
            setFilter({
              order_number: orderNumber.trim() || undefined,
              created_after: startDate
                ? Math.floor(startDate.getTime() / 1000)
                : undefined,
              created_before: endDate
                ? Math.floor(
                    new Date(
                      endDate.getFullYear(),
                      endDate.getMonth(),
                      endDate.getDate() + 1
                    ).getTime() / 1000
                  )
                : undefined,
            })
          }}
        >
          <Input
            className='sm:col-span-2'
            aria-label={t('Order number')}
            placeholder={t('Order number')}
            maxLength={64}
            value={orderNumber}
            onChange={(e) => setOrderNumber(e.target.value)}
          />
          <DatePicker
            ariaLabel={t('Order start date')}
            placeholder={t('Order start date')}
            selected={startDate}
            onSelect={setStartDate}
            className='w-full'
          />
          <DatePicker
            ariaLabel={t('Order end date')}
            placeholder={t('Order end date')}
            selected={endDate}
            onSelect={setEndDate}
            className='w-full'
          />
          {invalidRange && (
            <p role='alert' className='text-destructive text-sm sm:col-span-2'>
              {t('Start date must not be after end date.')}
            </p>
          )}
          <div className='flex gap-2 sm:col-span-2'>
            <Button type='submit' disabled={invalidRange}>
              {t('Search')}
            </Button>
            <Button
              type='button'
              variant='outline'
              onClick={() => {
                setOrderNumber('')
                setStartDate(undefined)
                setEndDate(undefined)
                setFilter({})
                setPage(1)
                setSelected(0)
              }}
            >
              {t('Reset')}
            </Button>
          </div>
        </form>
        <div className='mt-6 space-y-5' aria-busy={orders.isFetching}>
          {orders.isPending && <Skeleton className='h-64 rounded-2xl' />}

          {orders.isError && (
            <ErrorState
              title={t('Failed to load your orders')}
              onRetry={() => void orders.refetch()}
            />
          )}

          {orders.isSuccess && orders.data.items.length === 0 && (
            <EmptyState
              title={filtered ? t('No results found') : t('No orders yet')}
              description={
                filtered
                  ? undefined
                  : t('Book a setup service and it will show up here.')
              }
              action={
                <Button render={<Link to='/guardian/services' />}>
                  {t('Browse services')}
                </Button>
              }
              bordered
              className='rounded-2xl py-14'
            />
          )}

          {orders.isSuccess && orders.data.items.length > 0 && (
            <Carousel
              key={`${page}-${filter.order_number ?? ''}-${filter.created_after ?? ''}-${filter.created_before ?? ''}`}
              setApi={setCarousel}
              opts={{ align: 'start' }}
              aria-label={t('My orders')}
            >
              <div className='mb-4 flex items-center justify-between gap-4'>
                <span
                  className='text-sm text-[var(--g-ink-soft)]'
                  aria-live='polite'
                >
                  {t('Slide {{current}} of {{total}}', {
                    current: formatNumber(
                      (page - 1) * PAGE_SIZE + selected + 1,
                      locale
                    ),
                    total: formatNumber(total, locale),
                  })}
                </span>
                <div className='flex gap-2'>
                  <CarouselPrevious className='static translate-y-0' />
                  <CarouselNext className='static translate-y-0' />
                </div>
              </div>
              <CarouselContent className='items-start'>
                {orders.data.items.map((order, index) => (
                  <CarouselItem
                    key={order.id}
                    inert={index !== selected}
                    aria-hidden={index !== selected}
                  >
                    <OrderCard order={order} queue={orders.data.queue} />
                  </CarouselItem>
                ))}
              </CarouselContent>
            </Carousel>
          )}
        </div>

        {orders.isSuccess && pageCount > 1 && (
          <nav
            aria-label={t('Pagination')}
            className='mt-8 flex items-center justify-between text-sm'
          >
            <Button
              variant='outline'
              disabled={page <= 1}
              onClick={() => setPage((value) => value - 1)}
            >
              {t('Previous')}
            </Button>
            <span className='text-[var(--g-ink-soft)]'>
              {t('Page {{page}} of {{count}}', {
                page: formatNumber(page, locale),
                count: formatNumber(pageCount, locale),
              })}
            </span>
            <Button
              variant='outline'
              disabled={page >= pageCount}
              onClick={() => setPage((value) => value + 1)}
            >
              {t('Next')}
            </Button>
          </nav>
        )}
      </div>
    </GuardianShell>
  )
}
