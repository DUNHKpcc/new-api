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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import type { CampaignStatus, DeliveryStatus, EmailCategory } from '../types'

export function EmailStatus(props: {
  status: CampaignStatus | DeliveryStatus
}) {
  const { t } = useTranslation()
  const labels = {
    draft: t('Draft'),
    queued: t('Queued'),
    sending: t('Sending'),
    completed: t('Completed'),
    cancelled: t('Cancelled'),
    pending: t('Pending delivery'),
    retry: t('Retry scheduled'),
    sent: t('Accepted by SMTP'),
    failed: t('Failed'),
    skipped: t('Skipped'),
    uncertain: t('Delivery uncertain'),
  }
  return (
    <Badge
      variant={
        props.status === 'failed' || props.status === 'uncertain'
          ? 'destructive'
          : 'secondary'
      }
    >
      {labels[props.status] ?? props.status}
    </Badge>
  )
}
export function EmailCategoryLabel(props: { category: EmailCategory }) {
  const { t } = useTranslation()
  return props.category === 'promotion'
    ? t('Promotional emails')
    : t('Platform updates')
}
