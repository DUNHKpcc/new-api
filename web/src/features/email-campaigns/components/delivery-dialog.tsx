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

import {
  DataTablePagination,
  StaticDataTable,
  useDataTable,
} from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampToDate } from '@/lib/format'

import { listEmailDeliveries } from '../api'
import type { DeliveryStatus, EmailCampaign, EmailDelivery } from '../types'
import { EmailStatus } from './email-status'

const EMPTY_DELIVERIES: EmailDelivery[] = []
export function EmailDeliveryDialog(props: {
  campaign: EmailCampaign
  onClose: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 20 })
  const [status, setStatus] = useState('')
  const records = useQuery({
    queryKey: [
      'email-campaigns',
      props.campaign.id,
      'deliveries',
      pagination,
      status,
    ],
    queryFn: () =>
      listEmailDeliveries(
        props.campaign.id,
        pagination.pageIndex + 1,
        pagination.pageSize,
        status
      ),
    refetchInterval: 10000,
  })
  const { table } = useDataTable({
    data: records.data?.items ?? EMPTY_DELIVERIES,
    columns: [],
    totalCount: records.data?.total ?? 0,
    manualPagination: true,
    columnFilters: [],
    pagination,
    onPaginationChange: setPagination,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  const filters: Array<{ value: DeliveryStatus | ''; label: string }> = [
    { value: '', label: t('All Status') },
    { value: 'pending', label: t('Pending delivery') },
    { value: 'sending', label: t('Sending') },
    { value: 'retry', label: t('Retry scheduled') },
    { value: 'sent', label: t('Accepted by SMTP') },
    { value: 'failed', label: t('Failed') },
    { value: 'skipped', label: t('Skipped') },
    { value: 'uncertain', label: t('Delivery uncertain') },
    { value: 'cancelled', label: t('Cancelled') },
  ]
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
      title={t('Email delivery records')}
      description={props.campaign.subject}
      contentClassName='sm:max-w-5xl'
    >
      <div className='space-y-3'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Accepted by SMTP does not guarantee inbox delivery. Uncertain deliveries are not retried automatically.'
          )}
        </p>
        <dl className='grid grid-cols-2 gap-2 text-xs sm:grid-cols-4'>
          {filters.map(
            (filter) =>
              filter.value && (
                <div key={filter.value} className='rounded-md border p-2'>
                  <dt className='text-muted-foreground'>{filter.label}</dt>
                  <dd className='mt-1 font-medium tabular-nums'>
                    {formatNumber(
                      props.campaign.delivery_counts[filter.value],
                      locale
                    )}
                  </dd>
                </div>
              )
          )}
        </dl>
        <NativeSelect
          aria-label={t('Delivery status')}
          value={status}
          onChange={(event) => {
            setStatus(event.target.value)
            setPagination((previous) => ({ ...previous, pageIndex: 0 }))
          }}
        >
          {filters.map((filter) => (
            <NativeSelectOption key={filter.value} value={filter.value}>
              {filter.label}
            </NativeSelectOption>
          ))}
        </NativeSelect>
        <div aria-busy={records.isFetching}>
          {records.isPending && <LoadingState />}
          {records.isError && (
            <ErrorState
              title={t('Failed to load delivery records')}
              onRetry={() => void records.refetch()}
            />
          )}
          {records.isSuccess && (
            <>
              {records.data.items.length ? (
                <StaticDataTable tableClassName='min-w-[850px]'>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Email')}</TableHead>
                      <TableHead>{t('Status')}</TableHead>
                      <TableHead>{t('Attempts')}</TableHead>
                      <TableHead>{t('Updated')}</TableHead>
                      <TableHead>{t('Next attempt')}</TableHead>
                      <TableHead>{t('Accepted at')}</TableHead>
                      <TableHead>{t('Error')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {records.data.items.map((delivery) => (
                      <TableRow key={delivery.id}>
                        <TableCell className='max-w-60 break-all whitespace-normal'>
                          {delivery.email}
                        </TableCell>
                        <TableCell>
                          <EmailStatus status={delivery.status} />
                        </TableCell>
                        <TableCell>
                          {formatNumber(delivery.attempts, locale)}
                        </TableCell>
                        <TableCell className='text-xs'>
                          {formatTimestampToDate(delivery.updated_at)}
                        </TableCell>
                        <TableCell className='text-xs'>
                          {formatTimestampToDate(delivery.next_attempt_at)}
                        </TableCell>
                        <TableCell className='text-xs'>
                          {formatTimestampToDate(delivery.sent_at)}
                        </TableCell>
                        <TableCell className='max-w-72 text-xs break-words whitespace-normal'>
                          {delivery.last_error || '—'}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </StaticDataTable>
              ) : (
                <EmptyState
                  title={t('No email delivery records')}
                  className='min-h-32'
                />
              )}
              <DataTablePagination table={table} compact />
            </>
          )}
        </div>
      </div>
    </Dialog>
  )
}
