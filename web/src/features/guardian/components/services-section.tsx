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
import { useNavigate } from '@tanstack/react-router'
import { ArrowUpRight, Clock } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { getGuardianCatalog } from '../api'
import { GUARDIAN_LINKS, GUARDIAN_QUERY_KEYS } from '../constants'
import { useGuardianLabels } from '../hooks/use-guardian-labels'
import { formatCny } from '../lib/price'
import type { GuardianService } from '../types'
import { CheckoutDialog } from './checkout-dialog'

/**
 * Cards keep a fixed, compact width and sit in a centred row, so the group
 * stays on the page's centre line and a short last row stays centred too.
 */
const SERVICE_CARD_WIDTH = 'w-full max-w-[22rem] sm:w-[20rem]'

function ServiceCard(props: {
  service: GuardianService
  purchaseEnabled: boolean
  onBook: (service: GuardianService) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()

  return (
    <article className='g-rule relative flex h-full flex-col rounded-2xl border bg-[var(--g-paper-raised)] p-6'>
      <div className='flex items-start justify-between gap-4'>
        <p className='g-mono text-xs text-[var(--g-ink-soft)]'>
          {labels.serviceRoute[props.service.route]}
        </p>
        {props.service.badge && (
          <span className='rounded-full bg-[var(--g-accent-wash)] px-2.5 py-0.5 text-xs font-medium text-[var(--g-accent)]'>
            {props.service.badge}
          </span>
        )}
      </div>
      <h3 className='g-heading mt-3 text-2xl'>{props.service.title}</h3>
      {props.service.subtitle && (
        <p className='mt-2 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
          {props.service.subtitle}
        </p>
      )}
      <p className='g-display mt-5 text-4xl'>
        {formatCny(props.service.price_cents, locale)}
      </p>
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
      <div className='mt-auto pt-6'>
        <Button
          className='h-10 w-full'
          disabled={!props.purchaseEnabled}
          onClick={() => props.onBook(props.service)}
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
        render={
          <a
            href={GUARDIAN_LINKS.telegram}
            target='_blank'
            rel='noopener noreferrer'
          />
        }
      >
        {t('Contact us on Telegram')}
        <ArrowUpRight aria-hidden='true' />
      </Button>
    </div>
  )
}

export function ServicesSection(props: {
  requestedServiceId?: number
  onRequestHandled: () => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.auth.user)
  const [selected, setSelected] = useState<GuardianService | null>(null)
  const catalog = useQuery({
    queryKey: GUARDIAN_QUERY_KEYS.catalog,
    queryFn: getGuardianCatalog,
    meta: { errorToast: false },
    retry: 1,
  })
  const services = catalog.data?.services ?? []
  const purchaseEnabled = !!catalog.data?.purchase_enabled
  const { requestedServiceId, onRequestHandled } = props

  // Resume the booking a visitor started before signing in.
  useEffect(() => {
    if (!requestedServiceId || !catalog.isSuccess) return
    const service = catalog.data.services.find(
      (item) => item.id === requestedServiceId
    )
    if (service && user && catalog.data.purchase_enabled) setSelected(service)
    onRequestHandled()
  }, [
    requestedServiceId,
    catalog.isSuccess,
    catalog.data,
    user,
    onRequestHandled,
  ])

  const book = (service: GuardianService) => {
    if (!user) {
      void navigate({
        to: '/sign-in',
        search: { redirect: `/guardian/services?service=${service.id}` },
      })
      return
    }
    setSelected(service)
  }

  return (
    <div id='catalog' className='scroll-mt-24'>
      {catalog.isPending && (
        <div className='flex flex-wrap justify-center gap-5'>
          {[0, 1, 2].map((key) => (
            <Skeleton
              key={key}
              className={cn('h-96 rounded-2xl', SERVICE_CARD_WIDTH)}
            />
          ))}
        </div>
      )}

      {(catalog.isError || (catalog.isSuccess && services.length === 0)) && (
        <ServicesFallback
          title={t('Services are being prepared')}
          body={t(
            'Online booking is not open yet. Reach out on Telegram and we will help you plan your setup.'
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
                'Online ordering is paused right now. Contact us on Telegram to book.'
              )}
            </p>
          )}
          <div className='flex flex-wrap justify-center gap-5'>
            {services.map((service, index) => (
              <AnimateInView
                key={service.id}
                delay={index * 80}
                className={SERVICE_CARD_WIDTH}
              >
                <ServiceCard
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
          service={selected}
          payMethods={catalog.data.pay_methods}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  )
}
