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

export const appHeaderLayoutClasses = {
  // Editorial wireframe × dark-tech topbar: near-solid paper surface in
  // light mode, deeper-than-page ink surface in dark mode (both via the
  // `--app-topbar-surface` token), hairline border, and the signature
  // gradient accent line contributed by `app-topbar-accent`.
  root: 'app-topbar-accent border-foreground/12 dark:border-border bg-(--app-topbar-surface) sticky top-0 z-40 h-[var(--app-header-height,3.75rem)] w-full shrink-0 border-b backdrop-blur-xl',
  bar: 'app-header-bar mx-auto grid h-full max-w-7xl min-w-0 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 px-4 md:px-6',
  left: 'app-header-left col-start-1 row-start-1 flex min-w-0 items-center gap-2 justify-self-start',
  sidebarTrigger: 'size-10 shrink-0 rounded-sm [&_svg]:size-5',
  actions:
    'app-header-actions col-start-3 row-start-1 flex min-w-0 items-center justify-self-end',
  nav: 'app-header-nav col-start-2 row-start-1 hidden h-full shrink-0 items-center justify-self-center lg:flex',
  search: 'app-header-search-icon size-10 shrink-0 rounded-sm',
  utilities:
    'app-header-utilities flex shrink-0 items-center gap-0.5 [&>button]:size-10 [&>button]:rounded-sm [&>button_svg]:size-5 [&_[data-slot=avatar]]:size-7',
  brand: {
    link: 'app-header-brand text-foreground inline-flex h-10 min-w-0 items-center gap-3 font-medium transition-colors outline-none select-none',
    mark: 'app-header-brand-mark ring-foreground/15 dark:ring-border flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-sm ring-1',
    name: 'app-header-brand-name hidden max-w-[12rem] min-w-0 truncate text-xl leading-none font-semibold tracking-tight [font-family:var(--font-serif)] md:block',
  },
  topNav: {
    compact: 'lg:hidden',
    compactTrigger: 'size-10 rounded-sm [&_svg]:size-5',
    desktop: 'hidden h-full items-center lg:flex',
    // 13px editorial nav label; the active indicator is a full-bleed
    // hairline bar flush with the header's bottom edge (animated via
    // scale-x) instead of the previous small centered dash. In dark mode
    // the bar gains a soft primary glow.
    link: "relative flex h-full shrink-0 items-center px-3.5 text-[0.8125rem] font-medium tracking-[0.01em] whitespace-nowrap [font-family:var(--font-sans)] transition-[background-color,color] duration-150 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 after:origin-center after:scale-x-0 after:bg-primary after:transition-transform after:duration-200 after:content-[''] hover:bg-foreground/[0.04] hover:text-foreground motion-reduce:after:transition-none dark:hover:bg-foreground/[0.07]",
    linkActive:
      'text-foreground after:scale-x-100 dark:after:shadow-[0_0_8px_var(--primary)]',
  },
} as const
