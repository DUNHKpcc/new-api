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
import { ArrowRight, BookOpen, Star } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SiGithub } from 'react-icons/si'

import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import { GUARDIAN_LINKS } from '../constants'
import { useGithubStars } from '../hooks/use-github-stars'
import { TierChip, type GuardianTier } from './tier-chip'

const REVEAL_STEP_MS = 420

function ConversationCard() {
  const { t } = useTranslation()
  const plan: { tier: GuardianTier; label: string }[] = [
    { tier: 'A', label: t('Read-only network check') },
    { tier: 'B', label: t('Install the proxy client and TUN, approved once') },
    {
      tier: 'C',
      label: t('Switch default outbound to block, confirm on the spot'),
    },
  ]

  return (
    <figure
      aria-label={t('Example conversation with the Skill')}
      className='g-rule relative rounded-2xl border bg-[var(--g-paper-raised)] shadow-[0_1px_0_var(--g-rule),0_24px_48px_-32px_rgb(40_30_20/0.35)]'
    >
      <div className='g-rule flex items-center justify-between border-b px-5 py-3'>
        <span className='g-mono text-xs text-[var(--g-ink-soft)]'>
          codex · claude-guardian
        </span>
        <span className='g-mono text-xs text-[var(--g-ink-soft)]'>
          {t('Interview')}
        </span>
      </div>

      <div className='space-y-5 px-5 py-6 text-[0.9375rem] leading-7'>
        <div className='g-reveal' style={{ animationDelay: '120ms' }}>
          <p className='g-mono text-xs text-[var(--g-ink-soft)]'>{t('You')}</p>
          <p className='mt-1'>
            <span className='g-mono text-[var(--g-accent)]'>
              $claude-guardian
            </span>{' '}
            {t(
              'First time subscribing to Claude, on a Mac. Compare local and VM first, then decide what to check.'
            )}
          </p>
        </div>

        <div
          className='g-reveal'
          style={{ animationDelay: `${120 + REVEAL_STEP_MS}ms` }}
        >
          <p className='g-mono text-xs text-[var(--g-ink-soft)]'>Agent</p>
          <p className='mt-1'>
            {t(
              'Three questions first: what stage you are at, which system, and what the goal is. I will not open or sign in to Claude while we prepare.'
            )}
          </p>
        </div>

        <ol className='g-rule space-y-2.5 border-l pl-4'>
          {plan.map((step, index) => (
            <li
              key={step.tier}
              className='g-reveal flex items-center gap-3 text-sm'
              style={{
                animationDelay: `${120 + REVEAL_STEP_MS * (index + 2)}ms`,
              }}
            >
              <TierChip tier={step.tier} />
              <span>{step.label}</span>
            </li>
          ))}
        </ol>

        <p
          className='g-reveal flex items-center gap-2 text-sm text-[var(--g-ink-soft)]'
          style={{ animationDelay: `${120 + REVEAL_STEP_MS * 5}ms` }}
        >
          <span className='relative flex size-2'>
            <span className='absolute inline-flex size-full animate-ping rounded-full bg-[var(--g-accent)] opacity-60 motion-reduce:animate-none' />
            <span className='relative inline-flex size-2 rounded-full bg-[var(--g-accent)]' />
          </span>
          {t('Waiting for your confirmation')}
        </p>
      </div>
    </figure>
  )
}

/** Hints that the page continues below a first screen that fills the viewport. */
function ScrollCue() {
  const { t } = useTranslation()
  return (
    <Link
      to='/guardian'
      hash='principles'
      className='g-mono mx-auto mb-8 hidden flex-col items-center gap-3 text-xs text-[var(--g-ink-soft)] transition-colors hover:text-[var(--g-ink)] lg:flex'
    >
      {t('Scroll to explore')}
      <span aria-hidden='true' className='g-scroll-line' />
    </Link>
  )
}

function StarOnGithub() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const stars = useGithubStars()

  return (
    <a
      href={GUARDIAN_LINKS.repo}
      target='_blank'
      rel='noopener noreferrer'
      className='group inline-flex items-center gap-2.5 rounded-md text-[0.9375rem] text-[var(--g-ink-soft)] transition-colors hover:text-[var(--g-ink)]'
    >
      <Star
        aria-hidden='true'
        className='size-4 transition-colors group-hover:fill-[var(--g-accent)] group-hover:text-[var(--g-accent)]'
      />
      {t('Star on GitHub')}
      {stars.data !== undefined && (
        <span className='g-mono rounded-full border border-[var(--g-rule-strong)] px-2 py-px text-xs text-[var(--g-ink)]'>
          {formatNumber(stars.data, locale)}
        </span>
      )}
    </a>
  )
}

