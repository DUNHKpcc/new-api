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
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Field, FieldError, FieldLabel } from '@/components/ui/field'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'

import {
  getGuardianReseller,
  setGuardianResellerAccess,
  type GuardianReseller,
} from '../../reseller-api'

const schema = z.object({
  tier: z.number().int().min(0).max(3),
  status: z.enum(['active', 'suspended']),
  reason: z.string().trim().min(1, 'Please enter a reason').max(500),
})

export function GuardianResellerAccessDialog(props: {
  user: { id: number; username: string }
  onClose: () => void
}) {
  const { t } = useTranslation()
  const profile = useQuery({
    queryKey: ['guardian', 'reseller', props.user.id],
    queryFn: () => getGuardianReseller(props.user.id),
  })

  return (
    <Dialog
      open
      title={t('Guardian reseller access')}
      description={props.user.username}
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
    >
      {profile.isPending && <LoadingState />}
      {profile.isError && <ErrorState onRetry={() => void profile.refetch()} />}
      {profile.isSuccess && (
        <ResellerAccessForm
          userId={props.user.id}
          profile={profile.data}
          onClose={props.onClose}
        />
      )}
    </Dialog>
  )
}

function ResellerAccessForm(props: {
  userId: number
  profile: GuardianReseller | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      tier: props.profile?.tier ?? 1,
      status: props.profile?.status ?? 'active',
      reason: '',
    },
  })
  const save = useMutation({
    mutationFn: (values: z.infer<typeof schema>) =>
      setGuardianResellerAccess(props.userId, values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['guardian'] })
      toast.success(t('Guardian reseller access saved'))
      props.onClose()
    },
  })

  return (
    <form
      className='space-y-4'
      onSubmit={form.handleSubmit((values) => save.mutate(values))}
    >
      <p className='text-muted-foreground text-sm'>
        {t(
          'Guardian access is separate from recharge commissions. Suspending sales preserves existing orders and earnings.'
        )}
      </p>
      <Field>
        <FieldLabel htmlFor='guardian-reseller-status'>
          {t('Status')}
        </FieldLabel>
        <NativeSelect
          id='guardian-reseller-status'
          disabled={save.isPending}
          {...form.register('status')}
        >
          <NativeSelectOption value='active'>
            {t('Sales enabled')}
          </NativeSelectOption>
          <NativeSelectOption value='suspended' disabled={!props.profile}>
            {t('Sales suspended')}
          </NativeSelectOption>
        </NativeSelect>
      </Field>
      {!props.profile && (
        <Field>
          <FieldLabel htmlFor='guardian-reseller-tier'>
            {t('Reseller tier')}
          </FieldLabel>
          <NativeSelect
            id='guardian-reseller-tier'
            {...form.register('tier', { valueAsNumber: true })}
          >
            {[1, 2, 3].map((tier) => (
              <NativeSelectOption key={tier} value={tier}>
                {t('Tier {{tier}}', { tier })}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
      )}
      <Field data-invalid={!!form.formState.errors.reason}>
        <FieldLabel htmlFor='guardian-reseller-reason'>
          {t('Reason')}
        </FieldLabel>
        <Textarea
          id='guardian-reseller-reason'
          maxLength={500}
          disabled={save.isPending}
          aria-invalid={!!form.formState.errors.reason}
          {...form.register('reason')}
        />
        {form.formState.errors.reason && (
          <FieldError>{t('Please enter a reason')}</FieldError>
        )}
      </Field>
      <div className='flex justify-end gap-2'>
        <Button
          type='button'
          variant='outline'
          onClick={props.onClose}
          disabled={save.isPending}
        >
          {t('Cancel')}
        </Button>
        <Button type='submit' disabled={save.isPending}>
          {save.isPending ? t('Saving...') : t('Save')}
        </Button>
      </div>
    </form>
  )
}
