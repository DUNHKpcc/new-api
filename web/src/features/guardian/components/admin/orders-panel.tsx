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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DataTablePagination,
  StaticDataTable,
  useDataTable,
} from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import { toIntlLocale } from '@/i18n/languages'
import { formatTimestampToDate } from '@/lib/format'

import { listAdminGuardianOrders } from '../../api'
import { GUARDIAN_QUERY_KEYS } from '../../constants'
import { useGuardianLabels } from '../../hooks/use-guardian-labels'
import { formatCny } from '../../lib/price'
import type { AdminGuardianOrder, GuardianOrderStatus } from '../../types'
import { DiscoveryOptions, DiscoveryLabel } from '../discovery-options'
import { OrderStatusBadge } from '../order-status-badge'
import { GuardianOrderDialog } from './order-dialog'

const EMPTY_ORDERS: AdminGuardianOrder[] = []

export function GuardianOrdersPanel() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const queryClient = useQueryClient()
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 20 })
  const [status, setStatus] = useState<GuardianOrderStatus | ''>('')
  const [discovery, setDiscovery] = useState('')
  const [anomaliesOnly, setAnomaliesOnly] = useState(false)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [managing, setManaging] = useState<AdminGuardianOrder | null>(null)
  const orders = useQuery({
    queryKey: [
      ...GUARDIAN_QUERY_KEYS.adminOrders,
      pagination,
      status,
      keyword,
      anomaliesOnly,
      discovery,
    ],
    queryFn: () =>
      listAdminGuardianOrders({
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        status,
        keyword,
        payment_anomaly: anomaliesOnly,
        discovery_channel: discovery,
      }),
  })
  const { table } = useDataTable({
    data: orders.data?.items ?? EMPTY_ORDERS,
    columns: [],
    totalCount: orders.data?.total ?? 0,
    manualPagination: true,
    columnFilters: [],
    pagination,
    onPaginationChange: setPagination,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  const resetPage = () => setPagination((value) => ({ ...value, pageIndex: 0 }))

  return (
    <TitledCard
      title={t('Service orders')}
      description={t(
        'Orders are marked paid by the payment callback. Move them through scheduling and delivery here.'
      )}
      disableHoverEffect
    >
      <form
        className='mb-4 flex flex-wrap gap-2'
        onSubmit={(event) => {
          event.preventDefault()
          setKeyword(keywordInput.trim())
          resetPage()
        }}
      >
        <NativeSelect
          aria-label={t('Status')}
          className='w-40'
          value={status}
          onChange={(event) => {
            setStatus(event.target.value as GuardianOrderStatus | '')
            resetPage()
          }}
        >
          <NativeSelectOption value=''>{t('All statuses')}</NativeSelectOption>
          {Object.entries(labels.status).map(([value, label]) => (
            <NativeSelectOption key={value} value={value}>
              {label}
            </NativeSelectOption>
          ))}
        </NativeSelect>
        <NativeSelect
          aria-label={t('How did you hear about us?')}
          value={discovery}
          onChange={(e) => {
            setDiscovery(e.target.value)
            resetPage()
          }}
        >
          <NativeSelectOption value=''>{t('All sources')}</NativeSelectOption>
          <NativeSelectOption value='unrecorded'>
            {t('Not recorded')}
          </NativeSelectOption>
          <DiscoveryOptions />
        </NativeSelect>
        <Button
          type='button'
          variant={anomaliesOnly ? 'default' : 'outline'}
          aria-pressed={anomaliesOnly}
          onClick={() => {
            setAnomaliesOnly(!anomaliesOnly)
            resetPage()
          }}
        >
          {t('Payment anomalies')}
        </Button>
        <Input
          aria-label={t('Search orders')}
          placeholder={t('Order number, username or contact')}
          className='w-64'
          value={keywordInput}
          onChange={(event) => setKeywordInput(event.target.value)}
        />
        <Button type='submit' variant='outline'>
          {t('Search')}
        </Button>
      </form>

      <div className='space-y-3' aria-busy={orders.isFetching}>
        {orders.isPending && <LoadingState />}
        {orders.isError && (
          <ErrorState
            title={t('Failed to load orders')}
            onRetry={() => void orders.refetch()}
          />
        )}
        {orders.isSuccess && orders.data.items.length === 0 && (
          <EmptyState title={t('No orders found')} className='min-h-40' />
        )}
        {orders.isSuccess && orders.data.items.length > 0 && (
          <StaticDataTable tableClassName='min-w-[860px]'>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Order number')}</TableHead>
                <TableHead>{t('Order source')}</TableHead>
                <TableHead>{t('Discovery channel')}</TableHead>
                <TableHead>{t('User')}</TableHead>
                <TableHead>{t('Service')}</TableHead>
                <TableHead>{t('Amount')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Ordered at')}</TableHead>
                <TableHead>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.data.items.map((order) => (
                <TableRow key={order.id}>
                  <TableCell className='font-mono text-xs'>
                    {order.trade_no}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={order.reseller_code ? 'secondary' : 'outline'}
                    >
                      {order.reseller_code
                        ? t('Reseller')
                        : t('Platform direct')}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <DiscoveryLabel value={order.discovery_channel} />
                  </TableCell>
                  <TableCell>
                    <div>{order.username}</div>
                    <div className='text-muted-foreground text-xs'>
                      {labels.contactType[order.contact_type] ??
                        order.contact_type}{' '}
                      · {order.contact}
                    </div>
                  </TableCell>
                  <TableCell className='max-w-56 break-words whitespace-normal'>
                    {order.service_title}
                    {order.ip_option_name && (
                      <span className='block text-xs text-[var(--g-accent)]'>
                        {t('IP add-ons')} · {order.ip_option_name} ·{' '}
                        {order.ip_billing_period && (
                          <>
                            {labels.billingPeriod[order.ip_billing_period]}{' '}
                            ·{' '}
                          </>
                        )}
                        {formatCny(order.ip_option_price_cents ?? 0, locale)}
                        <span className='block whitespace-pre-wrap'>
                          {order.ip_option_description}
                        </span>
                      </span>
                    )}
                    {(order.claude_max_price_cents ?? 0) > 0 && (
                      <span className='block text-xs text-[var(--g-accent)]'>
                        {t('Claude MAX subscription')} ·{' '}
                        {formatCny(order.claude_max_price_cents ?? 0, locale)}
                      </span>
                    )}
                    {(order.claude_pro_price_cents ?? 0) > 0 && (
                      <span className='block text-xs text-[var(--g-accent)]'>
                        {t('Claude Pro subscription')} ·{' '}
                        {formatCny(order.claude_pro_price_cents ?? 0, locale)}
                      </span>
                    )}
                    {(order.claude_ultra_price_cents ?? 0) > 0 && (
                      <span className='block text-xs text-[var(--g-accent)]'>
                        {t('Claude Ultra subscription')} ·{' '}
                        {formatCny(order.claude_ultra_price_cents ?? 0, locale)}
                      </span>
                    )}
                    {order.billing_period && (
                      <span className='text-muted-foreground block text-xs'>
                        {labels.billingPeriod[order.billing_period]}
                      </span>
                    )}
                  </TableCell>
                  <TableCell>{formatCny(order.amount_cents, locale)}</TableCell>
                  <TableCell>
                    <OrderStatusBadge status={order.status} />
                    {order.payment_anomaly && (
                      <p className='text-destructive mt-1 text-xs'>
                        {t('Payment anomalies')}
                      </p>
                    )}
                  </TableCell>
                  <TableCell className='text-xs'>
                    {formatTimestampToDate(order.created_at)}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setManaging(order)}
                    >
                      {t('Manage')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </StaticDataTable>
        )}
        {orders.isSuccess && <DataTablePagination table={table} compact />}
      </div>

      {managing && (
        <GuardianOrderDialog
          order={managing}
          onClose={() => setManaging(null)}
          onSaved={() =>
            void queryClient.invalidateQueries({
              queryKey: GUARDIAN_QUERY_KEYS.adminOrders,
            })
          }
        />
      )}
    </TitledCard>
  )
}
