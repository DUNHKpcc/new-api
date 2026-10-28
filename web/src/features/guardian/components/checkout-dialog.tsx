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
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ShieldAlert } from 'lucide-react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Textarea } from '@/components/ui/textarea'
import { submitPaymentForm } from '@/features/wallet/lib/payment'
import { toIntlLocale } from '@/i18n/languages'
import { handleServerError } from '@/lib/handle-server-error'

import { createGuardianOrder } from '../api'
import { useGuardianLabels } from '../hooks/use-guardian-labels'
import { formatCny } from '../lib/price'
import { guardianOrderSchema, type GuardianOrderForm } from '../lib/schema'
import type { GuardianPayMethod, GuardianService } from '../types'

const FORM_ID = 'guardian-checkout'

export function CheckoutDialog(props: {
  service: GuardianService
  payMethods: GuardianPayMethod[]
  onClose: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const navigate = useNavigate()
  const form = useForm<GuardianOrderForm>({
    resolver: zodResolver(guardianOrderSchema),
    defaultValues: {
      payment_method: props.payMethods[0]?.type ?? '',
      route: props.service.route === 'any' ? 'unsure' : props.service.route,
      os: 'macos',
      contact_type: 'telegram',
      contact: '',
      note: '',
    },
  })
  const checkout = useMutation({
    mutationFn: (values: GuardianOrderForm) =>
      createGuardianOrder({ service_id: props.service.id, ...values }),
    onSuccess: (result) => {
      submitPaymentForm(result.pay_url, result.pay_params)
      toast.success(t('Payment page opened'))
      props.onClose()
      void navigate({ to: '/guardian/orders' })
    },
    onError: (error) => handleServerError(error, t('Payment request failed')),
  })
  const errors = form.formState.errors
  const price = formatCny(props.service.price_cents, locale)

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !checkout.isPending) props.onClose()
      }}
      title={props.service.title}
      description={props.service.subtitle || undefined}
      contentClassName='guardian-root max-sm:w-[calc(100vw-1.5rem)] sm:max-w-lg'
      titleClassName='g-heading text-xl'
      footer={
        <>
          <Button
            variant='outline'
            onClick={props.onClose}
            disabled={checkout.isPending}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={FORM_ID} disabled={checkout.isPending}>
            {checkout.isPending
              ? t('Opening payment...')
              : t('Pay {{price}}', { price })}
          </Button>
        </>
      }
    >
      <form
        id={FORM_ID}
        className='space-y-5'
        noValidate
        onSubmit={form.handleSubmit((values) => checkout.mutate(values))}
      >
        {props.service.description && (
          <div className='max-h-40 space-y-3 overflow-y-auto rounded-lg bg-[var(--g-paper-sunk)] p-4 text-sm leading-6 text-[var(--g-ink-soft)]'>
            {props.service.description.split(/\n{2,}/).map((paragraph) => (
              <p key={paragraph} className='whitespace-pre-line'>
                {paragraph}
              </p>
            ))}
          </div>
        )}

        <div className='grid gap-4 sm:grid-cols-2'>
          <Field>
            <FieldLabel htmlFor='guardian-route'>{t('Route')}</FieldLabel>
            <NativeSelect id='guardian-route' {...form.register('route')}>
              {Object.entries(labels.orderRoute).map(([value, label]) => (
                <NativeSelectOption key={value} value={value}>
                  {label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field>
            <FieldLabel htmlFor='guardian-os'>
              {t('Operating system')}
            </FieldLabel>
            <NativeSelect id='guardian-os' {...form.register('os')}>
              {Object.entries(labels.os).map(([value, label]) => (
                <NativeSelectOption key={value} value={value}>
                  {label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
        </div>

        <div className='grid gap-4 sm:grid-cols-[9rem_1fr]'>
          <Field>
            <FieldLabel htmlFor='guardian-contact-type'>
              {t('Contact via')}
            </FieldLabel>
            <NativeSelect
              id='guardian-contact-type'
              {...form.register('contact_type')}
            >
              {Object.entries(labels.contactType).map(([value, label]) => (
                <NativeSelectOption key={value} value={value}>
                  {label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field data-invalid={!!errors.contact}>
            <FieldLabel htmlFor='guardian-contact'>{t('Contact')}</FieldLabel>
            <Input
              id='guardian-contact'
              maxLength={128}
              autoComplete='off'
              aria-invalid={!!errors.contact}
              aria-describedby={
                errors.contact ? 'guardian-contact-error' : undefined
              }
              {...form.register('contact')}
            />
            {errors.contact && (
              <FieldError id='guardian-contact-error'>
                {t(errors.contact.message ?? '')}
              </FieldError>
            )}
          </Field>
        </div>

        <Field data-invalid={!!errors.note}>
          <FieldLabel htmlFor='guardian-note'>
            {t('Note (optional)')}
          </FieldLabel>
          <Textarea
            id='guardian-note'
            rows={3}
            maxLength={1000}
            aria-invalid={!!errors.note}
            aria-describedby='guardian-note-help'
            {...form.register('note')}
          />
          <FieldDescription id='guardian-note-help'>
            {t(
              'Your current setup, budget, or anything we should know before scheduling.'
            )}
          </FieldDescription>
          {errors.note && (
            <FieldError>{t(errors.note.message ?? '')}</FieldError>
          )}
        </Field>

        <Alert className='border-[color-mix(in_srgb,var(--g-tier-c)_35%,transparent)]'>
          <ShieldAlert className='text-[var(--g-tier-c)]' aria-hidden='true' />
          <AlertDescription>
            {t(
              'Never put passwords, keys or subscription links in this form. We arrange remote access privately after payment, and you can revoke it at any time.'
            )}
          </AlertDescription>
        </Alert>

        <Field data-invalid={!!errors.payment_method}>
          <FieldLabel id='guardian-pay-label'>{t('Payment method')}</FieldLabel>
          <Controller
            control={form.control}
            name='payment_method'
            render={({ field }) => (
              <RadioGroup
                aria-labelledby='guardian-pay-label'
                value={field.value}
                onValueChange={(value) => field.onChange(String(value))}
                className='grid-cols-2'
              >
                {props.payMethods.map((method) => (
                  <label
                    key={method.type}
                    className='g-rule flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 text-sm has-data-checked:border-[var(--g-ink)]'
                  >
                    <RadioGroupItem value={method.type} />
                    {method.name || method.type}
                  </label>
                ))}
              </RadioGroup>
            )}
          />
          {errors.payment_method && (
            <FieldError>{t(errors.payment_method.message ?? '')}</FieldError>
          )}
        </Field>
      </form>
    </Dialog>
  )
}
