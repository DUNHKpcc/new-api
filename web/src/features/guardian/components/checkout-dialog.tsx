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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ShieldAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Controller, FormProvider, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldError, FieldLabel } from '@/components/ui/field'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Textarea } from '@/components/ui/textarea'
import { submitPaymentForm } from '@/features/wallet/lib/payment'
import { toIntlLocale } from '@/i18n/languages'
import { handleServerError } from '@/lib/handle-server-error'
import { useAuthStore } from '@/stores/auth-store'

import { createGuardianOrder } from '../api'
import { useGuardianLabels } from '../hooks/use-guardian-labels'
import { formatCny, guardianPeriodPrice } from '../lib/price'
import { guardianOrderSchema, type GuardianOrderForm } from '../lib/schema'
import { resellerRead } from '../reseller-api'
import type {
  GuardianBillingPeriod,
  GuardianPayMethod,
  GuardianService,
} from '../types'
import { ClaudeSubscriptionAddon } from './claude-subscription-addon'
import { GuardianContactFields } from './contact-fields'
import { DiscoveryOptions } from './discovery-options'

const FORM_ID = 'guardian-checkout'

export function CheckoutDialog(props: {
  resellerCode?: string
  customerServiceId?: number
  directConfirmationRequired?: boolean
  billingPeriod?: GuardianBillingPeriod
  addClaudeMax?: boolean
  addClaudePro?: boolean
  addClaudeUltra?: boolean
  service: GuardianService
  payMethods: GuardianPayMethod[]
  onClose: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const email = useAuthStore((state) => state.auth.user?.email)
  const maskedEmail = email ? email.replace(/^(.).*(@.*)$/, '$1***$2') : ''
  const navigate = useNavigate()
  const [accepted, setAccepted] = useState(false)
  const needsConsent =
    !!props.resellerCode || !!props.directConfirmationRequired
  const [addClaudeMax, setAddClaudeMax] = useState(!!props.addClaudeMax)
  const [addClaudePro, setAddClaudePro] = useState(!!props.addClaudePro)
  const [addClaudeUltra, setAddClaudeUltra] = useState(!!props.addClaudeUltra)
  const addonPrice =
    props.service.route === 'mac_mini' && addClaudeMax
      ? (props.service.claude_max_price_cents ?? 0)
      : 0
  const proPrice =
    props.service.route === 'mac_mini' && addClaudePro
      ? (props.service.claude_pro_price_cents ?? 0)
      : 0
  const ultraPrice =
    props.service.route === 'mac_mini' && addClaudeUltra
      ? (props.service.claude_ultra_price_cents ?? 0)
      : 0
  const unavailable =
    (!!props.service.out_of_stock && !props.customerServiceId) ||
    (addClaudePro && !!props.service.claude_pro_out_of_stock) ||
    (addClaudeMax && !!props.service.claude_max_out_of_stock) ||
    (addClaudeUltra && !!props.service.claude_ultra_out_of_stock)
  const form = useForm<GuardianOrderForm>({
    resolver: zodResolver(guardianOrderSchema),
    defaultValues: {
      payment_method: props.payMethods[0]?.type ?? '',
      route: props.service.route === 'any' ? 'unsure' : props.service.route,
      os: 'macos',
      contact_type: 'telegram',
      contact: '',
      note: '',
      invoice_information: '',
    },
  })
  const previous = useQuery({
    queryKey: ['guardian', 'checkout-context'],
    queryFn: () =>
      resellerRead<{
        discovery_channel: GuardianOrderForm['discovery_channel'] | ''
      }>('/checkout-context'),
    retry: false,
    meta: { errorToast: false },
  })
  useEffect(() => {
    if (
      previous.data?.discovery_channel &&
      !form.getFieldState('discovery_channel').isDirty
    ) {
      form.setValue('discovery_channel', previous.data.discovery_channel)
    }
  }, [previous.data, form])
  const checkout = useMutation({
    mutationFn: (values: GuardianOrderForm) =>
      createGuardianOrder({
        service_id: props.service.id,
        ...(props.resellerCode || props.customerServiceId
          ? {
              reseller_code: props.resellerCode,
              customer_service_id: props.customerServiceId,
              expected_amount_cents:
                hostingPrice + addonPrice + ultraPrice + proPrice,
              accept_refund_fee: accepted,
              accept_direct_renewal:
                !!props.directConfirmationRequired && accepted,
            }
          : {}),
        ...values,
        invoice_information: props.service.invoice_enabled
          ? values.invoice_information || undefined
          : undefined,
        ...(props.service.route === 'mac_mini'
          ? {
              billing_period: props.billingPeriod ?? 'monthly',
              ...(addClaudeMax ? { add_claude_max: true } : {}),
              ...(addClaudePro ? { add_claude_pro: true } : {}),
              ...(addClaudeUltra ? { add_claude_ultra: true } : {}),
            }
          : {}),
      }),
    onSuccess: (result) => {
      submitPaymentForm(result.pay_url, result.pay_params)
      toast.success(t('Payment page opened'))
      props.onClose()
      void navigate({ to: '/guardian/orders' })
    },
    onError: (error) => handleServerError(error, t('Payment request failed')),
  })
  const errors = form.formState.errors
  const hostingPrice =
    props.service.route === 'mac_mini'
      ? guardianPeriodPrice(props.service, props.billingPeriod ?? 'monthly')
          .total
      : props.service.price_cents
  const price = formatCny(
    hostingPrice + addonPrice + ultraPrice + proPrice,
    locale
  )

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
          <Button
            type='submit'
            form={FORM_ID}
            disabled={
              unavailable || checkout.isPending || (needsConsent && !accepted)
            }
          >
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
        onSubmit={form.handleSubmit((values) => {
          if (!unavailable && (!needsConsent || accepted)) {
            checkout.mutate(values)
          }
        })}
      >
        {props.service.route === 'mac_mini' && (
          <p className='text-sm font-medium'>
            {labels.billingPeriod[props.billingPeriod ?? 'monthly']} ·{' '}
            {formatCny(hostingPrice, locale)}
          </p>
        )}
        {props.service.route === 'mac_mini' && (
          <div className='grid auto-rows-fr grid-cols-3 gap-2 sm:gap-3'>
            <ClaudeSubscriptionAddon
              variant='pro'
              outOfStock={props.service.claude_pro_out_of_stock}
              disabled={props.service.out_of_stock && !props.customerServiceId}
              priceCents={props.service.claude_pro_price_cents ?? 0}
              checked={addClaudePro}
              onCheckedChange={(checked) => {
                setAddClaudePro(checked)
                if (checked) {
                  setAddClaudeMax(false)
                  setAddClaudeUltra(false)
                }
              }}
            />
            <ClaudeSubscriptionAddon
              variant='max'
              outOfStock={props.service.claude_max_out_of_stock}
              disabled={props.service.out_of_stock && !props.customerServiceId}
              priceCents={props.service.claude_max_price_cents ?? 0}
              checked={addClaudeMax}
              onCheckedChange={(checked) => {
                setAddClaudeMax(checked)
                if (checked) {
                  setAddClaudeUltra(false)
                  setAddClaudePro(false)
                }
              }}
            />
            <ClaudeSubscriptionAddon
              variant='ultra'
              outOfStock={props.service.claude_ultra_out_of_stock}
              disabled={props.service.out_of_stock && !props.customerServiceId}
              priceCents={props.service.claude_ultra_price_cents ?? 0}
              checked={addClaudeUltra}
              onCheckedChange={(checked) => {
                setAddClaudeUltra(checked)
                if (checked) {
                  setAddClaudeMax(false)
                  setAddClaudePro(false)
                }
              }}
            />
          </div>
        )}
        {unavailable && (
          <p role='alert'>
            {t(
              'The selected service or add-on is out of stock. Please choose again.'
            )}
          </p>
        )}
        {props.service.description && (
          <div className='max-h-40 space-y-3 overflow-y-auto rounded-lg bg-[var(--g-paper-sunk)] p-4 text-sm leading-6 text-[var(--g-ink-soft)]'>
            {props.service.description.split(/\n{2,}/).map((paragraph) => (
              <p key={paragraph} className='whitespace-pre-line'>
                {paragraph}
              </p>
            ))}
          </div>
        )}

        {props.service.route !== 'mac_mini' && (
          <div className='grid gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='guardian-route'>{t('Route')}</FieldLabel>
              <NativeSelect id='guardian-route' {...form.register('route')}>
                {Object.entries(labels.orderRoute)
                  .filter(([value]) => value !== 'mac_mini')
                  .map(([value, label]) => (
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
        )}

        {needsConsent && (
          <label className='flex items-start gap-3 rounded-lg border p-3 text-sm'>
            <Checkbox
              checked={accepted}
              onCheckedChange={(v) => setAccepted(!!v)}
            />
            <span>
              {props.directConfirmationRequired
                ? t(
                    'The original reseller is suspended. I agree to renew directly with the platform; future renewals will also be direct.'
                  )
                : t(
                    'I agree that all refunds deduct the actual payment fee not returned by the channel. The platform handles delivery and support.'
                  )}
            </span>
          </label>
        )}
        {props.service.invoice_enabled && (
          <Field data-invalid={!!errors.invoice_information}>
            <FieldLabel htmlFor='guardian-invoice-information'>
              {t('Invoice information')}
            </FieldLabel>
            <Textarea
              id='guardian-invoice-information'
              aria-invalid={!!errors.invoice_information}
              maxLength={2000}
              rows={4}
              placeholder={t(
                'Enter your invoice information if you need an invoice; otherwise leave blank.'
              )}
              {...form.register('invoice_information')}
            />
            {errors.invoice_information && (
              <FieldError>
                {t('Message must be at most 2000 characters')}
              </FieldError>
            )}
          </Field>
        )}
        <Field data-invalid={!!errors.discovery_channel}>
          <FieldLabel htmlFor='guardian-discovery'>
            {t('How did you hear about us?')}
          </FieldLabel>
          <NativeSelect
            id='guardian-discovery'
            aria-invalid={!!errors.discovery_channel}
            defaultValue=''
            {...form.register('discovery_channel')}
          >
            <NativeSelectOption value='' disabled>
              {t('Please select')}
            </NativeSelectOption>
            <DiscoveryOptions />
          </NativeSelect>
          {errors.discovery_channel && (
            <FieldError>{t('Choose how you heard about us')}</FieldError>
          )}
        </Field>
        <p className='text-muted-foreground text-sm'>
          {maskedEmail
            ? t('Your payment receipt will be emailed to {{email}}.', {
                email: maskedEmail,
              })
            : t(
                'View your receipt in My orders after payment. Bind an account email to receive it by email.'
              )}
        </p>
        <FormProvider {...form}>
          <GuardianContactFields />
        </FormProvider>
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
