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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import { handleServerError } from '@/lib/handle-server-error'

import { updateGuardianQueueSetting } from '../../api'
import { GUARDIAN_QUERY_KEYS } from '../../constants'
import {
  guardianQueueSettingSchema,
  type GuardianQueueSettingForm,
} from '../../lib/schema'
import type { GuardianQueueSetting } from '../../types'

function toForm(setting: GuardianQueueSetting): GuardianQueueSettingForm {
  return {
    daily_capacity: setting.daily_capacity,
    extra_slots: setting.extra_slots,
    paused: setting.paused,
    resume_at: setting.resume_at
      ? new Date(setting.resume_at * 1000)
      : undefined,
  }
}

export function GuardianQueueSettingsForm(props: {
  setting: GuardianQueueSetting
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<GuardianQueueSettingForm>({
    resolver: zodResolver(guardianQueueSettingSchema),
    defaultValues: toForm(props.setting),
  })
  const paused = useWatch({ control: form.control, name: 'paused' })
  const save = useMutation({
    mutationFn: (values: GuardianQueueSettingForm) =>
      updateGuardianQueueSetting({
        daily_capacity: values.daily_capacity,
        extra_slots: values.extra_slots,
        paused: values.paused,
        resume_at:
          values.paused && values.resume_at
            ? Math.floor(values.resume_at.getTime() / 1000)
            : 0,
      }),
    onSuccess: (setting) => {
      toast.success(t('Queue settings saved'))
      form.reset(toForm(setting))
      void queryClient.invalidateQueries({
        queryKey: GUARDIAN_QUERY_KEYS.adminQueue,
      })
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save queue settings')),
  })
  const errors = form.formState.errors
  const fieldError = (message?: string) =>
    message ? <FieldError>{t(message)}</FieldError> : null

  return (
    <TitledCard
      title={t('Queue settings')}
      description={t(
        'Buyers see these changes the next time their orders page refreshes.'
      )}
      disableHoverEffect
    >
      <form
        className='space-y-4'
        onSubmit={form.handleSubmit((values) => save.mutate(values))}
      >
        <div className='grid gap-4 sm:grid-cols-2'>
          <Field data-invalid={!!errors.daily_capacity}>
            <FieldLabel htmlFor='guardian-queue-capacity'>
              {t('Daily start limit')}
            </FieldLabel>
            <Input
              id='guardian-queue-capacity'
              type='number'
              min={0}
              max={1000}
              aria-invalid={!!errors.daily_capacity}
              {...form.register('daily_capacity', { valueAsNumber: true })}
            />
            <FieldDescription>
              {t('Orders started per day. 0 means no limit.')}
            </FieldDescription>
            {fieldError(errors.daily_capacity?.message)}
          </Field>
          <Field data-invalid={!!errors.extra_slots}>
            <FieldLabel htmlFor='guardian-queue-extra'>
              {t('Extra queue slots')}
            </FieldLabel>
            <Input
              id='guardian-queue-extra'
              type='number'
              min={0}
              max={1000}
              aria-invalid={!!errors.extra_slots}
              {...form.register('extra_slots', { valueAsNumber: true })}
            />
            <FieldDescription>
              {t('Counted as orders ahead of every buyer in the queue.')}
            </FieldDescription>
            {fieldError(errors.extra_slots?.message)}
          </Field>
        </div>
        <Field orientation='horizontal' className='items-center'>
          <Controller
            control={form.control}
            name='paused'
            render={({ field }) => (
              <Switch
                id='guardian-queue-paused'
                checked={field.value}
                onCheckedChange={field.onChange}
              />
            )}
          />
          <FieldLabel htmlFor='guardian-queue-paused'>
            {t('Pause scheduling')}
          </FieldLabel>
        </Field>
        {paused && (
          <Field data-invalid={!!errors.resume_at}>
            <FieldLabel>{t('Resume at')}</FieldLabel>
            <Controller
              control={form.control}
              name='resume_at'
              render={({ field }) => (
                <DateTimePicker
                  value={field.value}
                  onChange={field.onChange}
                  className='max-w-sm'
                />
              )}
            />
            <FieldDescription>
              {t('Leave empty to stay paused until you turn it off.')}
            </FieldDescription>
            {fieldError(errors.resume_at?.message)}
          </Field>
        )}
        <Button type='submit' disabled={save.isPending}>
          {save.isPending ? t('Saving...') : t('Save')}
        </Button>
      </form>
    </TitledCard>
  )
}
