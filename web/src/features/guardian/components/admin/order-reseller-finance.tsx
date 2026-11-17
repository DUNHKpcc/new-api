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

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'

import { formatCny } from '../../lib/price'
import {
  resellerRead,
  resellerWrite,
  type AdminResellerOrder,
} from '../../reseller-api'
import {
  ResellerActionDialog,
  type ResellerAction,
} from '../reseller/action-dialog'

export function GuardianOrderResellerFinance({
  orderId,
  onRefundCompleted,
}: {
  orderId: number
  onRefundCompleted: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const money = (value: number) => formatCny(value, locale)
  const [action, setAction] = useState<ResellerAction | null>(null)
  const query = useQuery({
    queryKey: ['guardian', 'order-reseller-finance', orderId],
    queryFn: () =>
      resellerRead<AdminResellerOrder>(`/admin/orders/${orderId}/reseller`),
  })
  if (query.isPending) return <LoadingState />
  if (query.isError) return <ErrorState onRetry={() => void query.refetch()} />
  const o = query.data
  const path = '/admin/reseller-orders'
  const fee = o.fee_confirmed_at ? o.actual_fee_cents : o.estimated_fee_cents
  const details = [
    { label: t('Reseller user ID'), value: String(o.reseller_id) },
    { label: t('Supply price'), value: money(o.supply_cents) },
    {
      label: t('Payment fee'),
      value: `${money(fee)} · ${o.fee_confirmed_at ? t('Reconciled') : t('Estimated')}`,
    },
    {
      label: t('Order earnings'),
      value: !o.paid_at
        ? '—'
        : money(
            o.status === 'refunded'
              ? 0
              : o.amount_cents -
                  o.supply_cents -
                  (o.settlement_fee_cents ?? fee)
          ),
    },
    ...(o.refund_state === 'completed'
      ? [{ label: t('Refund amount'), value: money(o.refund_cents) }]
      : []),
    ...(o.refund_reason
      ? [{ label: t('Refund reason'), value: o.refund_reason }]
      : []),
    ...(o.fee_reference
      ? [{ label: t('Channel statement reference'), value: o.fee_reference }]
      : []),
    ...(o.refund_reference
      ? [
          {
            label: t('Refund transaction reference'),
            value: o.refund_reference,
          },
        ]
      : []),
  ]
  return (
    <section className='space-y-3 rounded-lg border p-3'>
      <h3 className='text-sm font-medium'>{t('Reseller settlement')}</h3>
      <dl className='grid gap-2 text-sm sm:grid-cols-2'>
        {details.map(({ label, value }) => (
          <div key={label}>
            <dt className='text-muted-foreground'>{label}</dt>
            <dd className='wrap-break-word'>{value}</dd>
          </div>
        ))}
      </dl>
      {o.refund_state === 'processing' && (
        <p className='text-destructive text-sm'>{t('Refund processing')}</p>
      )}
      <div className='flex gap-2'>
        {!!o.paid_at && o.status !== 'refunded' && !o.fee_confirmed_at && (
          <Button
            variant='outline'
            onClick={() =>
              setAction({
                title: 'Reconcile actual payment fee',
                description: o.trade_no,
                fields: [
                  {
                    name: 'fee_cents',
                    label: 'Actual payment fee (CNY)',
                    kind: 'money',
                    value: String(o.estimated_fee_cents / 100),
                  },
                  {
                    name: 'reference',
                    label: 'Channel statement reference',
                  },
                ],
                submit: (v) =>
                  resellerWrite(`${path}/${o.order_id}/fee`, {
                    ...v,
                    fee_cents: Math.round(Number(v.fee_cents) * 100),
                  }),
              })
            }
          >
            {t('Reconcile')}
          </Button>
        )}
        {!!o.paid_at && o.status !== 'refunded' && (
          <Button
            variant='outline'
            onClick={() =>
              setAction({
                title: 'Mark as refunded',
                description:
                  'The reason is visible to the reseller. This removes their earnings for the order and does not issue a payment refund.',
                fields: [{ name: 'reason', label: 'Refund reason' }],
                submit: async ({ reason }) => {
                  await resellerWrite(
                    `${path}/${o.order_id}/refund`,
                    { action: 'mark', reason },
                    'post'
                  )
                  onRefundCompleted()
                },
              })
            }
          >
            {t('Mark as refunded')}
          </Button>
        )}
      </div>
      {action && (
        <ResellerActionDialog action={action} onClose={() => setAction(null)} />
      )}
    </section>
  )
}
