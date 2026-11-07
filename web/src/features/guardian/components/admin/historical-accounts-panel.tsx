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

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Field, FieldError, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { TitledCard } from '@/components/ui/titled-card'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  getGuardianHistoricalAccounts,
  saveGuardianHistoricalAccounts,
} from '../../api'
import { GUARDIAN_QUERY_KEYS } from '../../constants'
import { guardianHistoricalAccountsSchema } from '../../lib/schema'
import type { GuardianHistoricalAccounts } from '../../types'

export function GuardianHistoricalAccountsPanel() {
  const { t } = useTranslation()
  const history = useQuery({
    queryKey: GUARDIAN_QUERY_KEYS.historicalAccounts,
    queryFn: getGuardianHistoricalAccounts,
  })
  if (history.isPending) return <LoadingState />
  if (history.isError) {
    return (
      <ErrorState
        title={t('Failed to load account status')}
        onRetry={() => void history.refetch()}
      />
    )
  }
  return (
    <GuardianHistoricalAccountsForm
      initial={history.data.settings}
      baseSales={history.data.base_sales}
    />
  )
}

export function GuardianHistoricalAccountsForm({
  initial,
  baseSales,
}: {
  initial: GuardianHistoricalAccounts
  baseSales: number
}) {
  const { t, i18n } = useTranslation()
  const client = useQueryClient()
  const form = useForm<GuardianHistoricalAccounts>({
    resolver: zodResolver(
      guardianHistoricalAccountsSchema.refine(
        (value) => value.total_users <= baseSales,
        {
          path: ['total_users'],
          message: 'Historical users cannot exceed base sales',
        }
      )
    ),
    defaultValues: initial,
  })
  const fields = [
    { key: 'total_users', label: t('Historical base users') },
    { key: 'banned_only', label: t('Ban only') },
    { key: 'kyc_only', label: t('KYC only') },
    { key: 'phone_only', label: t('Phone verification only') },
    { key: 'banned_kyc', label: t('Ban and KYC only') },
    { key: 'banned_phone', label: t('Ban and phone verification only') },
    { key: 'kyc_phone', label: t('KYC and phone verification only') },
    { key: 'all_three', label: t('All three conditions flagged') },
  ] as const
  const save = useMutation({
    mutationFn: saveGuardianHistoricalAccounts,
    onSuccess: (values) => {
      form.reset(values)
      client.setQueryData(GUARDIAN_QUERY_KEYS.historicalAccounts, {
        settings: values,
        base_sales: baseSales,
      })
      void client.invalidateQueries({
        queryKey: GUARDIAN_QUERY_KEYS.accountStatistics,
      })
      toast.success(t('Account status saved'))
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save account status')),
  })
  return (
    <TitledCard
      title={t('Historical user configuration')}
      description={t(
        'Historical users cannot exceed base sales. Each user belongs to one historical group. Link returning purchasers in their account status to avoid counting them twice.'
      )}
      disableHoverEffect
    >
      <p className='text-muted-foreground mb-4 text-sm'>
        {t('Base sales limit: {{total}}', {
          total: formatNumber(
            baseSales,
            toIntlLocale(i18n.resolvedLanguage || i18n.language)
          ),
        })}
      </p>
      <form
        className='space-y-4'
        onSubmit={form.handleSubmit((values) => save.mutate(values))}
      >
        <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
          {fields.map(({ key, label }) => (
            <Field key={key} data-invalid={!!form.formState.errors[key]}>
              <FieldLabel htmlFor={`history-${key}`}>{label}</FieldLabel>
              <Input
                id={`history-${key}`}
                type='number'
                min={0}
                max={1000000000}
                step={1}
                aria-invalid={!!form.formState.errors[key]}
                {...form.register(key, { valueAsNumber: true })}
              />
              {form.formState.errors[key] && (
                <FieldError>
                  {t(form.formState.errors[key]?.message || 'Invalid value')}
                </FieldError>
              )}
            </Field>
          ))}
        </div>
        <Button type='submit' disabled={save.isPending}>
          {save.isPending ? t('Saving...') : t('Save')}
        </Button>
      </form>
    </TitledCard>
  )
}
