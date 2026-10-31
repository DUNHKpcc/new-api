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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowDown, ArrowUp } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { getAdminGuardianQueue, moveGuardianQueueOrder } from '../../api'
import { GUARDIAN_QUERY_KEYS } from '../../constants'
import type { AdminGuardianOrder, AdminGuardianQueue } from '../../types'
import { OrderStatusBadge } from '../order-status-badge'
import { GuardianQueueSettingsForm } from './queue-settings-form'

const QUEUE_REFRESH_MS = 30000

function OrderSummary(props: { order: AdminGuardianOrder }) {
  return (
    <>
      <TableCell className='font-mono text-xs'>
        {props.order.trade_no}
      </TableCell>
      <TableCell>{props.order.username}</TableCell>
      <TableCell className='max-w-56 break-words whitespace-normal'>
        {props.order.service_title}
      </TableCell>
      <TableCell className='text-xs'>
        {formatTimestampToDate(props.order.paid_at)}
      </TableCell>
    </>
  )
}

function WaitingRow(props: {
  queue: AdminGuardianQueue
  index: number
  moving: boolean
  onMove: (position: number) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const order = props.queue.waiting[props.index]
  const position = props.index + 1
  const [target, setTarget] = useState('')

  return (
    <TableRow>
      <TableCell className='font-medium'>
        {formatNumber(position, locale)}
      </TableCell>
      <OrderSummary order={order} />
      <TableCell className='text-xs'>
        <div>
          {t('{{count}} ahead', {
            count: formatNumber(order.queue_ahead, locale),
          })}
        </div>
        <div className='text-muted-foreground'>
          {order.queue_start_date || t('No estimate')}
        </div>
      </TableCell>
      <TableCell>
        <form
          className='flex items-center gap-1'
          onSubmit={(event) => {
            event.preventDefault()
            const value = Number(target)
            if (!Number.isInteger(value) || value < 1) return
            props.onMove(value)
            setTarget('')
          }}
        >
          <Button
            type='button'
            variant='ghost'
            size='icon'
            aria-label={t('Move up')}
            disabled={props.moving || position === 1}
            onClick={() => props.onMove(position - 1)}
          >
            <ArrowUp aria-hidden='true' />
          </Button>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            aria-label={t('Move down')}
            disabled={props.moving || position === props.queue.waiting.length}
            onClick={() => props.onMove(position + 1)}
          >
            <ArrowDown aria-hidden='true' />
          </Button>
          <Input
            type='number'
            min={1}
            max={props.queue.waiting.length}
            aria-label={t('Move to position')}
            placeholder='#'
            className='w-16'
            value={target}
            onChange={(event) => setTarget(event.target.value)}
          />
          <Button
            type='submit'
            variant='outline'
            size='sm'
            disabled={props.moving || target === ''}
          >
            {t('Move')}
          </Button>
        </form>
      </TableCell>
    </TableRow>
  )
}

export function GuardianQueuePanel() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const queryClient = useQueryClient()
  const queue = useQuery({
    queryKey: GUARDIAN_QUERY_KEYS.adminQueue,
    queryFn: getAdminGuardianQueue,
    refetchInterval: QUEUE_REFRESH_MS,
  })
  const move = useMutation({
    mutationFn: (args: { id: number; position: number }) =>
      moveGuardianQueueOrder(args.id, args.position),
    onSettled: () =>
      queryClient.invalidateQueries({
        queryKey: GUARDIAN_QUERY_KEYS.adminQueue,
      }),
    onError: (error) => handleServerError(error, t('Failed to move the order')),
  })

  if (queue.isPending) return <LoadingState />
  if (queue.isError) {
    return (
      <ErrorState
        title={t('Failed to load the queue')}
        onRetry={() => void queue.refetch()}
      />
    )
  }
  const data = queue.data
  const capacity = data.setting.daily_capacity

  return (
    <div className='space-y-4'>
      <GuardianQueueSettingsForm setting={data.setting} />
      <TitledCard
        title={t('Delivery queue')}
        description={t(
          'Orders start from the top. Buyers see how many orders are ahead of them, including extra slots.'
        )}
        disableHoverEffect
      >
        <p className='text-muted-foreground mb-4 flex flex-wrap gap-x-6 gap-y-1 text-sm'>
          <span>
            {t('In progress: {{count}}', {
              count: formatNumber(data.in_progress.length, locale),
            })}
          </span>
          <span>
            {capacity > 0
              ? t('Started today: {{count}} of {{limit}}', {
                  count: formatNumber(data.started_today, locale),
                  limit: formatNumber(capacity, locale),
                })
              : t('Started today: {{count}}', {
                  count: formatNumber(data.started_today, locale),
                })}
          </span>
          {data.paused && (
            <span className='text-destructive'>
              {t('Scheduling is paused')}
            </span>
          )}
        </p>
        {data.in_progress.length === 0 && data.waiting.length === 0 ? (
          <EmptyState title={t('No orders are waiting')} className='min-h-40' />
        ) : (
          <StaticDataTable tableClassName='min-w-[900px]'>
            <TableHeader>
              <TableRow>
                <TableHead>#</TableHead>
                <TableHead>{t('Order number')}</TableHead>
                <TableHead>{t('User')}</TableHead>
                <TableHead>{t('Service')}</TableHead>
                <TableHead>{t('Paid at')}</TableHead>
                <TableHead>{t('Buyer sees')}</TableHead>
                <TableHead>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.in_progress.map((order) => (
                <TableRow key={order.id}>
                  <TableCell>—</TableCell>
                  <OrderSummary order={order} />
                  <TableCell>
                    <OrderStatusBadge status={order.status} />
                  </TableCell>
                  <TableCell />
                </TableRow>
              ))}
              {data.waiting.map((order, index) => (
                <WaitingRow
                  key={order.id}
                  queue={data}
                  index={index}
                  moving={move.isPending}
                  onMove={(position) => move.mutate({ id: order.id, position })}
                />
              ))}
            </TableBody>
          </StaticDataTable>
        )}
      </TitledCard>
    </div>
  )
}
