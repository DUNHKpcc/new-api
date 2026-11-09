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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { getGuardianCatalog } from '../api'
import { useGuardianLabels } from '../hooks/use-guardian-labels'
import {
  resellerRead,
  type CustomerService,
  type RenewalQuote,
} from '../reseller-api'
import type { GuardianBillingPeriod } from '../types'
import { CheckoutDialog } from './checkout-dialog'

export function GuardianCustomerServices() {
  const { t } = useTranslation()
  const labels = useGuardianLabels()
  const [selected, setSelected] = useState<{
    id: number
    quote: RenewalQuote
  } | null>(null)
  const [period, setPeriod] = useState<GuardianBillingPeriod>('monthly')
  const items = useQuery({
    queryKey: ['guardian', 'customer-services'],
    queryFn: () => resellerRead<CustomerService[]>('/customer-services'),
  })
  const catalog = useQuery({
    queryKey: ['guardian', 'renewal-payment'],
    queryFn: getGuardianCatalog,
  })
  const quote = useMutation({
    mutationFn: async (id: number) => ({
      id,
      quote: await resellerRead<RenewalQuote>(`/customer-services/${id}/quote`),
    }),
    onSuccess: (value) => {
      setPeriod('monthly')
      setSelected(value)
    },
  })
  if (items.isError) return <ErrorState onRetry={() => void items.refetch()} />
  if (!items.data?.length) return null
  return (
    <section className='my-6 space-y-4'>
      <h2 className='text-xl font-semibold'>{t('My services and renewals')}</h2>
      {items.data.map((item) => (
        <div
          key={item.id}
          className='flex flex-wrap items-center justify-between gap-3 rounded-lg border p-4'
        >
          <div>
            <p>{item.title}</p>
            <p className='text-muted-foreground text-sm'>
              {t('Customer service ID')}: {item.id} ·{' '}
              {item.reseller_id ? t('Reseller renewal') : t('Direct renewal')}
            </p>
          </div>
          <Button
            disabled={quote.isPending || !catalog.data?.purchase_enabled}
            onClick={() => quote.mutate(item.id)}
          >
            {t('Renew')}
          </Button>
        </div>
      ))}
      {selected && catalog.data && (
        <div className='space-y-4 rounded-lg border p-4'>
          <p>{t('Choose renewal period')}</p>
          <NativeSelect
            aria-label={t('Billing period')}
            value={period}
            onChange={(e) => setPeriod(e.target.value as GuardianBillingPeriod)}
          >
            <NativeSelectOption value='monthly'>
              {labels.billingPeriod.monthly}
            </NativeSelectOption>
            {!!selected.quote.service.quarter_price_cents && (
              <NativeSelectOption value='quarterly'>
                {labels.billingPeriod.quarterly}
              </NativeSelectOption>
            )}
            {!!selected.quote.service.annual_price_cents && (
              <NativeSelectOption value='yearly'>
                {labels.billingPeriod.yearly}
              </NativeSelectOption>
            )}
          </NativeSelect>
          <RenewalCheckout
            selected={selected}
            period={period}
            onClose={() => setSelected(null)}
            payMethods={catalog.data.pay_methods.filter(
              (m) => !selected.quote.reseller_code || m.type === 'alipay'
            )}
          />
        </div>
      )}
    </section>
  )
}

function RenewalCheckout(props: {
  selected: { id: number; quote: RenewalQuote }
  period: GuardianBillingPeriod
  onClose: () => void
  payMethods: import('../types').GuardianPayMethod[]
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button onClick={() => setOpen(true)}>{t('Continue to checkout')}</Button>
      {open && (
        <CheckoutDialog
          customerServiceId={props.selected.id}
          resellerCode={props.selected.quote.reseller_code}
          directConfirmationRequired={
            props.selected.quote.direct_confirmation_required
          }
          service={props.selected.quote.service}
          billingPeriod={props.period}
          payMethods={props.payMethods}
          onClose={props.onClose}
        />
      )}
    </>
  )
}
