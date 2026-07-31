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
import { ArrowUpRight, Check, ChevronDown, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

import type { TopNavLink } from '../types'
import {
  isPublicNavLinkActive,
  publicHeaderLayoutClasses,
} from './public-header-layout'

export type PublicNavLinkClickHandler = (
  event: React.MouseEvent<HTMLAnchorElement>,
  link: TopNavLink,
  closeMobile?: boolean
) => void

type PublicHeaderNavLinksProps = {
  links: readonly TopNavLink[]
  pathname: string
  onLinkClick: PublicNavLinkClickHandler
}

export function PublicHeaderDesktopLinkMeasurement(props: {
  links: readonly TopNavLink[]
}) {
  const { t } = useTranslation()

  return props.links.map((link) => (
    <span
      key={`${link.href}:${link.title}`}
      className={publicHeaderLayoutClasses.desktopLink}
    >
      {t(link.title)}
    </span>
  ))
}

export function PublicHeaderDesktopLinks(props: PublicHeaderNavLinksProps) {
  const { t } = useTranslation()

  return (
    <>
      {props.links.map((link) => {
        const isActive = isPublicNavLinkActive(props.pathname, link)
        const className = cn(
          publicHeaderLayoutClasses.desktopLink,
          isActive
            ? publicHeaderLayoutClasses.desktopLinkActive
            : 'text-muted-foreground',
          link.disabled && 'pointer-events-none opacity-50'
        )

        if (link.external) {
          return (
            <a
              key={`${link.href}:${link.title}`}
              href={link.href}
              target='_blank'
              rel='noopener noreferrer'
              aria-disabled={link.disabled}
              tabIndex={link.disabled ? -1 : undefined}
              onClick={(event) => props.onLinkClick(event, link)}
              className={className}
            >
              {t(link.title)}
            </a>
          )
        }

        return (
          <Link
            key={`${link.href}:${link.title}`}
            to={link.href}
            disabled={link.disabled}
            aria-current={isActive ? 'page' : undefined}
            onClick={(event) => props.onLinkClick(event, link)}
            className={className}
          >
            {t(link.title)}
          </Link>
        )
      })}
    </>
  )
}

export function PublicHeaderOverflowMenu(props: PublicHeaderNavLinksProps) {
  const { t } = useTranslation()
  const hasActiveLink = props.links.some((link) =>
    isPublicNavLinkActive(props.pathname, link)
  )

  if (props.links.length === 0) return null

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        render={
          <Button
            type='button'
            variant='ghost'
            className={cn(
              publicHeaderLayoutClasses.desktopLink,
              'gap-1 rounded-none bg-transparent hover:bg-transparent',
              hasActiveLink
                ? publicHeaderLayoutClasses.desktopLinkActive
                : 'text-muted-foreground'
            )}
            aria-label={t('More')}
          />
        }
      >
        <span>{t('More')}</span>
        <ChevronDown className='size-3.5' aria-hidden='true' />
      </DropdownMenuTrigger>
      <DropdownMenuContent align='center' sideOffset={8} className='min-w-44'>
        {props.links.map((link) => {
          const isActive = isPublicNavLinkActive(props.pathname, link)
          let trailingIcon: React.ReactNode = null
          if (link.external) {
            trailingIcon = (
              <ArrowUpRight
                className='text-muted-foreground size-4'
                aria-hidden='true'
              />
            )
          } else if (isActive) {
            trailingIcon = (
              <Check className='text-primary size-4' aria-hidden='true' />
            )
          }

          const content = (
            <>
              <span className='min-w-0 flex-1 truncate'>{t(link.title)}</span>
              {trailingIcon}
            </>
          )

          return (
            <DropdownMenuItem
              key={`${link.href}:${link.title}`}
              disabled={link.disabled}
              className={cn(
                'min-h-9 gap-2 px-2.5 [font-family:var(--font-sans)]',
                isActive && 'bg-primary/5 text-foreground'
              )}
              render={
                link.external ? (
                  <a
                    href={link.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    aria-disabled={link.disabled}
                    tabIndex={link.disabled ? -1 : undefined}
                    onClick={(event) => props.onLinkClick(event, link)}
                  />
                ) : (
                  <Link
                    to={link.href}
                    disabled={link.disabled}
                    aria-current={isActive ? 'page' : undefined}
                    onClick={(event) => props.onLinkClick(event, link)}
                  />
                )
              }
            >
              {content}
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function PublicHeaderMobileLinks(props: PublicHeaderNavLinksProps) {
  const { t } = useTranslation()

  return (
    <>
      {props.links.map((link, index) => {
        const isActive = isPublicNavLinkActive(props.pathname, link)
        const className = cn(
          publicHeaderLayoutClasses.mobileLink,
          isActive
            ? publicHeaderLayoutClasses.mobileLinkActive
            : publicHeaderLayoutClasses.mobileLinkIdle,
          link.disabled && 'pointer-events-none opacity-50'
        )
        const content = (
          <>
            <span className={publicHeaderLayoutClasses.mobileLinkIndex}>
              {String(index + 1).padStart(2, '0')}
            </span>
            <span className='min-w-0 truncate'>{t(link.title)}</span>
            {link.external ? (
              <ArrowUpRight
                className='text-muted-foreground size-4'
                aria-hidden='true'
              />
            ) : (
              <ChevronRight
                className='text-muted-foreground size-4 transition-transform duration-150 group-hover:translate-x-0.5'
                aria-hidden='true'
              />
            )}
          </>
        )

        if (link.external) {
          return (
            <a
              key={`${link.href}:${link.title}`}
              href={link.href}
              target='_blank'
              rel='noopener noreferrer'
              aria-disabled={link.disabled}
              tabIndex={link.disabled ? -1 : undefined}
              onClick={(event) => props.onLinkClick(event, link, true)}
              className={className}
            >
              {content}
            </a>
          )
        }

        return (
          <Link
            key={`${link.href}:${link.title}`}
            to={link.href}
            disabled={link.disabled}
            aria-current={isActive ? 'page' : undefined}
            onClick={(event) => props.onLinkClick(event, link, true)}
            className={className}
          >
            {content}
          </Link>
        )
      })}
    </>
  )
}
