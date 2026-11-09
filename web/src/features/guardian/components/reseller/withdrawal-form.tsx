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
import { useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Field, FieldError, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'

import {
  withdrawalSchema,
  type WithdrawalForm,
} from '../../lib/reseller-schema'
import { resellerWrite } from '../../reseller-api'

export function ResellerWithdrawalForm() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const verification = useSecureVerification()
  const previous = useRef<{ data: string; key: string } | null>(null)
  const form = useForm<WithdrawalForm>({
    resolver: zodResolver(withdrawalSchema),
    defaultValues: { amount: '', account: '', payee_name: '' },
  })
  const mutation = useMutation({
    mutationFn: async (values: WithdrawalForm) => {
      const data = JSON.stringify(values)
      if (!previous.current || previous.current.data !== data) {
        previous.current = { data, key: crypto.randomUUID() }
      }
      const request = {
        amount_cents: Math.round(Number(values.amount) * 100),
        account: values.account,
        payee_name: values.payee_name,
        request_key: previous.current.key,
      }
      const proof = await verification.requestVerification({
        scope: 'guardian.withdrawal',
        context: request,
        title: t('Verify withdrawal'),
      })
      if (!proof) return false
      await resellerWrite(
        '/reseller/withdrawals',
        request,
        'post',
        proof.proof_token
      )
      return true
    },
    onSuccess: async (submitted) => {
      if (!submitted) return
      previous.current = null
      form.reset()
      await client.invalidateQueries({ queryKey: ['guardian'] })
      toast.success(t('Withdrawal requested'))
    },
  })
  return (
    <>
      <form
        className='grid gap-4 sm:grid-cols-3'
        onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
      >
        {(
          [
            { name: 'amount', label: 'Withdrawal amount (CNY)' },
            { name: 'account', label: 'Alipay account' },
            { name: 'payee_name', label: 'Verified payee name' },
          ] as const
        ).map((f) => (
          <Field key={f.name} data-invalid={!!form.formState.errors[f.name]}>
            <FieldLabel htmlFor={`withdraw-${f.name}`}>{t(f.label)}</FieldLabel>
            <Input
              id={`withdraw-${f.name}`}
              disabled={mutation.isPending}
              {...form.register(f.name)}
            />
            {form.formState.errors[f.name] && (
              <FieldError>
                {t(form.formState.errors[f.name]?.message ?? '')}
              </FieldError>
            )}
          </Field>
        ))}
        <p className='text-muted-foreground text-sm sm:col-span-3'>
          {t(
            'Minimum CNY 1. Payout fees are deducted from this amount. The platform verifies the recipient before a manual Alipay transfer.'
          )}
        </p>
        <Button type='submit' disabled={mutation.isPending}>
          {t('Request withdrawal')}
        </Button>
      </form>
      <SecureVerificationDialog {...verification.dialogProps} />
    </>
  )
}
