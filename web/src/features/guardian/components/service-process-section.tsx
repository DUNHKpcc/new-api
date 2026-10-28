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
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

import { GuardianSection, SectionHeading } from './section-heading'

export function ServiceProcessSection() {
  const { t } = useTranslation()
  const steps = [
    {
      title: t('Book and pay'),
      body: t(
        'Choose a service, leave a way to reach you, and pay through the site payment gateway.'
      ),
    },
    {
      title: t('We schedule'),
      body: t(
        'We reach out through your contact to agree on a time and confirm the scope.'
      ),
    },
    {
      title: t('Set up together'),
      body: t(
        'A remote session on your machine that follows the authorization tiers. Risky steps wait for your approval.'
      ),
    },
    {
      title: t('Delivered'),
      body: t(
        'You keep a working setup and a record of what changed. Progress and notes stay on your order.'
      ),
    },
  ]

  return (
    <GuardianSection id='process' tone='sunk'>
      <SectionHeading
        id='process'
        eyebrow={t('How a service works')}
        title={t('From booking to a working setup.')}
      />
      <ol className='mt-14 grid gap-x-10 gap-y-10 md:grid-cols-2 lg:grid-cols-4'>
        {steps.map((step, index) => (
          <AnimateInView
            as='li'
            key={step.title}
            delay={index * 80}
            className='g-rule border-t pt-6'
          >
            <span className='g-numeral text-3xl'>
              {String(index + 1).padStart(2, '0')}
            </span>
            <h3 className='g-heading mt-4 text-xl'>{step.title}</h3>
            <p className='mt-3 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
              {step.body}
            </p>
          </AnimateInView>
        ))}
      </ol>
    </GuardianSection>
  )
}
