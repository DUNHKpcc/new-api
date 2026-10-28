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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import { listMyGuardianOrders } from './api'
import { GuardianShell } from './components/guardian-shell'
import { OrderCard } from './components/order-card'
import { GUARDIAN_QUERY_KEYS } from './constants'

const PAGE_SIZE = 10
const PENDING_REFRESH_MS = 15000

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
  const orders = useQuery({
    queryKey: [...GUARDIAN_QUERY_KEYS.myOrders, page],
    queryFn: () => listMyGuardianOrders(page, PAGE_SIZE),
    refetchInterval: (query) =>
      query.state.data?.items.some(
        (order) => order.status === 'pending_payment'
      )
        ? PENDING_REFRESH_MS
        : false,
  })
  const total = orders.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <GuardianShell>
      <div className='mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:py-20'>
        <p className='g-eyebrow'>{t('Hands-on services')}</p>
        <h1 className='g-display mt-4 text-[2.5rem] sm:text-5xl'>
          {t('My orders')}
        </h1>
        <p className='g-lede mt-4'>
          {t(
            'Track payment and delivery for the services you booked. Messages from support appear on each order.'
          )}
        </p>

        {props.payResult && <PayResultBanner result={props.payResult} />}

        <div className='mt-10 space-y-5' aria-busy={orders.isFetching}>
          {orders.isPending &&
            [0, 1].map((key) => (
              <Skeleton key={key} className='h-64 rounded-2xl' />
            ))}

          {orders.isError && (
            <ErrorState
              title={t('Failed to load your orders')}
              onRetry={() => void orders.refetch()}
            />
          )}

          {orders.isSuccess && orders.data.items.length === 0 && (
            <EmptyState
              title={t('No orders yet')}
              description={t('Book a setup service and it will show up here.')}
              action={
                <Button render={<Link to='/guardian/services' />}>
                  {t('Browse services')}
                </Button>
              }
              bordered
              className='rounded-2xl py-14'
            />
          )}

          {orders.isSuccess &&
            orders.data.items.map((order) => (
              <OrderCard key={order.id} order={order} />
            ))}
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
