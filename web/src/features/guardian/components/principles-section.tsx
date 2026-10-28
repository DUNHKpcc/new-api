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

export function PrinciplesSection() {
  const { t } = useTranslation()
  const principles = [
    {
      title: t('Money and accounts stay with you'),
      body: t(
        'Payment, sign-up, sign-in, KYC and OAuth are always done by you in person.'
      ),
    },
    {
      title: t('No credentials required'),
      body: t(
        'You never hand over your IP, subscription links, passwords, UUIDs or full configs.'
      ),
    },
    {
      title: t('Hands off Claude while preparing'),
      body: t(
        'No launching the client, signing in or touching its services until the environment is ready.'
      ),
    },
    {
      title: t('Adapted to your machine'),
      body: t(
        'The agent checks your system and app versions first, adjusts for differences, and rolls back if something goes wrong.'
      ),
    },
    {
      title: t('No one can promise you will never be banned'),
      body: t(
        'Prices and access rules change quickly. Check the official page before you buy.'
      ),
    },
  ]
  const refusals = [
    t('KYC done on your behalf'),
    t('Ready-made accounts'),
    t('Top-ups paid on your behalf'),
    t('Ready-made email accounts'),
  ]

  return (
    <GuardianSection id='principles' tone='sunk'>
      <SectionHeading
        id='principles'
        eyebrow={t('Red lines')}
        title={t('A few lines we do not cross')}
        lede={t(
          'Every route, script and paid service on this page follows the same rules. They are why the Skill exists.'
        )}
      />

      <ol className='mt-14 grid gap-x-12 gap-y-10 md:grid-cols-2 lg:grid-cols-3'>
        {principles.map((principle, index) => (
          <AnimateInView
            as='li'
            key={principle.title}
            delay={index * 80}
            className='g-rule border-t pt-6'
          >
            <span className='g-numeral text-3xl'>
              {String(index + 1).padStart(2, '0')}
            </span>
            <h3 className='g-heading mt-4 text-xl'>{principle.title}</h3>
            <p className='mt-3 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
              {principle.body}
            </p>
          </AnimateInView>
        ))}
      </ol>

      <div className='mt-16 flex flex-col gap-4 rounded-xl border border-[color-mix(in_srgb,var(--g-tier-c)_35%,transparent)] px-6 py-5 sm:flex-row sm:items-center sm:gap-8'>
        <p className='g-mono shrink-0 text-xs font-bold tracking-[0.08em] text-[var(--g-tier-c)] uppercase'>
          {t('We never offer')}
        </p>
        <ul className='flex flex-wrap gap-x-6 gap-y-2 text-[0.9375rem]'>
          {refusals.map((item) => (
            <li key={item} className='flex items-center gap-2'>
              <span
                aria-hidden='true'
                className='w-3 border-t border-[var(--g-tier-c)]'
              />
              {item}
            </li>
          ))}
        </ul>
      </div>
    </GuardianSection>
  )
}
