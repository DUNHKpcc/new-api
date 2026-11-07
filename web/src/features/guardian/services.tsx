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
import { Link, useNavigate } from '@tanstack/react-router'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { ServicesFaqSection } from './components/faq-section'
import { GuardianShell } from './components/guardian-shell'
import { ServiceProcessSection } from './components/service-process-section'
import { ServicesSection } from './components/services-section'

function ServicesIntro() {
  const { t } = useTranslation()
  const promises = [
    t('Payment and sign-in stay with you'),
    t('No passwords in the order form'),
    t('Risky steps wait for your approval'),
  ]

  return (
    <div className='flex flex-col items-center text-center'>
      <h1
        id='services-title'
        className='g-display max-w-5xl text-[2.25rem] sm:text-[2.75rem] lg:text-[3.25rem]'
      >
        <span className='block'>
          {t('Rather have someone do it with you?')}
        </span>
        <span className='g-fade-text block'>
          {t('We set it up together, on your machine.')}
        </span>
      </h1>
      <Button
        variant='outline'
        className='mt-5 border-[var(--g-accent)] bg-[var(--g-accent-wash)] font-semibold text-[var(--g-accent)] hover:bg-[var(--g-accent-wash)] hover:text-[var(--g-accent)]'
        render={<Link to='/guardian/account-statistics' />}
      >
        {t('Click to view purchased user account status')}
      </Button>
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
    </div>
  )
}

export function GuardianServices(props: { requestedServiceId?: number }) {
  const navigate = useNavigate()
  const clearRequest = useCallback(() => {
    void navigate({
      to: '/guardian/services',
      search: {},
      hash: 'catalog',
      replace: true,
    })
  }, [navigate])

  return (
    <GuardianShell>
      {/* The intro and the catalog share one first screen, centred on both axes. */}
      <section
        aria-labelledby='services-title'
        className='mx-auto flex max-w-6xl flex-col justify-center gap-10 px-4 py-14 sm:px-6 lg:min-h-[calc(100svh-4rem)] lg:py-10 2xl:max-w-7xl'
      >
        <ServicesIntro />
        <ServicesSection
          requestedServiceId={props.requestedServiceId}
          onRequestHandled={clearRequest}
        />
      </section>
      <ServiceProcessSection />
      <ServicesFaqSection />
    </GuardianShell>
  )
}
