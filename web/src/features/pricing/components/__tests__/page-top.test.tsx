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

import type { PlanRecord } from '@/features/subscriptions/types'

import { PricingPageTop } from '../pricing-page-top'

const plans: PlanRecord[] = [
  {
    plan: {
      id: 1,
      title: 'Lite',
      subtitle: 'For individual developers',
      price_amount: 9.9,
      currency: 'USD',
      duration_unit: 'month',
      duration_value: 1,
      quota_reset_period: 'monthly',
      enabled: true,
      sort_order: 1,
      allow_balance_pay: true,
      allow_wallet_overflow: true,
      max_purchase_per_user: 0,
      total_amount: 5_000_000,
    },
  },
  {
    plan: {
      id: 2,
      title: 'Professional plan with a deliberately long title',
      price_amount: 29,
      currency: 'USD',
      duration_unit: 'month',
      duration_value: 1,
      quota_reset_period: 'never',
      enabled: true,
      sort_order: 2,
      allow_balance_pay: true,
      allow_wallet_overflow: true,
      max_purchase_per_user: 0,
      total_amount: 0,
    },
  },
]

async function renderPricingPageTop(records: PlanRecord[]) {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    resources: { en: { translation: {} } },
    returnNull: false,
  })

  const TestPage = () =>
    createElement(
      I18nextProvider,
      { i18n },
      createElement(PricingPageTop, {
        plans: records,
        searchInput: '',
        onSearchChange: () => undefined,
        onClearSearch: () => undefined,
      })
    )

  const rootRoute = createRootRoute()
  const routes = [
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/pricing',
      component: TestPage,
    }),
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/wallet',
      component: () => null,
    }),
  ]
  const router = createRouter({
    routeTree: rootRoute.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ['/pricing'] }),
  })

  await router.load()
  return renderToStaticMarkup(createElement(RouterProvider, { router }))
}

describe('pricing page top', () => {
  test('replaces the model marketing header with the search field', async () => {
    const markup = await renderPricingPageTop([])

    assert.doesNotMatch(markup, /<h1/)
    assert.doesNotMatch(markup, /Model Square/)
    assert.doesNotMatch(markup, /This site currently has/)
    assert.match(markup, /aria-label="Search models"/)
    assert.doesNotMatch(markup, /data-subscription-plan-showcase/)
  })

  test('shows every configured plan above the model search field', async () => {
    const markup = await renderPricingPageTop(plans)
    const showcasePosition = markup.indexOf(
      'data-subscription-plan-showcase="true"'
    )
    const searchPosition = markup.indexOf('aria-label="Search models"')

    assert.ok(showcasePosition >= 0)
    assert.ok(searchPosition > showcasePosition)
    assert.match(markup, /data-subscription-plan-id="1"/)
    assert.match(markup, /data-subscription-plan-id="2"/)
    assert.match(markup, />Lite</)
    assert.match(markup, /Professional plan with a deliberately long title/)
    assert.match(markup, /¥9\.90/)
    assert.doesNotMatch(markup, /\$9\.90/)
    assert.match(markup, /href="\/wallet"/)
    assert.match(markup, />Subscribe Now</)
    assert.match(markup, /data-subscription-plan-carousel="true"/)
    assert.match(markup, /basis-\[88%\].*sm:basis-1\/2.*xl:basis-1\/3/)
    assert.match(markup, /aria-label="Previous slide"/)
    assert.match(markup, /aria-label="Next slide"/)
  })
})
