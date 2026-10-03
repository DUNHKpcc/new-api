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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { previewEmailCampaign, queueEmailCampaign } from '../api'
import type { EmailCampaign } from '../types'
import { EmailCategoryLabel } from './email-status'

export function EmailQueueDialog(props: {
  campaign: EmailCampaign
  onClose: () => void
  onQueued: () => void
  sendingReady: boolean
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const preview = useQuery({
    queryKey: ['email-campaigns', props.campaign.id, 'preview'],
    queryFn: () => previewEmailCampaign(props.campaign.id),
    retry: false,
    staleTime: 0,
  })
  const queue = useMutation({
    mutationFn: () => queueEmailCampaign(props.campaign.id),
    onSuccess: () => {
      toast.success(t('Email campaign queued'))
      props.onQueued()
      props.onClose()
    },
    onError: (error) =>
      handleServerError(error, t('Failed to queue email campaign')),
  })
  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open && !queue.isPending) props.onClose()
      }}
      title={t('Preview and queue email campaign')}
      desc={t(
        'Queueing starts delivery to subscribed recipients. The subject, message and audience cannot be edited afterwards.'
      )}
      confirmText={t('Confirm and queue')}
      isLoading={queue.isPending}
      disabled={
        !props.sendingReady ||
        preview.isFetching ||
        preview.isError ||
        !preview.data?.eligible
      }
      handleConfirm={() => queue.mutate()}
    >
      <div className='max-h-[55svh] space-y-3 overflow-y-auto'>
        <p className='text-sm'>
          <EmailCategoryLabel category={props.campaign.category} />
        </p>
        <h3 className='font-medium break-words'>{props.campaign.subject}</h3>
        <div className='bg-muted/40 rounded-md border p-3 text-sm break-words whitespace-pre-wrap'>
          {props.campaign.body}
        </div>
        {preview.isPending && <LoadingState size='sm' />}
        {preview.isError && (
          <ErrorState
            className='min-h-24'
            title={t('Failed to preview recipients')}
            onRetry={() => void preview.refetch()}
          />
        )}
        {preview.data && (
          <p role='status' className='text-sm font-medium'>
            {t('Eligible recipients: {{count}}', {
              count: formatNumber(preview.data.eligible, locale),
            })}
          </p>
        )}
        <p className='text-muted-foreground text-xs'>
          {t(
            'Recipients are checked again before each send. Unsubscribed users are skipped.'
          )}
        </p>
        {!props.sendingReady && (
          <p role='alert' className='text-destructive text-sm'>
            {t('Configure SMTP and a public server address before queueing.')}
          </p>
        )}
        {queue.isError && (
          <p role='alert' className='text-destructive text-sm'>
            {t('Failed to queue email campaign')}
          </p>
        )}
      </div>
    </ConfirmDialog>
  )
}
