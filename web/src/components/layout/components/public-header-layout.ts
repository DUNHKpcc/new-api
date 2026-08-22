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

import type { TopNavLink } from '../types'

export const publicHeaderDesktopMediaQuery = '(min-width: 1024px)'
export const publicHeaderCompactLinkCount = 4
export const publicHeaderDesktopSideGap = 16

type PublicHeaderMeasurements = {
  barWidth: number
  navWidth: number
  leftWidth: number
  rightWidth: number
}

export function canPublicHeaderShowAllLinks(
  measurements: PublicHeaderMeasurements
): boolean {
  const widestSide = Math.max(measurements.leftWidth, measurements.rightWidth)
  const requiredWidth =
    measurements.navWidth + 2 * (widestSide + publicHeaderDesktopSideGap)

  return measurements.barWidth >= requiredWidth
}

export function isPublicNavLinkActive(
  pathname: string,
  link: TopNavLink
): boolean {
  if (link.external) return false
  if (link.isActive !== undefined) return link.isActive
  if (!link.href.startsWith('/')) return false
  if (link.href === '/') return pathname === '/'

  return pathname === link.href || pathname.startsWith(`${link.href}/`)
}

export function splitPublicHeaderLinks(links: readonly TopNavLink[]) {
  return {
    primaryLinks: links.slice(0, publicHeaderCompactLinkCount),
    overflowLinks: links.slice(publicHeaderCompactLinkCount),
  }
}

export const publicHeaderLayoutClasses = {
  // Same editorial wireframe × dark-tech language as the console header:
  // `--app-topbar-surface` paper/ink surface, hairline border, and the
  // `app-topbar-accent` gradient line along the bottom edge.
  header: {
    base: 'app-topbar-accent pointer-events-auto fixed inset-x-0 top-0 z-50 h-[var(--app-header-height,3.75rem)] border-b border-foreground/12 dark:border-border bg-(--app-topbar-surface) backdrop-blur-xl transition-[height,background-color,border-color,box-shadow] duration-200 ease-out motion-reduce:transition-none',
    idle: 'shadow-none',
    scrolled: 'shadow-[0_12px_32px_-24px_rgba(0,0,0,0.55)]',
  },
  shell: {
    base: 'app-header-bar mx-auto h-full max-w-7xl px-4 md:px-6',
  },
  bar: {
    base: 'grid min-w-0 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center transition-[height] duration-200 ease-out motion-reduce:transition-none',
    idle: 'h-full',
    scrolled: 'h-full',
  },
  left: 'app-header-left col-start-1 row-start-1 flex min-w-0 items-center gap-3 justify-self-start',
  brand: {
    link: 'app-header-brand group flex min-w-0 items-center gap-3 justify-self-start',
    mark: 'app-header-brand-mark ring-foreground/15 dark:ring-border flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-sm ring-1 transition-all duration-300 group-hover:scale-105',
    name: 'app-header-brand-name min-w-0 max-w-[min(42vw,13rem)] truncate text-xl leading-none font-semibold tracking-tight [font-family:var(--font-serif)] lg:max-w-40 xl:max-w-44 2xl:max-w-56',
  },
  desktopNav:
    'app-header-nav col-start-2 row-start-1 hidden h-full shrink-0 items-center justify-self-center lg:flex',
  desktopNavMeasurement:
    'pointer-events-none invisible absolute flex h-full w-max items-stretch',
  // 13px editorial nav label with a full-bleed hairline active bar flush
  // with the header's bottom edge; the bar glows primary in dark mode.
  desktopLink:
    "relative flex h-full shrink-0 items-center px-3.5 text-[0.8125rem] font-medium tracking-[0.01em] whitespace-nowrap [font-family:var(--font-sans)] transition-[background-color,color] duration-150 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 after:origin-center after:scale-x-0 after:bg-primary after:transition-transform after:duration-200 after:content-[''] hover:bg-foreground/[0.04] hover:text-foreground motion-reduce:after:transition-none dark:hover:bg-foreground/[0.07]",
  desktopLinkActive:
    'text-foreground after:scale-x-100 dark:after:shadow-[0_0_8px_var(--primary)]',
  desktopActions:
    'app-header-actions col-start-3 row-start-1 hidden shrink-0 items-center justify-self-end lg:flex [&>button_[data-slot=avatar]]:size-7',
  utilityActions:
    'app-header-utilities flex items-center gap-0.5 [&_button]:size-10 [&_button]:rounded-sm [&_button_svg]:size-5',
  authButton:
    'dark:ring-foreground/25 h-10 rounded-sm bg-foreground px-5 text-[0.8125rem] font-semibold tracking-[0.02em] text-background transition-[opacity,box-shadow] hover:opacity-90 dark:ring-1',
  authSkeleton: 'h-10 w-20 rounded-sm',
  mobileActions:
    'col-start-3 row-start-1 flex shrink-0 items-center gap-2 justify-self-end lg:hidden [&_button]:size-10 [&_button]:rounded-sm [&_button_svg]:size-5 [&_[data-slot=avatar]]:size-7',
  mobilePanel: 'w-full gap-0 border-l-0 sm:max-w-[26rem] sm:border-l lg:hidden',
  mobilePanelHeader:
    'flex-row items-center justify-between gap-4 border-b px-5 py-4 sm:px-6',
  mobilePanelBrand:
    'flex min-w-0 items-center gap-3 text-lg font-semibold [font-family:var(--font-serif)]',
  mobileNav: 'flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto px-5 py-5',
  mobileLink:
    'group grid min-h-12 grid-cols-[2rem_minmax(0,1fr)_auto] items-center border-l-2 px-3 py-2.5 text-base font-medium [font-family:var(--font-sans)] transition-[background-color,border-color,color] duration-150',
  mobileLinkActive: 'border-primary bg-primary/5 text-foreground',
  mobileLinkIdle:
    'border-transparent text-muted-foreground hover:border-border hover:bg-muted/50 hover:text-foreground',
  mobileLinkIndex: 'text-muted-foreground/60 text-xs font-medium tabular-nums',
  mobileAuthButton:
    'inline-flex h-11 items-center justify-center rounded-sm bg-foreground px-4 text-base font-medium text-background transition-opacity hover:opacity-90 active:opacity-80',
  mobileFooter: 'mt-auto gap-4 border-t p-5 sm:p-6',
  mobileLanguageAction:
    'flex items-center justify-between text-sm font-medium [font-family:var(--font-sans)] [&_button]:size-10 [&_button]:rounded-sm [&_button_svg]:size-5',
} as const
