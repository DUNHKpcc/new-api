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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampToDate } from '@/lib/format'

import { getGuardianOrderHistory, getGuardianPaymentAnomalies } from '../../api'
import { useGuardianLabels } from '../../hooks/use-guardian-labels'
import { formatCny } from '../../lib/price'

export function GuardianOrderHistory(props: { orderId: number }) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const [kind, setKind] = useState<'events' | 'payments' | null>(null)
  const [page, setPage] = useState(1)
  const events = useQuery({
    queryKey: ['guardian', 'history', props.orderId, page],
    queryFn: () => getGuardianOrderHistory(props.orderId, page),
    enabled: kind === 'events',
  })
  const payments = useQuery({
    queryKey: ['guardian', 'payment-anomalies', props.orderId, page],
    queryFn: () => getGuardianPaymentAnomalies(props.orderId, page),
    enabled: kind === 'payments',
  })
  const active = kind === 'events' ? events : payments
  const actors: Record<string, string> = {
    user: t('User'),
    admin: t('Administrator'),
    system: t('System'),
    payment: t('Payment'),
  }
  const actions: Record<string, string> = {
    created: t('Order created'),
    expired: t('Expired'),
    payment_failed: t('Payment failed'),
    paid: t('Paid'),
    cancelled: t('Cancelled'),
    admin_updated: t('Order updated'),
    contact_updated: t('Edit contact and note'),
  }
  const fields = {
    status: t('Status'),
    admin_note: t('Message to the customer'),
    contact_type: t('Contact via'),
    contact: t('Contact'),
    note: t('Your note'),
  }
  return (
    <section className='space-y-3'>
      <div className='flex flex-wrap gap-2'>
        {(['events', 'payments'] as const).map((value) => (
          <Button
            key={value}
            variant='outline'
            size='sm'
            aria-expanded={kind === value}
            onClick={() => {
              setKind(kind === value ? null : value)
              setPage(1)
            }}
          >
            {value === 'events' ? t('Order history') : t('Payment anomalies')}
          </Button>
        ))}
      </div>
      {kind && (
        <>
          {active.isPending && <LoadingState />}
          {active.isError && (
            <ErrorState onRetry={() => void active.refetch()} />
          )}
          {active.isSuccess && active.data.items.length === 0 && (
            <EmptyState title={t('No records')} className='min-h-20' />
          )}
          {kind === 'events' &&
            events.data?.items.map((event) => (
              <article
                key={event.id}
                className='space-y-2 rounded-lg border p-3 text-sm'
              >
                <p className='text-muted-foreground'>
                  {formatTimestampToDate(event.created_at)} ·{' '}
                  {actors[event.actor_type] ?? event.actor_type}
                  {event.actor_id > 0 && ` #${event.actor_id}`} ·{' '}
                  {actions[event.action] ?? event.action}
                </p>
                {(Object.keys(fields) as (keyof typeof fields)[])
                  .filter((key) => event.before[key] !== event.after[key])
                  .map((key) => {
                    let before = event.before[key],
                      after = event.after[key]
                    if (key === 'status') {
                      before = labels.status[event.before.status] || before
                      after = labels.status[event.after.status] || after
                    }
                    if (key === 'contact_type') {
                      before =
                        labels.contactType[event.before.contact_type] || before
                      after =
                        labels.contactType[event.after.contact_type] || after
                    }
                    return (
                      <div
                        key={key}
                        className='break-words whitespace-pre-wrap'
                      >
                        <span className='font-medium'>{fields[key]}</span>
                        <div className='text-muted-foreground'>
                          {t('Before')}: {before || '—'}
                        </div>
                        <div>
                          {t('After')}: {after || '—'}
                        </div>
                      </div>
                    )
                  })}
              </article>
            ))}
          {kind === 'payments' &&
            payments.data?.items.map((item) => (
              <article
                key={item.id}
                className='space-y-1 rounded-lg border p-3 text-sm break-words'
              >
                <p>
                  {t('Expected amount')}:{' '}
                  {formatCny(item.expected_cents, locale)} ·{' '}
                  {t('Callback amount')}: {item.paid_money} CNY
                </p>
                <p>
                  {t('Payment method')}: {item.payment_method} ·{' '}
                  {t('Occurrences')}: {formatNumber(item.occurrences, locale)}
                </p>
                <p>
                  {t('First seen')}: {formatTimestampToDate(item.first_seen_at)}
                </p>
                <p>
                  {t('Last seen')}: {formatTimestampToDate(item.last_seen_at)}
                </p>
              </article>
            ))}
          {active.isSuccess && active.data.total > 10 && (
            <div className='flex items-center gap-3'>
              <Button
                size='sm'
                variant='outline'
                disabled={page === 1}
                onClick={() => setPage(page - 1)}
              >
                {t('Previous')}
              </Button>
              <span>
                {formatNumber(page, locale)} /{' '}
                {formatNumber(Math.ceil(active.data.total / 10), locale)}
              </span>
              <Button
                size='sm'
                variant='outline'
                disabled={page * 10 >= active.data.total}
                onClick={() => setPage(page + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          )}
        </>
      )}
    </section>
  )
}
