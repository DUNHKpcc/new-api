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

export function RoutesSection() {
  const { t } = useTranslation()
  const routes = [
    {
      key: 'local',
      name: t('Local machine'),
      fit: t(
        'Use Claude directly on your own Mac or Windows. The default recommendation.'
      ),
      points: [
        t('Clash Verge or FlClash with TUN and DNS'),
        t('Windows Firewall, or LuLu on Mac'),
        t('Guided step by step with prompts'),
      ],
      stack: ['Clash Verge', 'FlClash', 'LuLu', 'Windows Firewall'],
    },
    {
      key: 'vm',
      name: t('Virtual machine'),
      fit: t('Build a separate, isolated working environment inside a VM.'),
      points: [
        t('Omarchy guest, or macOS on a Mac host'),
        t('Dedicated LAN network adapter'),
        t('Fail-closed firewall: no proxy, no network'),
      ],
      stack: ['Omarchy', 'macOS', 'FlClash TUN', 'nftables'],
    },
    {
      key: 'vps',
      name: t('Remote server'),
      fit: t(
        'Treat a VPS as your dev machine, with Claude Code running on it.'
      ),
      points: [
        t('Consistent Ubuntu 24.04 environment'),
        t('Remote desktop via Xvfb and noVNC'),
        t('Inbound firewall and line optimization'),
      ],
      stack: ['Ubuntu 24.04', 'Xvfb + noVNC', 'Remote-SSH', 'tmux'],
    },
  ]

  return (
    <GuardianSection id='routes' tone='sunk'>
      <SectionHeading
        id='routes'
        eyebrow={t('Three routes')}
        title={t('Pick where Claude runs. Not an upgrade path.')}
        lede={t(
          'The three routes are parallel choices. Using a VPS as your network exit does not mean you chose the VPS route. No proxy or server yet? Start with an offline pre-check.'
        )}
      />

      <div className='mt-14 grid gap-5 md:grid-cols-3'>
        {routes.map((route, index) => (
          <AnimateInView
            key={route.key}
            delay={index * 90}
            className='g-rule flex flex-col rounded-2xl border bg-[var(--g-paper)] p-6'
          >
            <p className='g-mono text-xs text-[var(--g-ink-soft)]'>
              {String(index + 1).padStart(2, '0')}
            </p>
            <h3 className='g-heading mt-3 text-2xl'>{route.name}</h3>
            <p className='mt-3 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
              {route.fit}
            </p>
            <ul className='g-rule mt-6 space-y-2.5 border-t pt-5 text-[0.9375rem]'>
              {route.points.map((point) => (
                <li key={point} className='flex gap-3'>
                  <span
                    aria-hidden='true'
                    className='mt-[0.7em] w-3 shrink-0 border-t border-[var(--g-accent)]'
                  />
                  {point}
                </li>
              ))}
            </ul>
            <p className='g-mono mt-auto pt-8 text-xs leading-5 text-[var(--g-ink-soft)]'>
              {route.stack.join(' · ')}
            </p>
          </AnimateInView>
        ))}
      </div>
    </GuardianSection>
  )
}
