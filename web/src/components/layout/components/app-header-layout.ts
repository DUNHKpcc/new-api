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
  root: 'bg-background/95 sticky top-0 z-40 h-[var(--app-header-height,3.75rem)] w-full shrink-0 border-b backdrop-blur-xl',
  bar: 'flex h-full min-w-0 items-center gap-2 px-3 sm:gap-3 sm:px-4',
  sidebarTrigger: 'size-10 shrink-0 rounded-md [&_svg]:size-5',
  actions: 'ms-auto flex min-w-0 items-center gap-1.5 sm:gap-2',
  nav: 'me-1 hidden lg:block',
  search: 'h-10 sm:w-44 lg:w-56 xl:w-64',
  utilities:
    'flex shrink-0 items-center gap-0.5 [&>button]:size-10 [&>button]:rounded-md [&>button_svg]:size-5 [&_[data-slot=avatar]]:size-7',
  brand: {
    link: 'text-foreground inline-flex h-10 min-w-0 items-center gap-2.5 rounded-md px-2 font-medium transition-colors outline-none select-none',
    mark: 'flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg',
    name: 'hidden max-w-[12rem] min-w-0 truncate text-xl leading-none font-semibold [font-family:var(--font-serif)] md:block',
  },
  topNav: {
    compact: 'xl:hidden',
    compactTrigger: 'size-10 rounded-md [&_svg]:size-5',
    desktop: 'hidden h-10 items-center gap-1 xl:flex',
    link: 'hover:bg-accent hover:text-accent-foreground flex h-10 items-center rounded-md px-3 text-sm font-medium whitespace-nowrap transition-colors',
  },
} as const
