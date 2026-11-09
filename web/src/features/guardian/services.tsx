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
import { useNavigate } from '@tanstack/react-router'
import { gsap } from 'gsap'
import { useCallback, useLayoutEffect, useRef } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { SiApple } from 'react-icons/si'

import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

import { GuardianAccountStatisticsPanel } from './account-statistics'
import { ServicesFaqSection } from './components/faq-section'
import { GuardianShell } from './components/guardian-shell'
import { MacMiniDelivery } from './components/mac-mini-delivery'
import { ServiceProcessSection } from './components/service-process-section'
import { ServicesSection } from './components/services-section'
import type { GuardianBillingPeriod } from './types'

function ServicesIntro(props: { macMini: boolean; accountStatus: boolean }) {
  const { t } = useTranslation()
  const macMiniRef = useRef<HTMLDivElement>(null)

  useLayoutEffect(() => {
    const product = macMiniRef.current
    if (!props.macMini || !product) return

    const media = gsap.matchMedia()
    media.add('(prefers-reduced-motion: no-preference)', () => {
      gsap.from(product.children, {
        autoAlpha: 0,
        y: 28,
        scale: 0.97,
        duration: 0.85,
        stagger: 0.12,
        ease: 'power3.out',
        clearProps: 'transform,opacity,visibility',
      })
    })
    return () => media.revert()
  }, [props.macMini])

  const promises = [
    t('Payment and sign-in stay with you'),
    t('No passwords in the order form'),
    t('Risky steps wait for your approval'),
  ]

  return (
    <div className='flex flex-col items-center text-center'>
      {props.accountStatus && (
        <h1
          id='services-title'
          className='g-display text-[2.25rem] sm:text-[2.75rem] lg:text-[3.25rem]'
        >
          {t('Purchased user account status')}
        </h1>
      )}
      {!props.accountStatus && (
        <h1
          id='services-title'
          className='g-display max-w-5xl text-[2.25rem] sm:text-[2.75rem] lg:text-[3.25rem]'
        >
          <span className='block'>
            {props.macMini ? (
              <Trans
                t={t}
                i18nKey='Claude on an overseas Mac mini'
                components={{
                  brand: (
                    <span className='inline-flex items-center gap-2 align-baseline whitespace-nowrap' />
                  ),
                  logo: (
                    <img
                      src='/images/guardian/claude-color.png'
                      alt=''
                      aria-hidden='true'
                      width={640}
                      height={640}
                      className='size-[0.85em] shrink-0 object-contain'
                    />
                  ),
                }}
              />
            ) : (
              t('Rather have someone do it with you?')
            )}
          </span>
          <span className='g-fade-text block'>
            {props.macMini
              ? t('Remote access. Hosted overseas.')
              : t('We set it up together, on your machine.')}
          </span>
        </h1>
      )}
      {props.macMini && (
        <div
          ref={macMiniRef}
          className='mt-8 flex w-full max-w-2xl flex-col items-center gap-4'
        >
          <p className='flex items-center gap-2 text-sm font-medium tracking-wide text-[var(--g-ink-soft)]'>
            <SiApple
              aria-hidden='true'
              className='size-4 text-[var(--g-ink)]'
            />
            <span>Mac mini M4/M6</span>
          </p>
          <img
            src='/images/guardian/mac-mini-front.jpg'
            alt='Mac mini'
            width={1048}
            height={420}
            className='g-mac-mini-cutout h-auto w-full'
            fetchPriority='high'
          />
        </div>
      )}
      <TabsList
        className='mt-5 max-w-full flex-wrap group-data-horizontal/tabs:h-auto'
        aria-label={t('Service category')}
      >
        <TabsTrigger value='account_status'>
          {t('Purchased user account status')}
        </TabsTrigger>
        <TabsTrigger value='setup'>{t('Environment setup')}</TabsTrigger>
        <TabsTrigger value='mac_mini'>{t('Overseas Mac mini')}</TabsTrigger>
      </TabsList>
      {!props.accountStatus && !props.macMini && (
        <ul className='mt-6 flex flex-wrap justify-center gap-x-6 gap-y-2 text-sm text-[var(--g-ink-soft)]'>
          {promises.map((promise) => (
            <li key={promise} className='flex items-center gap-2'>
              <span
                aria-hidden='true'
                className='w-3 border-t border-[var(--g-accent)]'
              />
              {promise}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function GuardianServices(props: {
  resellerCode?: string
  requestedBillingPeriod?: GuardianBillingPeriod
  requestedClaudeMax?: boolean
  requestedClaudeUltra?: boolean
  requestedServiceId?: number
  category?: 'setup' | 'mac_mini' | 'account_status'
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const clearRequest = useCallback(() => {
    void navigate({
      to: '/guardian/services',
      search: { category: props.category, reseller: props.resellerCode },
      hash: 'catalog',
      replace: true,
    })
  }, [navigate, props.category, props.resellerCode])

  return (
    <GuardianShell resellerCode={props.resellerCode}>
      {/* The intro and the catalog share one first screen, centred on both axes. */}
      <section
        aria-labelledby='services-title'
        className='mx-auto flex max-w-6xl flex-col justify-center gap-10 px-4 py-14 sm:px-6 lg:min-h-[calc(100svh-4rem)] lg:py-10 2xl:max-w-7xl'
      >
        <Tabs
          className='gap-10'
          value={props.category ?? 'setup'}
          onValueChange={(value) => {
            void navigate({
              to: '/guardian/services',
              search: {
                reseller: props.resellerCode,
                category: value as 'setup' | 'mac_mini' | 'account_status',
              },
              replace: true,
            })
          }}
        >
          <ServicesIntro
            macMini={props.category === 'mac_mini'}
            accountStatus={props.category === 'account_status'}
          />
          <TabsContent value='account_status'>
            <GuardianAccountStatisticsPanel />
          </TabsContent>
          {(['setup', 'mac_mini'] as const).map((category) => (
            <TabsContent key={category} value={category}>
              {category === 'mac_mini' && (
                <p className='mb-6 text-center text-sm text-[var(--g-ink-soft)]'>
                  {t(
                    'Choose a hosting plan. Hardware, location and service period are listed in each plan.'
                  )}
                </p>
              )}
              <ServicesSection
                resellerCode={props.resellerCode}
                category={category}
                requestedBillingPeriod={props.requestedBillingPeriod}
                requestedClaudeMax={props.requestedClaudeMax}
                requestedClaudeUltra={props.requestedClaudeUltra}
                requestedServiceId={props.requestedServiceId}
                onRequestHandled={clearRequest}
              />
            </TabsContent>
          ))}
        </Tabs>
      </section>
      {props.category === 'mac_mini' && (
        <MacMiniDelivery resellerSale={!!props.resellerCode} />
      )}
      {(!props.category || props.category === 'setup') && (
        <>
          <ServiceProcessSection />
          <ServicesFaqSection resellerSale={!!props.resellerCode} />
        </>
      )}
    </GuardianShell>
  )
}
