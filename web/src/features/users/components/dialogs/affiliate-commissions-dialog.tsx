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
import { ChevronLeft, ChevronRight, RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  DataTableRowActionMenu,
  StaticDataTable,
} from '@/components/data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Button } from '@/components/ui/button'
import {
  DropdownMenuItem,
  DropdownMenuShortcut,
} from '@/components/ui/dropdown-menu'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
import {
  formatAffiliateMinor,
  formatAffiliateQuota,
  formatAffiliateRate,
} from '@/features/affiliate/lib'
import { formatTimestamp } from '@/lib/format'

import {
  getAdminAffiliateCommissions,
  reverseAdminAffiliateCommission,
} from '../../api'
import {
  isAffiliateCommissionReversible,
  normalizeAffiliateCommissionReverseReason,
} from '../../lib/affiliate-commission'
import type {
  AdminAffiliateCommission,
  AffiliateCommissionStatus,
} from '../../types'

const PAGE_SIZE = 20

const STATUS_CONFIG: Record<
  AffiliateCommissionStatus,
  { label: string; variant: StatusVariant }
> = {
  pending: { label: 'Commission pending', variant: 'warning' },
  available: { label: 'Commission available', variant: 'success' },
  transferred: { label: 'Commission transferred', variant: 'info' },
  reversed: { label: 'Commission reversed', variant: 'danger' },
}

function CommissionStatus(props: { status: AffiliateCommissionStatus }) {
  const { t } = useTranslation()
  const config = STATUS_CONFIG[props.status]

  return (
    <StatusBadge
      label={t(config.label)}
      variant={config.variant}
      copyable={false}
    />
  )
}

function CommissionTimes(props: { commission: AdminAffiliateCommission }) {
  const { t } = useTranslation()

  return (
    <div className='min-w-44 space-y-1 text-xs'>
      <div>
        <span className='text-muted-foreground'>{t('Created At')}: </span>
        {formatTimestamp(props.commission.created_at)}
      </div>
      <div>
        <span className='text-muted-foreground'>{t('Available At')}: </span>
        {props.commission.available_at
          ? formatTimestamp(props.commission.available_at)
          : '-'}
      </div>
      {props.commission.transferred_at ? (
        <div>
          <span className='text-muted-foreground'>{t('Transferred At')}: </span>
          {formatTimestamp(props.commission.transferred_at)}
        </div>
      ) : null}
      {props.commission.reversed_at ? (
        <div>
          <span className='text-muted-foreground'>{t('Reversed At')}: </span>
          {formatTimestamp(props.commission.reversed_at)}
        </div>
      ) : null}
    </div>
  )
}

