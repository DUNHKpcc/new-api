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
import { MessageSquareText } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { toIntlLocale } from '@/i18n/languages'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { GUARDIAN_ORDER_TRACK } from '../constants'
import { useGuardianLabels } from '../hooks/use-guardian-labels'
import { formatCny } from '../lib/price'
import type { GuardianOrder, GuardianQueueState } from '../types'
import { EditGuardianContact } from './contact-dialog'
import { OrderActions } from './order-actions'
import { OrderStatusBadge } from './order-status-badge'
import { QueueTracker } from './queue-tracker'

export function OrderProgress(props: { order: GuardianOrder }) {
  const { t } = useTranslation()
  const labels = useGuardianLabels()
  const current = GUARDIAN_ORDER_TRACK.indexOf(props.order.status)
  if (current === -1) return null

  return (
    <ol aria-label={t('Order progress')} className='grid grid-cols-5 gap-2'>
      {GUARDIAN_ORDER_TRACK.map((step, index) => (
        <li
          key={step}
          aria-current={index === current ? 'step' : undefined}
          className='min-w-0'
        >
          <span
            aria-hidden='true'
            className={cn(
              'block h-1 rounded-full',
              index <= current ? 'bg-[var(--g-accent)]' : 'bg-[var(--g-rule)]'
            )}
          />
          <span
            className={cn(
              'mt-2 block truncate text-xs',
              index === current
                ? 'font-medium text-[var(--g-ink)]'
                : 'text-[var(--g-ink-soft)]'
            )}
          >
            {labels.status[step]}
          </span>
        </li>
      ))}
    </ol>
  )
}

export function OrderCard(props: {
  order: GuardianOrder
  queue?: GuardianQueueState
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const order = props.order
  const details = [
    { label: t('Route'), value: labels.orderRoute[order.route] ?? order.route },
    { label: t('Operating system'), value: labels.os[order.os] ?? order.os },
    {
      label: t('Contact'),
      value: `${labels.contactType[order.contact_type] ?? order.contact_type} · ${order.contact}`,
    },
    { label: t('Ordered at'), value: formatTimestampToDate(order.created_at) },
  ]

  return (
    <article className='g-rule rounded-2xl border bg-[var(--g-paper-raised)] p-5 sm:p-6'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div className='min-w-0'>
          <h2 className='g-heading text-xl break-words'>
            {order.service_title}
          </h2>
          <p className='g-mono mt-1 flex items-center gap-1 text-xs text-[var(--g-ink-soft)]'>
            <span className='truncate'>{order.trade_no}</span>
            <CopyButton
              value={order.trade_no}
              variant='ghost'
              size='icon'
              className='size-6'
              tooltip={t('Copy order number')}
            />
          </p>
        </div>
        <div className='flex items-center gap-3'>
          <span className='g-heading text-xl'>
            {formatCny(order.amount_cents, locale)}
          </span>
          <OrderStatusBadge status={order.status} />
        </div>
      </div>

      <div className='mt-6'>
        <OrderProgress order={order} />
      </div>

      <QueueTracker order={order} queue={props.queue} />

      {order.status === 'pending_payment' && (
        <>
          <p className='mt-4 text-sm text-[var(--g-ink-soft)]'>
            {t(
              'If you have already paid, the status updates within a few minutes. Unpaid orders expire after two hours.'
            )}
          </p>
          <OrderActions order={order} />
        </>
      )}

      <dl className='g-rule mt-6 grid gap-x-8 gap-y-3 border-t pt-5 text-sm sm:grid-cols-2'>
        {details.map((detail) => (
          <div key={detail.label} className='flex min-w-0 gap-3'>
            <dt className='w-24 shrink-0 text-[var(--g-ink-soft)]'>
              {detail.label}
            </dt>
            <dd className='min-w-0 break-words'>{detail.value}</dd>
          </div>
        ))}
        {order.note && (
          <div className='flex min-w-0 gap-3 sm:col-span-2'>
            <dt className='w-24 shrink-0 text-[var(--g-ink-soft)]'>
              {t('Your note')}
            </dt>
            <dd className='min-w-0 break-words whitespace-pre-line'>
              {order.note}
            </dd>
          </div>
        )}
      </dl>

      {['pending_payment', 'paid', 'scheduled', 'in_progress'].includes(
        order.status
      ) && <EditGuardianContact order={order} />}

      {order.admin_note && (
        <div className='mt-5 rounded-xl bg-[var(--g-accent-wash)] p-4'>
          <p className='flex items-center gap-2 text-sm font-medium text-[var(--g-accent)]'>
            <MessageSquareText className='size-4' aria-hidden='true' />
            {t('Message from support')}
          </p>
          <p className='mt-2 text-sm leading-6 break-words whitespace-pre-line'>
            {order.admin_note}
          </p>
        </div>
      )}
    </article>
  )
}
