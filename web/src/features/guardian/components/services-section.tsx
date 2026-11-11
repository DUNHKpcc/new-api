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
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowUpRight, Clock } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Skeleton } from '@/components/ui/skeleton'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { getGuardianCatalog } from '../api'
import { GUARDIAN_QUERY_KEYS } from '../constants'
import { useGuardianLabels } from '../hooks/use-guardian-labels'
import { formatCny, guardianPeriodPrice } from '../lib/price'
import { getResellerCatalog } from '../reseller-api'
import type { GuardianService, GuardianBillingPeriod } from '../types'
import { CheckoutDialog } from './checkout-dialog'
import { ClaudeSubscriptionAddon } from './claude-subscription-addon'
import { MacChipIcon } from './mac-chip-icon'

/**
 * Cards keep a fixed, compact width and sit in a centred row, so the group
 * stays on the page's centre line and a short last row stays centred too.
 */
const SERVICE_CARD_WIDTH = 'w-full max-w-[22rem] sm:w-[20rem]'
const MAC_CARD_WIDTH = 'w-full max-w-[30rem] sm:w-[30rem]'

function ServiceCard(props: {
  resellerSale?: boolean
  service: GuardianService
  purchaseEnabled: boolean
  onBook: (
    service: GuardianService,
    period: GuardianBillingPeriod,
    addClaudeMax: boolean,
    addClaudeUltra: boolean
  ) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const [addClaudeMax, setAddClaudeMax] = useState(false)
  const [addClaudeUltra, setAddClaudeUltra] = useState(false)
  const [period, setPeriod] = useState<GuardianBillingPeriod>('monthly')
  const pricing = guardianPeriodPrice(props.service, period)
  const macMini = props.service.route === 'mac_mini'

  return (
    <article className='g-rule relative flex h-full flex-col rounded-2xl border bg-[var(--g-paper-raised)] p-6'>
      <div className='flex items-start justify-between gap-4'>
        <p className='g-mono shrink-0 text-xs text-[var(--g-ink-soft)]'>
          {labels.serviceRoute[props.service.route]}
        </p>
        <div className='flex flex-wrap justify-end gap-1.5'>
          {props.service.invoice_enabled && (
            <Badge
              variant='outline'
              className='border-[var(--g-rule)] text-[var(--g-ink-soft)]'
            >
              {t('Supports invoicing')}
            </Badge>
          )}
          {props.service.badge && (
            <Badge className='bg-[var(--g-accent-wash)] text-[var(--g-accent)]'>
              {props.service.badge}
            </Badge>
          )}
        </div>
      </div>
      <div className='mt-3 flex items-start justify-between gap-3'>
        <div className='min-w-0 flex-1'>
          <h3 className='g-heading text-2xl'>{props.service.title}</h3>
          {props.service.subtitle && (
            <p className='mt-2 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
              {props.service.subtitle}
            </p>
          )}
        </div>
        {props.service.route === 'mac_mini' && props.service.chip && (
          <MacChipIcon chip={props.service.chip} className='size-16' />
        )}
      </div>
      {props.service.route === 'mac_mini' &&
        (props.service.region || props.service.hardware_config) && (
          <dl className='mt-4 space-y-2 text-sm text-[var(--g-ink-soft)]'>
            {props.service.region && (
              <div className='flex flex-wrap justify-between gap-x-3 gap-y-1'>
                <dt>{t('Hosting region')}</dt>
                <dd className='text-[var(--g-ink)]'>
                  {labels.macRegion[props.service.region]}
                </dd>
              </div>
            )}
            {props.service.hardware_config && (
              <div className='flex flex-wrap justify-between gap-x-3 gap-y-1'>
                <dt>{t('Hardware configuration')}</dt>
                <dd className='text-[var(--g-ink)]'>
                  {labels.macConfig[props.service.hardware_config]}
                </dd>
              </div>
            )}
          </dl>
        )}
      <div className='mt-5 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1'>
        <p className='g-display text-4xl'>{formatCny(pricing.total, locale)}</p>
        {props.service.sold_count != null && (
          <p className='text-sm text-[var(--g-ink-soft)]'>
            <Trans
              t={t}
              i18nKey='Sold: {{total}}'
              defaults='Sold: <highlight>{{total}}</highlight>'
              values={{ total: formatNumber(props.service.sold_count, locale) }}
              components={{
                highlight: (
                  <span className='font-semibold text-[var(--g-accent)]' />
                ),
              }}
            />
          </p>
        )}
      </div>
      {macMini && (
        <RadioGroup
          aria-label={t('Billing period')}
          value={period}
          onValueChange={(value) => setPeriod(value as GuardianBillingPeriod)}
          className='mt-4 grid grid-cols-3 gap-2'
        >
          {(['monthly', 'quarterly', 'yearly'] as const).map((option) => {
            const plan = guardianPeriodPrice(props.service, option)
            return (
              <label
                key={option}
                className={cn(
                  'g-rule relative flex cursor-pointer flex-col items-center gap-1 rounded-lg border px-1 py-2 text-xs has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-[var(--g-accent)] has-[:focus-visible]:ring-offset-2',
                  period === option &&
                    'border-[var(--g-accent)] bg-[var(--g-accent-wash)]',
                  !plan.total && 'cursor-not-allowed opacity-40'
                )}
              >
                <RadioGroupItem
                  className='sr-only absolute'
                  value={option}
                  disabled={!plan.total}
                />
                <span>{labels.billingPeriod[option]}</span>
                <span className='font-medium'>
                  {plan.total ? formatCny(plan.total, locale) : '—'}
                </span>
                {plan.total > 0 && plan.savings > 0 && (
                  <span className='text-[var(--g-accent)]'>
                    {t('Save {{percent}}%', {
                      percent: formatNumber(plan.discount, locale),
                    })}
                  </span>
                )}
              </label>
            )
          })}
        </RadioGroup>
      )}

      {macMini && (
        <div className='mt-2 space-y-1 text-sm'>
          <p className='text-[var(--g-ink-soft)]'>
            {labels.billingPeriod[period]} ·{' '}
            {t('Equivalent to {{price}}/month', {
              price: formatCny(pricing.monthly, locale),
            })}
          </p>
          {period !== 'monthly' && pricing.savings > 0 && (
            <p className='font-medium text-[var(--g-accent)]'>
              {t('Save {{amount}} ({{percent}}%) vs monthly payments', {
                amount: formatCny(pricing.savings, locale),
                percent: formatNumber(pricing.discount, locale),
              })}
            </p>
          )}
        </div>
      )}
      {props.service.delivery_note && (
        <p className='mt-2 flex items-center gap-1.5 text-sm text-[var(--g-ink-soft)]'>
          <Clock className='size-3.5' aria-hidden='true' />
          {props.service.delivery_note}
        </p>
      )}
      {props.service.features.length > 0 && (
        <ul className='g-rule mt-5 space-y-2 border-t pt-4 text-[0.9375rem]'>
          {props.service.features.map((feature) => (
            <li key={feature} className='flex gap-3'>
              <span
                aria-hidden='true'
                className='mt-[0.7em] w-3 shrink-0 border-t border-[var(--g-accent)]'
              />
              {feature}
            </li>
          ))}
        </ul>
      )}
      <div className='mt-auto space-y-4 pt-6'>
        {macMini && (
          <div className='grid grid-cols-2 gap-3'>
            <ClaudeSubscriptionAddon
              variant='max'
              priceCents={props.service.claude_max_price_cents ?? 0}
              checked={addClaudeMax}
              onCheckedChange={(checked) => {
                setAddClaudeMax(checked)
                if (checked) setAddClaudeUltra(false)
              }}
            />
            <ClaudeSubscriptionAddon
              variant='ultra'
              priceCents={props.service.claude_ultra_price_cents ?? 0}
              checked={addClaudeUltra}
              onCheckedChange={(checked) => {
                setAddClaudeUltra(checked)
                if (checked) setAddClaudeMax(false)
              }}
            />
          </div>
        )}
        {macMini && (
          <p className='text-sm font-semibold'>
            {t('Total: {{price}}', {
              price: formatCny(
                pricing.total +
                  (addClaudeUltra
                    ? (props.service.claude_ultra_price_cents ?? 0)
                    : 0) +
                  (addClaudeMax
                    ? (props.service.claude_max_price_cents ?? 0)
                    : 0),
                locale
              ),
            })}
          </p>
        )}
        <Button
          className='h-10 w-full'
          disabled={!props.purchaseEnabled}
          onClick={() =>
            props.onBook(props.service, period, addClaudeMax, addClaudeUltra)
          }
        >
          {props.purchaseEnabled
            ? t('Book this service')
            : t('Ordering unavailable')}
        </Button>
      </div>
    </article>
  )
}

function ServicesFallback(props: { title: string; body: string }) {
  const { t } = useTranslation()
  return (
    <div className='g-rule rounded-2xl border border-dashed px-6 py-12 text-center'>
      <p className='g-heading text-xl'>{props.title}</p>
      <p className='mx-auto mt-3 max-w-md text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
        {props.body}
      </p>
      <Button
        variant='outline'
        className='mt-6 gap-1.5'
        render={<Link to='/support' />}
      >
        {t('Customer Service')}
        <ArrowUpRight aria-hidden='true' />
      </Button>
    </div>
  )
}

export function ServicesSection(props: {
  resellerCode?: string
  category?: 'setup' | 'mac_mini'
  requestedServiceId?: number
  requestedBillingPeriod?: GuardianBillingPeriod
  requestedClaudeMax?: boolean
  requestedClaudeUltra?: boolean
  onRequestHandled: () => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.auth.user)
  const [addClaudeMax, setAddClaudeMax] = useState(false)
  const [addClaudeUltra, setAddClaudeUltra] = useState(false)
  const [billingPeriod, setBillingPeriod] =
    useState<GuardianBillingPeriod>('monthly')
  const [selected, setSelected] = useState<GuardianService | null>(null)
  const catalog = useQuery({
    queryKey: [...GUARDIAN_QUERY_KEYS.catalog, props.resellerCode ?? 'direct'],
    queryFn: () =>
      props.resellerCode
        ? getResellerCatalog(props.resellerCode)
        : getGuardianCatalog(),
    meta: { errorToast: false },
    retry: 1,
  })
  const services = (catalog.data?.services ?? []).filter(
    (service) =>
      (service.route === 'mac_mini') === (props.category === 'mac_mini')
  )
  const purchaseEnabled = !!catalog.data?.purchase_enabled
  const {
    requestedServiceId,
    requestedBillingPeriod,
    requestedClaudeMax,
    requestedClaudeUltra,
    onRequestHandled,
  } = props

  // Resume the booking a visitor started before signing in.
  useEffect(() => {
    if (!requestedServiceId || !catalog.isSuccess || !user) return
    const service = catalog.data.services.find(
      (item) => item.id === requestedServiceId
    )
    if (service && user && catalog.data.purchase_enabled) {
      const requested = requestedBillingPeriod ?? 'monthly'
      setBillingPeriod(
        guardianPeriodPrice(service, requested).total > 0
          ? requested
          : 'monthly'
      )
      setAddClaudeMax(
        !!requestedClaudeMax &&
          !requestedClaudeUltra &&
          service.route === 'mac_mini' &&
          (service.claude_max_price_cents ?? 0) > 0
      )
      setAddClaudeUltra(
        !!requestedClaudeUltra &&
          service.route === 'mac_mini' &&
          (service.claude_ultra_price_cents ?? 0) > 0
      )
      setSelected(service)
    }
    onRequestHandled()
  }, [
    requestedServiceId,
    requestedBillingPeriod,
    requestedClaudeMax,
    requestedClaudeUltra,
    catalog.isSuccess,
    catalog.data,
    user,
    onRequestHandled,
  ])

  const book = (
    service: GuardianService,
    period: GuardianBillingPeriod,
    addon: boolean,
    ultra: boolean
  ) => {
    if (!user) {
      void navigate({
        to: '/sign-in',
        search: {
          redirect: `/guardian/services?service=${service.id}${props.resellerCode ? `&reseller=${encodeURIComponent(props.resellerCode)}` : ''}${service.route === 'mac_mini' ? `&category=mac_mini&billing_period=${period}${addon ? '&claude_max=yes' : ''}${ultra ? '&claude_ultra=yes' : ''}` : ''}`,
        },
      })
      return
    }
    setBillingPeriod(period)
    setAddClaudeMax(addon)
    setAddClaudeUltra(ultra)
    setSelected(service)
  }

  return (
    <div id='catalog' className='scroll-mt-24'>
      {catalog.isPending && (
        <div className='flex flex-wrap justify-center gap-5'>
          {[0, 1, 2].map((key) => (
            <Skeleton
              key={key}
              className={cn(
                'h-96 rounded-2xl',
                props.category === 'mac_mini'
                  ? MAC_CARD_WIDTH
                  : SERVICE_CARD_WIDTH
              )}
            />
          ))}
        </div>
      )}

      {(catalog.isError || (catalog.isSuccess && services.length === 0)) && (
        <ServicesFallback
          title={t('Services are being prepared')}
          body={t(
            'Online booking is not open yet. Contact customer service and we will help you plan your setup.'
          )}
        />
      )}

      {catalog.isSuccess && services.length > 0 && (
        <>
          {!purchaseEnabled && (
            <p
              role='status'
              className='mb-6 text-center text-sm text-[var(--g-ink-soft)]'
            >
              {t(
                'Online ordering is paused right now. Contact customer service to book.'
              )}
            </p>
          )}
          <div className='flex flex-wrap justify-center gap-5'>
            {services.map((service, index) => (
              <AnimateInView
                key={service.id}
                delay={index * 80}
                className={
                  service.route === 'mac_mini'
                    ? MAC_CARD_WIDTH
                    : SERVICE_CARD_WIDTH
                }
              >
                <ServiceCard
                  resellerSale={!!props.resellerCode}
                  service={service}
                  purchaseEnabled={purchaseEnabled}
                  onBook={book}
                />
              </AnimateInView>
            ))}
          </div>
          <p className='mt-6 text-center text-sm text-[var(--g-ink-soft)]'>
            {t(
              'Prices in CNY, paid through the site payment gateway. After payment we reach out through the contact you leave to schedule a time.'
            )}
          </p>
        </>
      )}

      {selected && catalog.data && (
        <CheckoutDialog
          resellerCode={props.resellerCode}
          service={selected}
          billingPeriod={billingPeriod}
          addClaudeMax={addClaudeMax}
          addClaudeUltra={addClaudeUltra}
          payMethods={catalog.data.pay_methods}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  )
}
