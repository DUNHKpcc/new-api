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
import { ArrowUpRight, Menu } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { ThemeSwitch } from '@/components/theme-switch'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useAuthStore } from '@/stores/auth-store'

import '@/styles/guardian.css'

import { GUARDIAN_LINKS } from '../constants'
import { PixelField } from './pixel-field'

export function GuardianMark(props: { className?: string }) {
  return (
    <svg
      viewBox='0 0 24 24'
      fill='none'
      aria-hidden='true'
      className={props.className}
    >
      <path
        d='M4.5 20.5V11a7.5 7.5 0 0 1 15 0v9.5'
        stroke='currentColor'
        strokeWidth='1.8'
        strokeLinecap='round'
      />
      <path
        d='M8.5 20.5V11.5a3.5 3.5 0 0 1 7 0v9'
        stroke='currentColor'
        strokeWidth='1.8'
        strokeLinecap='round'
        opacity='0.45'
      />
      <circle cx='12' cy='11.5' r='1.6' fill='var(--g-accent)' />
    </svg>
  )
}

function useSectionLinks() {
  const { t } = useTranslation()
  return [
    { to: '/guardian', hash: 'principles', label: t('Principles') },
    { to: '/guardian', hash: 'how-it-works', label: t('How it works') },
    { to: '/guardian', hash: 'routes', label: t('Routes') },
    { to: '/guardian/services', label: t('Services') },
    { to: '/guardian', hash: 'faq', label: t('FAQ') },
  ] as const
}

function GuardianHeader() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const sections = useSectionLinks()

  return (
    <header className='g-rule sticky top-0 z-40 border-b bg-[color-mix(in_srgb,var(--g-paper)_88%,transparent)] backdrop-blur-md'>
      <div className='mx-auto flex h-16 max-w-6xl items-center gap-6 px-4 sm:px-6 2xl:max-w-7xl'>
        <Link
          to='/guardian'
          className='flex items-center gap-2.5 rounded-md outline-none focus-visible:ring-3 focus-visible:ring-[var(--g-accent)]/40'
        >
          <GuardianMark className='size-6' />
          <span className='g-heading text-[1.0625rem]'>Claude-Guardian</span>
        </Link>

        <nav
          aria-label={t('Page sections')}
          className='hidden items-center gap-1 lg:flex'
        >
          {sections.map((section) => (
            <Link
              key={section.label}
              to={section.to}
              hash={'hash' in section ? section.hash : undefined}
              activeOptions={{ exact: true, includeHash: true }}
              className='rounded-md px-3 py-1.5 text-sm text-[var(--g-ink-soft)] transition-colors hover:text-[var(--g-ink)] data-[status=active]:text-[var(--g-ink)]'
            >
              {section.label}
            </Link>
          ))}
        </nav>

        <div className='ml-auto flex items-center gap-2'>
          <Link
            to='/'
            className='hidden rounded-md px-3 py-1.5 text-sm text-[var(--g-ink-soft)] transition-colors hover:text-[var(--g-ink)] sm:inline-flex'
          >
            {t('Back to main site')}
          </Link>
          <ThemeSwitch />
          {user ? (
            <Button
              variant='outline'
              className='h-9 px-3.5'
              render={<Link to='/guardian/orders' />}
            >
              {t('My orders')}
            </Button>
          ) : (
            <Button
              variant='outline'
              className='h-9 px-3.5'
              render={
                <Link to='/sign-in' search={{ redirect: '/guardian/orders' }} />
              }
            >
              {t('Sign in')}
            </Button>
          )}
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant='ghost'
                  size='icon'
                  className='size-9 lg:hidden'
                  aria-label={t('Open section menu')}
                />
              }
            >
              <Menu />
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end' className='guardian-root w-48'>
              {sections.map((section) => (
                <DropdownMenuItem
                  key={section.label}
                  render={
                    <Link
                      to={section.to}
                      hash={'hash' in section ? section.hash : undefined}
                    />
                  }
                >
                  {section.label}
                </DropdownMenuItem>
              ))}
              <DropdownMenuItem render={<Link to='/' />}>
                {t('Back to main site')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  )
}

function GuardianFooter() {
  const { t } = useTranslation()
  const { systemName } = useSystemConfig()
  const links = [
    { href: GUARDIAN_LINKS.repo, label: 'GitHub' },
    { href: GUARDIAN_LINKS.wiki, label: t('Wiki') },
    { href: GUARDIAN_LINKS.issues, label: t('Report an issue') },
    { href: GUARDIAN_LINKS.telegram, label: 'Telegram' },
  ]

  return (
    <footer className='g-rule border-t'>
      <div className='mx-auto grid max-w-6xl gap-10 px-4 py-14 sm:px-6 md:grid-cols-[1.4fr_1fr] 2xl:max-w-7xl'>
        <div className='space-y-4'>
          <div className='flex items-center gap-2.5'>
            <GuardianMark className='size-6' />
            <span className='g-heading text-lg'>Claude-Guardian</span>
          </div>
          <p className='max-w-md text-sm leading-6 text-[var(--g-ink-soft)]'>
            {t(
              'Claude-Guardian is an independent open-source project licensed under GPL-3.0-only. It is not affiliated with, endorsed by or sponsored by Anthropic. Claude is a trademark of Anthropic, PBC.'
            )}
          </p>
        </div>
        <div className='flex flex-col gap-6 md:items-end'>
          <ul className='flex flex-wrap gap-x-6 gap-y-2 text-sm'>
            {links.map((link) => (
              <li key={link.href}>
                <a
                  href={link.href}
                  target='_blank'
                  rel='noopener noreferrer'
                  className='g-link inline-flex items-center gap-1'
                >
                  {link.label}
                  <ArrowUpRight className='size-3.5' aria-hidden='true' />
                </a>
              </li>
            ))}
          </ul>
          <Link
            to='/'
            className='text-sm text-[var(--g-ink-soft)] transition-colors hover:text-[var(--g-ink)]'
          >
            {t('Services provided through {{name}}', {
              name: systemName || t('this site'),
            })}
          </Link>
        </div>
      </div>
    </footer>
  )
}

export function GuardianShell(props: { children: ReactNode }) {
  return (
    <div className='guardian-root relative isolate flex min-h-svh flex-col'>
      <PixelField />
      <GuardianHeader />
      <main id='main-content' className='relative z-10 flex-1'>
        {props.children}
      </main>
      <div className='relative z-10'>
        <GuardianFooter />
      </div>
    </div>
  )
}
