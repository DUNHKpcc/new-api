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
import { Link } from '@tanstack/react-router'
import { ArrowRight, Gift, Link2, Users, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const AFFILIATE_STEPS = [
  {
    icon: Link2,
    title: 'Share your link and earn rewards',
    description:
      'Share your invitation link and earn registration rewards when available.',
  },
  {
    icon: Users,
    title: 'Invite friends',
    description: 'Rewards are available after referred users register.',
  },
  {
    icon: WalletCards,
    title: 'Transfer Rewards',
    description: 'Move affiliate rewards to your main balance',
  },
] as const

export function AffiliateSection() {
  const { t } = useTranslation()

  return (
    <section
      data-home-section='affiliate'
      className='border-border/40 bg-muted/10 relative z-10 border-y px-6 py-24 md:py-32'
    >
      <div className='mx-auto max-w-6xl'>
        <div className='grid items-center gap-12 lg:grid-cols-[minmax(0,0.78fr)_minmax(0,1.22fr)] lg:gap-16'>
          <AnimateInView className='max-w-xl'>
            <div className='mb-5 flex items-center gap-3'>
              <span className='border-primary/25 bg-primary/8 text-primary flex size-11 items-center justify-center rounded-lg border'>
                <Users className='size-5' aria-hidden='true' />
              </span>
              <div>
                <p className='text-muted-foreground text-xs font-medium tracking-widest uppercase'>
                  {t('Referral Program')}
                </p>
                <p className='text-lg font-bold'>{t('Affiliate Center')}</p>
              </div>
            </div>

            <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-4xl'>
              {t('Share your link and earn rewards')}
            </h2>
            <p className='text-muted-foreground mt-5 text-sm leading-7 md:text-base'>
              {t(
                'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
              )}
            </p>

            <Button
              className='group mt-8 h-11 gap-2 px-5'
              render={<Link to='/affiliate' />}
            >
              {t('Open Affiliate Center')}
              <ArrowRight
                aria-hidden='true'
                className='size-4 transition-transform duration-200 group-hover:translate-x-0.5'
              />
            </Button>
          </AnimateInView>

          <AnimateInView animation='fade-left' delay={120}>
            <div className='border-border/60 border-y'>
              <ol
                aria-label={t('Referral Program')}
                className='grid sm:grid-cols-3'
              >
                {AFFILIATE_STEPS.map((step, index) => {
                  const Icon = step.icon

                  return (
                    <li
                      key={step.title}
                      className={cn(
                        'py-6 sm:px-6',
                        index > 0 &&
                          'border-border/60 border-t sm:border-t-0 sm:border-s'
                      )}
                    >
                      <div className='mb-4 flex items-center gap-3'>
                        <span className='bg-foreground text-background flex size-7 items-center justify-center rounded-full text-xs font-semibold tabular-nums'>
                          {String(index + 1).padStart(2, '0')}
                        </span>
                        <Icon
                          className='text-primary size-4'
                          aria-hidden='true'
                        />
                      </div>
                      <h3 className='text-sm leading-6 font-semibold'>
                        {t(step.title)}
                      </h3>
                      <p className='text-muted-foreground mt-2 text-sm leading-6'>
                        {t(step.description)}
                      </p>
                    </li>
                  )
                })}
              </ol>

              <div className='border-border/60 flex items-start gap-3 border-t py-5 sm:px-6'>
                <Gift
                  className='text-primary mt-0.5 size-5 shrink-0'
                  aria-hidden='true'
                />
                <div>
                  <p className='text-sm font-semibold'>
                    {t('Invitation rewards and commission management')}
                  </p>
                  <p className='text-muted-foreground mt-1 text-sm leading-6'>
                    {t(
                      'Invitation rewards and verified online payment commissions.'
                    )}
                  </p>
                </div>
              </div>
            </div>
          </AnimateInView>
        </div>
      </div>
    </section>
  )
}
