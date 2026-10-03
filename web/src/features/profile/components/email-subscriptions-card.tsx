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
import { Mail } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import {
  getEmailSubscriptions,
  updateEmailSubscriptions,
} from '@/features/email-campaigns/api'
import { handleServerError } from '@/lib/handle-server-error'

export function EmailSubscriptionsCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const subscriptions = useQuery({
    queryKey: ['email-subscriptions'],
    queryFn: getEmailSubscriptions,
  })
  const update = useMutation({
    mutationFn: updateEmailSubscriptions,
    onSuccess: (value) => {
      queryClient.setQueryData(['email-subscriptions'], value)
      toast.success(t('Email preferences saved'))
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save email preferences')),
  })
  return (
    <TitledCard
      title={t('Email subscriptions')}
      description={t(
        'Choose which emails to receive at your bound email address. Security and verification emails are unaffected.'
      )}
      icon={<Mail className='size-4' />}
      disableHoverEffect
    >
      {subscriptions.isPending && <LoadingState size='sm' />}
      {subscriptions.isError && (
        <ErrorState
          title={t('Failed to load email preferences')}
          onRetry={() => void subscriptions.refetch()}
        />
      )}
      {subscriptions.data && (
        <div className='space-y-4' aria-busy={update.isPending}>
          <div className='flex items-center justify-between gap-4'>
            <Label htmlFor='email-promotion'>{t('Promotional emails')}</Label>
            <Switch
              id='email-promotion'
              checked={subscriptions.data.promotion}
              disabled={update.isPending}
              onCheckedChange={(promotion) => {
                if (subscriptions.data) {
                  update.mutate({ ...subscriptions.data, promotion })
                }
              }}
            />
          </div>
          <div className='flex items-center justify-between gap-4'>
            <Label htmlFor='email-platform'>{t('Platform updates')}</Label>
            <Switch
              id='email-platform'
              checked={subscriptions.data.platform}
              disabled={update.isPending}
              onCheckedChange={(platform) => {
                if (subscriptions.data) {
                  update.mutate({ ...subscriptions.data, platform })
                }
              }}
            />
          </div>
          {update.isError && (
            <p role='alert' className='text-destructive text-sm'>
              {t('Failed to save email preferences')}
            </p>
          )}
        </div>
      )}
    </TitledCard>
  )
}
