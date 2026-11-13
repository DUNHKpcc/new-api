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
import { test } from 'node:test'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { RevenueOverview } from '../components/revenue-overview'

const i18n = createInstance()
await i18n
  .use(initReactI18next)
  .init({ lng: 'en', resources: { en: { translation: {} } } })

test('shows complete large revenue amounts and allows wrapping instead of ellipsis', () => {
  const client = new QueryClient({
    defaultOptions: { queries: { staleTime: Infinity } },
  })
  client.setQueryDefaults(['revenue', 'platform-summary'], {
    initialData: {
      data: {
        totals: [{ currency: 'CNY', amount_minor: '123456789', count: 12345 }],
        by_payment_method: [],
        timeline: [],
      },
    },
  })
  client.setQueryDefaults(['revenue', 'external-summary'], {
    initialData: { data: { totals: [], by_source: [], timeline: [] } },
  })
  client.setQueryDefaults(['revenue', 'costs'], {
    initialData: { data: { items: [] } },
  })
  const markup = renderToStaticMarkup(
    <QueryClientProvider client={client}>
      <I18nextProvider i18n={i18n}>
        <RevenueOverview />
      </I18nextProvider>
    </QueryClientProvider>
  )
  client.clear()
  const values = [
    ...markup.matchAll(/<p class="([^"]*)">CNY 1,234,567\.89<\/p>/g),
  ]
  assert.equal(values.length, 3)
  for (const [, classes] of values) {
    assert.doesNotMatch(classes, /truncate|overflow-hidden|text-ellipsis/)
    assert.match(classes, /wrap-anywhere/)
  }
})
