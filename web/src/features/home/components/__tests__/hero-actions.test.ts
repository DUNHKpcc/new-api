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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
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

import { useAuthStore } from '@/stores/auth-store'

import { Hero } from '../sections/hero'

async function renderHeroActions(isAuthenticated: boolean) {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    resources: {
      en: {
        translation: {
          'Resource Downloads': 'Resource Downloads',
          'View Pricing': 'View Pricing',
        },
      },
    },
  })

  const queryClient = new QueryClient()
  queryClient.setQueryData(['status'], {})
  useAuthStore.getState().auth.reset('idle')

  const TestPage = () =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        I18nextProvider,
        { i18n },
        createElement(Hero, { isAuthenticated })
      )
    )

  const rootRoute = createRootRoute()
  const routes = [
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/',
      component: TestPage,
    }),
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/sign-up',
      component: () => null,
    }),
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/dashboard',
      component: () => null,
    }),
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/pricing',
      component: () => null,
    }),
    createRoute({
      getParentRoute: () => rootRoute,
      path: '/resource-downloads',
      component: () => null,
    }),
  ]
  const router = createRouter({
    routeTree: rootRoute.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })

  await router.load()
  const markup = renderToStaticMarkup(createElement(RouterProvider, { router }))
  queryClient.clear()

  return markup
}

describe('home hero actions', () => {
  test('renders a resource download link in the public hero', async () => {
    const markup = await renderHeroActions(false)

    assert.match(markup, /href="\/resource-downloads"/)
    assert.match(markup, />Resource Downloads<\/span>/)
  })

  test('renders a pricing link for authenticated users', async () => {
    const markup = await renderHeroActions(true)

    assert.match(markup, /href="\/pricing"/)
    assert.match(markup, />View Pricing<\/a>/)
  })
})
