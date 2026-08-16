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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  canPublicHeaderShowAllLinks,
  isPublicNavLinkActive,
  publicHeaderCompactLinkCount,
  publicHeaderDesktopMediaQuery,
  publicHeaderLayoutClasses,
  splitPublicHeaderLinks,
} from '../public-header-layout'

function tokens(classes: string) {
  return classes.split(' ')
}

describe('public header layout', () => {
  test('keeps the public header at the shared app header height', () => {
    assert.ok(
      tokens(publicHeaderLayoutClasses.header.base).includes('border-b')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.header.base).includes(
        'h-[var(--app-header-height,3.75rem)]'
      )
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.header.base).includes(
        'app-topbar-accent'
      ),
      'mounts the gradient accent hairline along the bottom edge'
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.header.base).includes(
        'bg-(--app-topbar-surface)'
      ),
      'surface comes from the --app-topbar-surface token so presets and dark mode re-theme it'
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.shell.base).includes('max-w-7xl')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.shell.base).includes('app-header-bar'),
      'uses the same preset-aware bar width and spacing as the console header'
    )
    assert.ok(tokens(publicHeaderLayoutClasses.shell.base).includes('h-full'))
    assert.ok(tokens(publicHeaderLayoutClasses.bar.idle).includes('h-full'))
    assert.ok(tokens(publicHeaderLayoutClasses.bar.scrolled).includes('h-full'))
    assert.ok(
      tokens(publicHeaderLayoutClasses.left).includes('app-header-left')
    )
    assert.equal(
      publicHeaderLayoutClasses.bar.idle,
      publicHeaderLayoutClasses.bar.scrolled
    )
    assert.equal(
      tokens(publicHeaderLayoutClasses.header.scrolled).includes('rounded-2xl'),
      false
    )
    assert.ok(tokens(publicHeaderLayoutClasses.brand.mark).includes('size-8'))
    assert.ok(
      tokens(publicHeaderLayoutClasses.brand.mark).includes(
        'app-header-brand-mark'
      )
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.brand.name).includes(
        'app-header-brand-name'
      )
    )
    assert.ok(tokens(publicHeaderLayoutClasses.brand.mark).includes('ring-1'))
    assert.ok(
      tokens(publicHeaderLayoutClasses.brand.mark).includes('rounded-sm')
    )
    assert.ok(tokens(publicHeaderLayoutClasses.brand.name).includes('text-xl'))
    assert.ok(
      tokens(publicHeaderLayoutClasses.brand.name).includes('tracking-tight')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.desktopLink).includes('text-[0.8125rem]')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.desktopLink).includes('after:inset-x-0'),
      'active indicator spans the full link width instead of a centered dash'
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.desktopLinkActive).includes(
        'after:scale-x-100'
      )
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.authButton).includes('bg-foreground')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.authButton).includes('rounded-sm')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.utilityActions).includes(
        '[&_button]:size-10'
      )
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.utilityActions).includes(
        '[&_button]:rounded-sm'
      )
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.desktopNav).includes('app-header-nav')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.desktopNav).includes('items-center')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.desktopActions).includes(
        'app-header-actions'
      )
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.utilityActions).includes(
        'app-header-utilities'
      )
    )
  })

  test('measures desktop capacity and keeps the sheet below desktop width', () => {
    const desktopNavTokens = tokens(publicHeaderLayoutClasses.desktopNav)
    const measurementTokens = tokens(
      publicHeaderLayoutClasses.desktopNavMeasurement
    )
    const mobileTokens = tokens(publicHeaderLayoutClasses.mobileActions)
    const panelTokens = tokens(publicHeaderLayoutClasses.mobilePanel)

    assert.ok(desktopNavTokens.includes('lg:flex'))
    assert.ok(measurementTokens.includes('absolute'))
    assert.ok(measurementTokens.includes('invisible'))
    assert.ok(measurementTokens.includes('w-max'))
    assert.equal(publicHeaderDesktopMediaQuery, '(min-width: 1024px)')
    assert.equal(publicHeaderCompactLinkCount, 4)
    assert.ok(mobileTokens.includes('lg:hidden'))
    assert.ok(mobileTokens.includes('[&_[data-slot=avatar]]:size-7'))
    assert.ok(panelTokens.includes('lg:hidden'))
    assert.ok(panelTokens.includes('w-full'))
    assert.ok(tokens(publicHeaderLayoutClasses.mobileNav).includes('flex-1'))
  })

  test('shows every desktop link only when both side columns retain spacing', () => {
    assert.equal(
      canPublicHeaderShowAllLinks({
        barWidth: 932,
        navWidth: 420,
        leftWidth: 160,
        rightWidth: 240,
      }),
      true
    )
    assert.equal(
      canPublicHeaderShowAllLinks({
        barWidth: 931,
        navWidth: 420,
        leftWidth: 160,
        rightWidth: 240,
      }),
      false
    )
  })

  test('allows the brand label to shrink without pushing actions off screen', () => {
    assert.ok(tokens(publicHeaderLayoutClasses.bar.base).includes('min-w-0'))
    assert.ok(tokens(publicHeaderLayoutClasses.brand.link).includes('min-w-0'))
    assert.ok(tokens(publicHeaderLayoutClasses.brand.name).includes('truncate'))
    assert.ok(
      tokens(publicHeaderLayoutClasses.mobileActions).includes('shrink-0')
    )
    assert.ok(
      tokens(publicHeaderLayoutClasses.mobileActions).includes('col-start-3')
    )
  })

  test('keeps nested routes active without matching sibling prefixes', () => {
    assert.equal(isPublicNavLinkActive('/', { title: 'Home', href: '/' }), true)
    assert.equal(
      isPublicNavLinkActive('/dashboard/models', {
        title: 'Console',
        href: '/dashboard',
      }),
      true
    )
    assert.equal(
      isPublicNavLinkActive('/dashboard-tools', {
        title: 'Console',
        href: '/dashboard',
      }),
      false
    )
    assert.equal(
      isPublicNavLinkActive('/docs', {
        title: 'Docs',
        href: 'https://docs.example.com',
        external: true,
      }),
      false
    )
  })

  test('moves only overflow links into the compact More menu', () => {
    const links = [
      { title: 'Home', href: '/' },
      { title: 'Console', href: '/dashboard' },
      { title: 'Models', href: '/pricing' },
      { title: 'Rankings', href: '/rankings' },
      { title: 'Docs', href: '/docs' },
      { title: 'Support', href: '/about' },
    ]

    const result = splitPublicHeaderLinks(links)

    assert.deepEqual(
      result.primaryLinks.map((link) => link.href),
      ['/', '/dashboard', '/pricing', '/rankings']
    )
    assert.deepEqual(
      result.overflowLinks.map((link) => link.href),
      ['/docs', '/about']
    )
  })
})
