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
import { motion, useReducedMotion, useSpring } from 'motion/react'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { SiGithub } from 'react-icons/si'

import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import { GUARDIAN_LINKS } from '../constants'
import { useGithubStars } from '../hooks/use-github-stars'
import { TierChip, type GuardianTier } from './tier-chip'

function ClaudeMascot() {
  const anchor = useRef<HTMLSpanElement>(null)
  const reducedMotion = useReducedMotion()
  const x = useSpring(0, { stiffness: 120, damping: 16 })
  const y = useSpring(0, { stiffness: 120, damping: 16 })

  useEffect(() => {
    if (reducedMotion || !window.matchMedia('(pointer: fine)').matches) return
    const follow = (event: PointerEvent) => {
      if (!anchor.current || event.pointerType === 'touch') return
      const rect = anchor.current.getBoundingClientRect()
      x.set(
        Math.max(
          -3,
          Math.min(3, (event.clientX - rect.x - rect.width / 2) / 60)
        )
      )
      y.set(
        Math.max(
          -2,
          Math.min(2, (event.clientY - rect.y - rect.height / 2) / 80)
        )
      )
    }
    const reset = () => {
      x.set(0)
      y.set(0)
    }
    window.addEventListener('pointermove', follow)
    document.documentElement.addEventListener('pointerleave', reset)
    return () => {
      window.removeEventListener('pointermove', follow)
      document.documentElement.removeEventListener('pointerleave', reset)
      reset()
    }
  }, [reducedMotion, x, y])

  return (
    <span
      ref={anchor}
      aria-hidden='true'
      className='pointer-events-none absolute bottom-full left-1/2 z-10 -mb-[9px] h-10 w-16 -translate-x-1/2'
    >
      <svg
        viewBox='0 0 80 52'
        style={{ animationDuration: '0.4s' }}
        className='size-full origin-bottom overflow-visible transition-transform duration-200 group-focus-within/setup:scale-x-105 group-focus-within/setup:scale-y-110 group-focus-within/setup:animate-bounce group-hover/setup:scale-x-105 group-hover/setup:scale-y-110 group-hover/setup:animate-bounce motion-reduce:scale-100 motion-reduce:animate-none motion-reduce:transition-none'
      >
        <path
          fill='var(--g-button)'
          d='M10 2H70V22H80V32H70V52H65V42H60V52H55V42H25V52H20V42H15V52H10V32H0V22H10Z'
        />
        <motion.g
          fill='#251A14'
          style={{ x, y }}
          className='group-focus-within/setup:opacity-0 group-hover/setup:opacity-0'
        >
          <path d='M20 14H25V24H20ZM55 14H60V24H55Z' />
        </motion.g>
        <g className='opacity-0 group-focus-within/setup:opacity-100 group-hover/setup:opacity-100'>
          <path
            fill='#251A14'
            d='M18 21V16H22V12H26V16H30V21H26V17H22V21ZM50 21V16H54V12H58V16H62V21H58V17H54V21ZM33 28H37V32H43V28H47V35H33Z'
          />
          <path fill='#F6AC91' d='M15 25H27V29H15ZM53 25H65V29H53Z' />
        </g>
      </svg>
    </span>
  )
}

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
          <div className='g-hero-actions mt-9 flex flex-wrap items-center gap-3'>
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
            <span className='group/setup relative inline-flex'>
              <ClaudeMascot />
              <Button
                variant='outline'
                className='g-pill h-11 gap-2 px-5 text-[0.9375rem]'
                render={<Link to='/guardian/services' />}
              >
                {t('Book a setup service')}
                <ArrowRight aria-hidden='true' />
              </Button>
            </span>
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
