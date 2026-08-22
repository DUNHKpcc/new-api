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
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { createInstance } from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { AffiliateSection } from '../affiliate'

async function renderAffiliateSection() {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    resources: {
      en: {
        translation: {
          'Affiliate Center': 'Affiliate Center',
          'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.':
            'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.',
          'Invitation rewards and commission management':
            'Invitation rewards and commission management',
          'Invitation rewards and verified online payment commissions.':
            'Invitation rewards and verified online payment commissions.',
          'Invite friends': 'Invite friends',
          'Move affiliate rewards to your main balance':
            'Move affiliate rewards to your main balance',
          'Open Affiliate Center': 'Open Affiliate Center',
          'Referral Program': 'Referral Program',
          'Rewards are available after referred users register.':
            'Rewards are available after referred users register.',
          'Share your invitation link and earn registration rewards when available.':
            'Share your invitation link and earn registration rewards when available.',
          'Share your link and earn rewards':
            'Share your link and earn rewards',
          'Transfer Rewards': 'Transfer Rewards',
        },
      },
    },
  })

  const rootRoute = createRootRoute()
  const routes = [
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/',
      component: () =>
        createElement(
          I18nextProvider,
          { i18n },
          createElement(AffiliateSection)
        ),
    }),
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/affiliate',
      component: () => null,
    }),
  ]
  const router = createRouter({
    routeTree: rootRoute.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })

  await router.load()
  return renderToStaticMarkup(createElement(RouterProvider, { router }))
}

describe('affiliate home section', () => {
  test('introduces the referral program and links to the affiliate center', async () => {
    const markup = await renderAffiliateSection()

    assert.match(markup, /data-home-section="affiliate"/)
    assert.match(markup, /aria-label="Referral Program"/)
    assert.match(markup, /href="\/affiliate"/)
    assert.match(markup, /Share your link and earn rewards/)
    assert.match(markup, /sm:grid-cols-3/)
  })
})
