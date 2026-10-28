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
import { TierChip, type GuardianTier } from './tier-chip'

export function TiersSection() {
  const { t } = useTranslation()
  const tiers: {
    tier: GuardianTier
    name: string
    yourPart: string
    includes: string
  }[] = [
    {
      tier: 'A',
      name: t('Done within scope'),
      yourPart: t('Confirm the scope once, after the route and goal are set.'),
      includes: t('Read-only diagnostics, research, planning and dry runs.'),
    },
    {
      tier: 'B',
      name: t('One approval per plan'),
      yourPart: t(
        'Approve one plan. The agent runs it end to end and undoes it automatically if it fails.'
      ),
      includes: t(
        'Installing clients, adding allow rules, foreground forwarding, temporary BBR, installing TUN.'
      ),
    },
    {
      tier: 'C',
      name: t('Confirmed on the spot'),
      yourPart: t(
        'The agent pauses here and waits for your reply, whenever you see it.'
      ),
      includes: t(
        'Default-deny outbound, enabling firewalls, SSH changes, persistent kernel changes, data cleanup, accounts and money, credentials, organization devices, contact with Claude.'
      ),
    },
  ]

  return (
    <GuardianSection id='tiers'>
      <SectionHeading
        id='tiers'
        eyebrow={t('Tiered authorization')}
        title={t('You do not need to be there for every step.')}
        lede={t(
          'Changes are sorted into three tiers by risk. Approvals only count when you reply explicitly in the conversation, and switching routes asks again.'
        )}
      />

      <div className='g-rule mt-14 border-t'>
        {tiers.map((tier, index) => (
          <AnimateInView
            key={tier.tier}
            delay={index * 90}
            className='g-rule grid gap-4 border-b py-7 md:grid-cols-[minmax(0,14rem)_1fr_1fr] md:gap-10'
          >
            <div className='flex items-center gap-3'>
              <TierChip tier={tier.tier} className='size-8 text-sm' />
              <h3 className='g-heading text-xl'>{tier.name}</h3>
            </div>
            <div>
              <p className='g-mono text-xs text-[var(--g-ink-soft)]'>
                {t('Your part')}
              </p>
              <p className='mt-2 text-[0.9375rem] leading-7'>{tier.yourPart}</p>
            </div>
            <div>
              <p className='g-mono text-xs text-[var(--g-ink-soft)]'>
                {t('Includes')}
              </p>
              <p className='mt-2 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
                {tier.includes}
              </p>
            </div>
          </AnimateInView>
        ))}
      </div>
    </GuardianSection>
  )
}
