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

import { Support } from '..'

async function renderSupportPage() {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    resources: { en: { translation: {} } },
  })

  const queryClient = new QueryClient()
  const TestPage = () =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(I18nextProvider, { i18n }, createElement(Support))
    )
  const rootRoute = createRootRoute()
  const supportRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/support',
    component: TestPage,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([supportRoute]),
    history: createMemoryHistory({ initialEntries: ['/support'] }),
  })

  await router.load()
  return renderToStaticMarkup(createElement(RouterProvider, { router }))
}

describe('customer support page', () => {
  test('shows the QQ and Telegram support channels with scannable QR images', async () => {
    const markup = await renderSupportPage()

    assert.match(markup, /Customer Support/)
    assert.match(markup, /data-support-channel="qq"/)
    assert.match(markup, /QQ support group QR code/)
    assert.match(markup, /data-support-channel="telegram"/)
    assert.match(markup, /Telegram group QR code/)
  })

  test('shows the official Bilibili link and Douyin account QR code', async () => {
    const markup = await renderSupportPage()

    assert.match(markup, /data-official-account="bilibili"/)
    assert.match(
      markup,
      /href="https:\/\/space\.bilibili\.com\/397853169\?spm_id_from=333\.1007\.0\.0"/
    )
    assert.match(markup, /target="_blank"/)
    assert.match(markup, /rel="noopener noreferrer"/)
    assert.match(markup, /data-official-account="douyin"/)
    assert.match(markup, /Douyin official account QR code/)
    assert.match(markup, /933648547/)
    assert.match(markup, /Occasional live streams with free Agent tool setup/)
  })
})
