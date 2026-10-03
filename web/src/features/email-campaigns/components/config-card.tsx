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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Mail } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { TitledCard } from '@/components/ui/titled-card'
import { handleServerError } from '@/lib/handle-server-error'

import { getEmailConfig, updateEmailConfig } from '../api'
import { emailRateSchema } from '../lib/schema'

export function EmailConfigCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const config = useQuery({
    queryKey: ['email-config'],
    queryFn: getEmailConfig,
  })
  const form = useForm<{ rate_per_minute: number }>({
    resolver: zodResolver(emailRateSchema),
    values: { rate_per_minute: config.data?.rate_per_minute ?? 20 },
  })
  const save = useMutation({
    mutationFn: (value: { rate_per_minute: number }) =>
      updateEmailConfig(value.rate_per_minute),
    onSuccess: (value) => {
      queryClient.setQueryData(['email-config'], value)
      toast.success(t('Email sending rate saved'))
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save email sending rate')),
  })
  return (
    <TitledCard
      title={t('Email delivery settings')}
      description={t(
        'Use the existing SMTP configuration for promotional emails and platform updates.'
      )}
      icon={<Mail className='size-4' />}
      disableHoverEffect
      action={
        <Button
          variant='outline'
          size='sm'
          render={
            <Link
              to='/system-settings/operations/$section'
              params={{ section: 'email' }}
            />
          }
        >
          {t('SMTP settings')}
        </Button>
      }
    >
      {config.isPending && <LoadingState size='sm' />}
      {config.isError && (
        <ErrorState
          title={t('Failed to load email settings')}
          onRetry={() => void config.refetch()}
        />
      )}
      {config.data && (
        <form
          className='space-y-3'
          noValidate
          onSubmit={form.handleSubmit((value) => save.mutate(value))}
        >
          <p className='text-sm'>
            {config.data.smtp_configured
              ? t('SMTP configured')
              : t('SMTP is not configured')}
          </p>
          {!config.data.server_address_ready && (
            <p role='alert' className='text-destructive text-sm'>
              {t('Configure a public server address for unsubscribe links.')}
            </p>
          )}
          <div className='flex flex-col gap-2 sm:flex-row sm:items-end'>
            <div className='space-y-2'>
              <Label htmlFor='email-rate'>
                {t('Maximum campaign emails per minute')}
              </Label>
              <Input
                id='email-rate'
                type='number'
                min={1}
                max={120}
                step={1}
                className='sm:w-48'
                {...form.register('rate_per_minute', { valueAsNumber: true })}
                aria-invalid={!!form.formState.errors.rate_per_minute}
                aria-describedby='email-rate-help'
              />
            </div>
            <Button type='submit' disabled={save.isPending}>
              {save.isPending ? t('Saving...') : t('Save')}
            </Button>
          </div>
          <p id='email-rate-help' className='text-muted-foreground text-xs'>
            {t(
              'Choose 1–120 emails per minute across all campaigns. Security and verification emails are unaffected.'
            )}
          </p>
          {form.formState.errors.rate_per_minute && (
            <p role='alert' className='text-destructive text-sm'>
              {t('Enter a whole number between 1 and 120')}
            </p>
          )}
          {save.isError && (
            <p role='alert' className='text-destructive text-sm'>
              {t('Failed to save email sending rate')}
            </p>
          )}
        </form>
      )}
    </TitledCard>
  )
}