function HeroStats() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const stars = useGithubStars()
  const stats = [
    {
      value: stars.data === undefined ? '—' : formatNumber(stars.data, locale),
      label: t('GitHub stars'),
    },
    { value: '100+', label: t('Interview questions') },
    { value: '3', label: t('Runtime routes') },
  ]

  return (
    <dl className='mt-12 flex flex-wrap gap-y-6'>
      {stats.map((stat, index) => (
        <div
          key={stat.label}
          className={cn(
            'flex flex-col pr-8',
            index > 0 && 'g-rule border-l pl-8'
          )}
        >
          <dt className='g-mono order-2 mt-2 text-[0.6875rem] tracking-[0.08em] text-[var(--g-ink-soft)] uppercase'>
            {stat.label}
          </dt>
          <dd className='g-heading -order-1 text-[1.875rem] leading-none'>
            {stat.value}
          </dd>
        </div>
      ))}
    </dl>
  )
}

export function GuardianHero() {
  const { t } = useTranslation()

  return (
    <section className='mx-auto flex max-w-6xl flex-col px-4 sm:px-6 lg:min-h-[calc(100svh-4rem)] 2xl:max-w-7xl'>
      <div className='grid flex-1 items-center gap-14 pt-16 pb-20 lg:grid-cols-[1.3fr_1fr] lg:py-16'>
        <div>
          <p className='g-rule inline-flex items-center gap-2 rounded-full border bg-[var(--g-paper-raised)] px-3 py-1 text-xs tracking-[0.02em] text-[var(--g-ink-soft)]'>
            <span
              aria-hidden='true'
              className='size-1.5 rounded-full bg-[var(--g-accent)]'
            />
            {t('An open-source Skill for agents')}
          </p>
          <h1 className='g-display mt-6 text-[2.5rem] sm:text-[3.5rem] lg:text-[3.75rem] 2xl:text-[4.25rem]'>
            <span className='block'>{t('Set up Claude right.')}</span>
            <span className='g-fade-text block'>{t('You make the call.')}</span>
          </h1>
          <p className='mt-6 max-w-xl text-[1.125rem] leading-[1.625] tracking-[-0.01em] text-pretty'>
            <span className='text-[var(--g-ink-soft)]'>
              {t(
                'Claude-Guardian gives your agent field-tested experience, curated references and scripts for subscription, account, egress IP and runtime setup.'
              )}
            </span>{' '}
            <span className='text-[color-mix(in_srgb,var(--g-ink-soft)_70%,var(--g-paper))]'>
              {t(
                'Free and open source. Risky steps always wait for your approval.'
              )}
            </span>
          </p>
          <div className='mt-9 flex flex-wrap items-center gap-3'>
            <Button
              className='g-pill h-11 gap-2 px-5 text-[0.9375rem]'
              render={
                <a
                  href={GUARDIAN_LINKS.repo}
                  target='_blank'
                  rel='noopener noreferrer'
                />
              }
            >
              <SiGithub aria-hidden='true' />
              {t('Install the Skill for free')}
            </Button>
            <Button
              variant='outline'
              className='g-pill h-11 gap-2 px-5 text-[0.9375rem]'
              render={<Link to='/guardian/services' />}
            >
              {t('Book a setup service')}
              <ArrowRight aria-hidden='true' />
            </Button>
          </div>
          <div className='mt-6 flex flex-wrap items-center gap-x-7 gap-y-3 pl-1'>
            <StarOnGithub />
            <a
              href={GUARDIAN_LINKS.wiki}
              target='_blank'
              rel='noopener noreferrer'
              className='inline-flex items-center gap-2.5 text-[0.9375rem] text-[var(--g-ink-soft)] transition-colors hover:text-[var(--g-ink)]'
            >
              <BookOpen aria-hidden='true' className='size-4' />
              {t('Read the Wiki')}
            </a>
          </div>
          <HeroStats />
        </div>
        <ConversationCard />
      </div>
      <ScrollCue />
    </section>
  )
}
