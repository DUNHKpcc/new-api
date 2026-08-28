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
import { after, describe, test } from 'node:test'

import { createInstance } from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

const { mock } = await import(`bun:${'test'}`)

const order = {
  id: 1,
  user_id: 42,
  amount: 100,
  money: 99,
  trade_no: 'wallet-42',
  payment_method: 'stripe',
  payment_provider: 'stripe',
  create_time: 1_700_000_000,
  complete_time: 1_700_000_010,
  status: 'success' as const,
}

const pendingOrder = {
  ...order,
  id: 2,
  trade_no: 'wallet-pending-42',
  status: 'pending' as const,
  complete_time: 0,
}

mock.module('@tanstack/react-query', () => ({
  keepPreviousData: {},
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[1] === 'platform-orders') {
      return {
        data: { data: { items: [order, pendingOrder], total: 2 } },
        isLoading: false,
        isError: false,
      }
    }

    return {
      data: { users: {}, subscriptions: {}, planTitles: {} },
      isLoading: false,
      isError: false,
    }
  },
}))

const { PlatformOrders } = await import('../components/platform-orders')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

describe('platform order display', () => {
  after(() => mock.restore())

  test('shows the credited amount separately from the payment amount', () => {
    const markup = renderToStaticMarkup(
      createElement(I18nextProvider, { i18n }, createElement(PlatformOrders))
    )

    assert.match(markup, />Amount<\/th>/)
    assert.match(markup, />Payment<\/th>/)
    assert.match(markup, />\$100<\/td>/)
    assert.match(markup, />99<\/td>/)
  })

  test('shows pending status and the manual completion action', () => {
    const markup = renderToStaticMarkup(
      createElement(I18nextProvider, { i18n }, createElement(PlatformOrders))
    )

    assert.match(markup, />Pending<\/span>/)
    assert.match(markup, />Complete Order<\/button>/)
    assert.match(markup, /wallet-pending-42/)
  })
})
