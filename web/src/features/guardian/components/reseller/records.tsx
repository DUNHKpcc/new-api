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

import { StaticDataTable } from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'

import { useGuardianLabels } from '../../hooks/use-guardian-labels'
import { formatCny } from '../../lib/price'
import {
  resellerRead,
  resellerWrite,
  type ResellerPage,
  type ResellerOrder,
  type Withdrawal,
} from '../../reseller-api'
import { ResellerActionDialog, type ResellerAction } from './action-dialog'

export function ResellerRecords({
  admin = false,
  withdrawals = false,
}: {
  admin?: boolean
  withdrawals?: boolean
}) {
  const { t, i18n } = useTranslation()
  const labels = useGuardianLabels()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const money = (v: number) => formatCny(v, locale)
  const [page, setPage] = useState(1)
  const [action, setAction] = useState<ResellerAction | null>(null)
  const path = admin
    ? `/admin/reseller-${withdrawals ? 'withdrawals' : 'orders'}`
    : `/reseller/${withdrawals ? 'withdrawals' : 'orders'}`
  const query = useQuery({
    queryKey: ['guardian', 'reseller-records', path, page],
    queryFn: () =>
      resellerRead<ResellerPage<ResellerOrder | Withdrawal>>(
        `${path}?p=${page}`
      ),
  })
  if (query.isPending) return <LoadingState />
  if (query.isError) return <ErrorState onRetry={() => void query.refetch()} />
  const statusLabels: Record<string, string> = {
    pending: t('Pending review'),
    processing: t('Processing'),
    paid: t('Paid'),
    failed: t('Failed'),
    cancelled: t('Cancelled'),
    rejected: t('Rejected'),
  }
  return (
    <div className='space-y-4'>
      {withdrawals ? (
        <StaticDataTable
          data={query.data.items as Withdrawal[]}
          getRowKey={(w) => w.id}
          emptyContent={t('No withdrawals')}
          columns={[
            { id: 'id', header: t('ID'), cell: (w) => String(w.id) },
            ...(admin
              ? [
                  {
                    id: 'reseller',
                    header: t('Reseller user ID'),
                    cell: (w: Withdrawal) => String(w.reseller_id),
                  },
                ]
              : []),
            {
              id: 'payee',
              header: t('Recipient'),
              cell: (w) => (
                <span className='break-all'>
                  {w.payee_name}
                  <br />
                  {w.account}
                </span>
              ),
            },
            {
              id: 'amount',
              header: t('Amount / payout fee'),
              cell: (w) => `${money(w.amount_cents)} / ${money(w.fee_cents)}`,
            },
            {
              id: 'net',
              header: t('Net transfer'),
              cell: (w) =>
                w.status === 'pending'
                  ? t('Pending review')
                  : money(w.amount_cents - w.fee_cents),
            },
            {
              id: 'status',
              header: t('Status'),
              cell: (w) => statusLabels[w.status] ?? w.status,
            },
            {
              id: 'reference',
              header: t('Reference'),
              cell: (w) => w.reference ?? '—',
            },
            {
              id: 'actions',
              header: t('Actions'),
              cell: (w) => {
                let actions: string[] = []
                if (admin && w.status === 'pending') {
                  actions = ['processing', 'rejected']
                }
                if (admin && w.status === 'processing') {
                  actions = ['paid', 'failed']
                }
                if (!admin && w.status === 'pending') actions = ['cancelled']
                return (
                  <div className='flex gap-2'>
                    {actions.map((status) => (
                      <Button
                        variant='outline'
                        key={status}
                        onClick={() =>
                          setAction({
                            title:
                              (
                                {
                                  processing:
                                    'Verify recipient and reserve payout',
                                  paid: 'Confirm completed transfer',
                                } as Record<string, string>
                              )[status] ?? statusLabels[status],
                            description:
                              status === 'paid'
                                ? t(
                                    'Record only an actual successful transfer of {{amount}}. This button does not send money.',
                                    {
                                      amount: money(
                                        w.amount_cents - w.fee_cents
                                      ),
                                    }
                                  )
                                : t(
                                    'Unknown transfer results must stay Processing. Confirm failure before releasing the reserved balance.'
                                  ),
                            fields: admin
                              ? [
                                  ...(status === 'processing'
                                    ? [
                                        {
                                          name: 'fee_cents',
                                          label: 'Payout fee (CNY)',
                                          kind: 'money' as const,
                                          value: '0',
                                        },
                                      ]
                                    : []),
                                  ...(['paid', 'failed'].includes(status)
                                    ? [
                                        {
                                          name: 'reference',
                                          label: 'Transfer result reference',
                                        },
                                      ]
                                    : []),
                                  {
                                    name: 'note',
                                    label: 'Reason or verification record',
                                  },
                                ]
                              : [],
                            preview:
                              status === 'processing'
                                ? (v) =>
                                    t('Net Alipay transfer: {{amount}}', {
                                      amount: money(
                                        w.amount_cents -
                                          Math.round(
                                            Number(v.fee_cents || 0) * 100
                                          )
                                      ),
                                    })
                                : undefined,
                            submit: (v) =>
                              resellerWrite(
                                admin
                                  ? `${path}/${w.id}`
                                  : `${path}/${w.id}/cancel`,
                                {
                                  ...v,
                                  status,
                                  fee_cents:
                                    status === 'processing'
                                      ? Math.round(Number(v.fee_cents) * 100)
                                      : w.fee_cents,
                                },
                                admin ? 'put' : 'post'
                              ),
                          })
                        }
                      >
                        {(
                          {
                            processing: t('Review'),
                            paid: t('Mark paid'),
                          } as Record<string, string>
                        )[status] ?? statusLabels[status]}
                      </Button>
                    ))}
                  </div>
                )
              },
            },
          ]}
        />
      ) : (
        <StaticDataTable
          data={query.data.items as ResellerOrder[]}
          getRowKey={(o) => o.order_id}
          emptyContent={t('No orders yet')}
          columns={[
            {
              id: 'order',
              header: t('Order'),
              cell: (o) => (
                <span>
                  {o.trade_no}
                  <br />
                  {o.service_title}
                </span>
              ),
            },
            ...(admin
              ? [
                  {
                    id: 'reseller',
                    header: t('Reseller user ID'),
                    cell: (o: ResellerOrder) => String(o.reseller_id),
                  },
                ]
              : []),
            {
              id: 'amount',
              header: t('Payment / supply'),
              cell: (o) =>
                `${money(o.amount_cents)} / ${money(o.supply_cents)}`,
            },
            {
              id: 'fee',
              header: t('Payment fee'),
              cell: (o) => (
                <span>
                  {money(
                    o.fee_confirmed_at
                      ? o.actual_fee_cents
                      : o.estimated_fee_cents
                  )}{' '}
                  · {o.fee_confirmed_at ? t('Reconciled') : t('Estimated')}
                </span>
              ),
            },
            {
              id: 'net',
              header: t('Order earnings'),
              cell: (o) => {
                if (!o.paid_at) return '—'
                return money(
                  o.refund_state === 'completed'
                    ? 0
                    : o.amount_cents -
                        o.supply_cents -
                        (o.fee_confirmed_at
                          ? o.actual_fee_cents
                          : o.estimated_fee_cents)
                )
              },
            },
            {
              id: 'status',
              header: t('Status'),
              cell: (o) =>
                o.refund_state === 'processing'
                  ? t('Refund processing')
                  : labels.status[o.status],
            },
          ]}
        />
      )}
      <div className='flex justify-between'>
        <Button
          variant='outline'
          disabled={page === 1}
          onClick={() => setPage((p) => p - 1)}
        >
          {t('Previous')}
        </Button>
        <Button
          variant='outline'
          disabled={page * 20 >= query.data.total}
          onClick={() => setPage((p) => p + 1)}
        >
          {t('Next')}
        </Button>
      </div>
      {action && (
        <ResellerActionDialog action={action} onClose={() => setAction(null)} />
      )}
    </div>
  )
}