export function AffiliateCommissionsDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: { id: number; username?: string }
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [reverseTarget, setReverseTarget] =
    useState<AdminAffiliateCommission | null>(null)
  const [reason, setReason] = useState('')
  const commissionsQuery = useQuery({
    queryKey: ['admin-affiliate-commissions', props.user.id, page],
    queryFn: () => getAdminAffiliateCommissions(props.user.id, page, PAGE_SIZE),
    enabled: props.open,
  })
  const queryResult = commissionsQuery.data
  const pageData = queryResult?.success ? queryResult.data : undefined
  const items = pageData?.items ?? []
  const total = pageData?.total ?? 0
  const loadError = commissionsQuery.isError || queryResult?.success === false

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setPage(1)
      setReverseTarget(null)
      setReason('')
    }
    props.onOpenChange(open)
  }

  const startReversal = (commission: AdminAffiliateCommission) => {
    if (!isAffiliateCommissionReversible(commission.status)) return
    setReverseTarget(commission)
    setReason('')
  }

  const normalizedReason = normalizeAffiliateCommissionReverseReason(reason)
  let commissionTableEmptyContent = <span>{t('No commissions yet')}</span>
  if (commissionsQuery.isLoading) {
    commissionTableEmptyContent = <span>{t('Loading...')}</span>
  } else if (loadError) {
    commissionTableEmptyContent = (
      <div className='flex flex-col items-center gap-2'>
        <span>{t('Failed to load commissions')}</span>
        <Button
          variant='outline'
          size='sm'
          onClick={() => void commissionsQuery.refetch()}
        >
          {t('Retry')}
        </Button>
      </div>
    )
  }

  const reverseMutation = useMutation({
    mutationFn: (input: { id: string; reason: string }) =>
      reverseAdminAffiliateCommission(input.id, input.reason),
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to reverse commission'))
        return
      }
      toast.success(t('Affiliate commission reversed'))
      setReverseTarget(null)
      setReason('')
      void queryClient.invalidateQueries({
        queryKey: ['admin-affiliate-commissions', props.user.id],
      })
      props.onSuccess()
    },
    onError: () => toast.error(t('Failed to reverse commission')),
  })

  const confirmReversal = () => {
    if (!reverseTarget || !normalizedReason) return
    reverseMutation.mutate({ id: reverseTarget.id, reason: normalizedReason })
  }

  return (
    <>
      <Sheet open={props.open} onOpenChange={handleOpenChange}>
        <SheetContent
          className={sideDrawerContentClassName('w-full sm:max-w-5xl')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Affiliate commissions')}</SheetTitle>
            <SheetDescription>
              {props.user.username || '-'} (ID: {props.user.id})
            </SheetDescription>
          </SheetHeader>

          <div className={sideDrawerFormClassName('gap-4')}>
            {commissionsQuery.isLoading || loadError || items.length === 0 ? (
              <div className='text-muted-foreground flex min-h-36 items-center justify-center rounded-md border p-4 text-center text-sm'>
                {commissionTableEmptyContent}
              </div>
            ) : (
              <StaticDataTable
                data={items}
                getRowKey={(commission) => commission.id}
                tableClassName='min-w-[1280px]'
                columns={[
                  {
                    id: 'id',
                    header: t('ID'),
                    cell: (commission) => <TableId value={commission.id} />,
                  },
                  {
                    id: 'paid',
                    header: t('Actual payment'),
                    cell: (commission) => (
                      <span className='font-medium whitespace-nowrap tabular-nums'>
                        {formatAffiliateMinor(
                          commission.paid_amount_minor,
                          commission.paid_currency
                        )}
                      </span>
                    ),
                  },
                  {
                    id: 'purchased-balance',
                    header: t('Purchased balance'),
                    cell: (commission) => (
                      <span className='font-medium whitespace-nowrap tabular-nums'>
                        {formatAffiliateQuota(commission.purchased_quota)}
                      </span>
                    ),
                  },
                  {
                    id: 'referral',
                    header: t('Referral'),
                    cell: (commission) => (
                      <div className='min-w-28 space-y-1 text-xs tabular-nums'>
                        <div>
                          <span className='text-muted-foreground'>
                            {t('User ID')}:{' '}
                          </span>
                          #{commission.referred_user_id}
                        </div>
                        <div>
                          <span className='text-muted-foreground'>
                            {t('Top-up')}:{' '}
                          </span>
                          #{commission.topup_id}
                        </div>
                      </div>
                    ),
                  },
                  {
                    id: 'commission',
                    header: t('Cash withdrawal value'),
                    cell: (commission) => (
                      <div className='space-y-1 whitespace-nowrap tabular-nums'>
                        <div>
                          {formatAffiliateMinor(
                            commission.commission_amount_minor,
                            commission.paid_currency
                          )}
                        </div>
                        <div className='text-muted-foreground text-xs'>
                          {formatAffiliateRate(commission.commission_rate_bps)}
                        </div>
                      </div>
                    ),
                  },
                  {
                    id: 'reward',
                    header: t('Balance reward'),
                    cell: (commission) => (
                      <span className='whitespace-nowrap tabular-nums'>
                        {formatAffiliateQuota(commission.reward_quota)}
                      </span>
                    ),
                  },
                  {
                    id: 'status',
                    header: t('Status'),
                    cell: (commission) => (
                      <CommissionStatus status={commission.status} />
                    ),
                  },
                  {
                    id: 'times',
                    header: t('Time'),
                    cell: (commission) => (
                      <CommissionTimes commission={commission} />
                    ),
                  },
                  {
                    id: 'reason',
                    header: t('Reversal reason'),
                    cell: (commission) => (
                      <div className='max-w-64 min-w-40 space-y-1'>
                        <div className='break-words whitespace-pre-wrap'>
                          {commission.reverse_reason || '-'}
                        </div>
                        {commission.reversed_by ? (
                          <div className='text-muted-foreground text-xs tabular-nums'>
                            {t('Admin')}: #{commission.reversed_by}
                          </div>
                        ) : null}
                      </div>
                    ),
                  },
                  {
                    id: 'actions',
                    header: t('Actions'),
                    className: 'text-right',
                    cellClassName: 'text-right',
                    cell: (commission) =>
                      isAffiliateCommissionReversible(commission.status) ? (
                        <DataTableRowActionMenu ariaLabel={t('Actions')}>
                          <DropdownMenuItem
                            variant='destructive'
                            onClick={() => startReversal(commission)}
                          >
                            {t('Reverse commission')}
                            <DropdownMenuShortcut>
                              <RotateCcw size={16} />
                            </DropdownMenuShortcut>
                          </DropdownMenuItem>
                        </DataTableRowActionMenu>
                      ) : (
                        '-'
                      ),
                  },
                ]}
              />
            )}

            {total > PAGE_SIZE ? (
              <div className='flex items-center justify-end gap-2'>
                <Button
                  variant='outline'
                  size='icon-sm'
                  disabled={commissionsQuery.isFetching || page <= 1}
                  onClick={() => setPage((value) => value - 1)}
                  aria-label={t('Previous page')}
                >
                  <ChevronLeft aria-hidden='true' />
                </Button>
                <span className='text-muted-foreground min-w-8 text-center text-sm tabular-nums'>
                  {page}
                </span>
                <Button
                  variant='outline'
                  size='icon-sm'
                  disabled={
                    commissionsQuery.isFetching || page * PAGE_SIZE >= total
                  }
                  onClick={() => setPage((value) => value + 1)}
                  aria-label={t('Next page')}
                >
                  <ChevronRight aria-hidden='true' />
                </Button>
              </div>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

      {reverseTarget ? (
        <ConfirmDialog
          open
          onOpenChange={(open) => {
            if (!open) {
              setReverseTarget(null)
              setReason('')
            }
          }}
          title={t('Reverse affiliate commission')}
          desc={t(
            "Only the affiliate commission will be reversed. The payment will not be refunded and the referred user's top-up quota will not be changed. If the commission was already transferred, it becomes commission debt and future commissions will offset it first. The reason will be visible to the affiliate and recorded in the admin audit log."
          )}
          confirmText={t('Reverse commission')}
          destructive
          disabled={!normalizedReason}
          isLoading={reverseMutation.isPending}
          handleConfirm={confirmReversal}
        >
          <div className='grid gap-2'>
            <label htmlFor='affiliate-commission-reversal-reason'>
              {t('Reversal reason')}
            </label>
            <Textarea
              id='affiliate-commission-reversal-reason'
              value={reason}
              onChange={(event) => setReason(event.currentTarget.value)}
              minLength={1}
              maxLength={255}
              aria-invalid={reason.length > 0 && !normalizedReason}
              placeholder={t('Reason (1-255 characters)')}
              autoFocus
            />
          </div>
        </ConfirmDialog>
      ) : null}
    </>
  )
}
