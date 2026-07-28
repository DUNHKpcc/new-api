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
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { isUserSubscriptionActive } from '@/features/subscriptions/lib/user-subscription-policy'
import { formatTimestamp } from '@/lib/format'

import type { User } from '../types'
import { UserQuotaCell } from './user-quota-cell'

function renderNoGiftSubscription(label: string) {
  return <StatusBadge label={label} variant='neutral' copyable={false} />
}

export function usePccAgentUsersColumns(): ColumnDef<User>[] {
  const { t } = useTranslation()

  return [
    {
      id: 'pcc_agent_access',
      header: t('PccAgent Access'),
      cell: ({ row }) => {
        const summary = row.original.pcc_agent_summary
        if (!summary) return null

        return (
          <div className='flex min-w-[150px] flex-col items-start gap-1'>
            <StatusBadge
              label={`${t('Active Devices')}: ${summary.active_device_count}`}
              variant={summary.active_device_count > 0 ? 'success' : 'neutral'}
              copyable={false}
            />
            <StatusBadge
              label={
                summary.wechat_verified
                  ? t('WeChat Verified')
                  : t('WeChat Not Verified')
              }
              variant={summary.wechat_verified ? 'info' : 'neutral'}
              copyable={false}
            />
          </div>
        )
      },
      enableSorting: false,
      size: 180,
      meta: { mobileOrder: 50 },
    },
    {
      id: 'pcc_agent_gift',
      header: t('Gift Subscription'),
      cell: ({ row }) => {
        const gift = row.original.pcc_agent_summary?.gift
        if (!gift) return renderNoGiftSubscription(t('No Gift Subscription'))

        // eslint-disable-next-line react-hooks/purity
        const isActive = isUserSubscriptionActive(gift, Date.now() / 1000)
        let statusLabel = t('Expired')
        let statusVariant: 'success' | 'neutral' = 'neutral'
        if (isActive) {
          statusLabel = t('Active')
          statusVariant = 'success'
        } else if (gift.status === 'cancelled') {
          statusLabel = t('Invalidated')
        }

        return (
          <div className='flex min-w-[160px] flex-col items-start gap-1'>
            <span
              className='max-w-[180px] truncate font-medium'
              title={gift.plan_title || `#${gift.plan_id}`}
            >
              {gift.plan_title || `#${gift.plan_id}`}
            </span>
            <StatusBadge
              label={statusLabel}
              variant={statusVariant}
              copyable={false}
            />
          </div>
        )
      },
      enableSorting: false,
      size: 210,
      meta: { mobileOrder: 60 },
    },
    {
      id: 'pcc_agent_gift_quota',
      header: t('Gift Quota'),
      cell: ({ row }) => {
        const gift = row.original.pcc_agent_summary?.gift
        if (!gift) return renderNoGiftSubscription(t('No Gift Subscription'))
        if (gift.amount_total <= 0) {
          return (
            <StatusBadge
              label={t('Unlimited')}
              variant='info'
              copyable={false}
            />
          )
        }
        return (
          <UserQuotaCell
            used={gift.amount_used}
            remaining={gift.amount_remaining}
          />
        )
      },
      enableSorting: false,
      size: 260,
      minSize: 240,
      meta: { mobileOrder: 70 },
    },
    {
      id: 'pcc_agent_gift_schedule',
      header: t('Gift Schedule'),
      cell: ({ row }) => {
        const gift = row.original.pcc_agent_summary?.gift
        if (!gift) return renderNoGiftSubscription(t('No Gift Subscription'))
        return (
          <div className='min-w-[190px] space-y-1 text-sm'>
            <div>
              <span className='text-muted-foreground'>{t('Next reset')}:</span>{' '}
              {formatTimestamp(gift.next_reset_time)}
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Expires at')}:</span>{' '}
              {formatTimestamp(gift.end_time)}
            </div>
          </div>
        )
      },
      enableSorting: false,
      size: 230,
      meta: { mobileOrder: 80 },
    },
  ]
}
