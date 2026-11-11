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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createRootRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { GuardianOrders } from '../orders'

let client: QueryClient

afterEach(() => {
  cleanup()
  client?.clear()
  vi.restoreAllMocks()
})

it('submits the order filter to the server and resets the search', async () => {
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: url.endsWith('/orders/self') ? { items: [], total: 0 } : [],
    },
  }))
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createRouter({
    routeTree: createRootRoute({ component: () => <GuardianOrders /> }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole('textbox', { name: 'Order number' })
  await userEvent.type(
    screen.getByRole('textbox', { name: 'Order number' }),
    '  ORDER-7  '
  )
  await userEvent.click(screen.getByRole('button', { name: 'Search' }))
  await waitFor(() =>
    expect(get).toHaveBeenCalledWith('/api/guardian/orders/self', {
      params: {
        p: 1,
        page_size: 10,
        order_number: 'ORDER-7',
        created_after: undefined,
        created_before: undefined,
      },
    })
  )
  expect(await screen.findByText('No results found')).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: 'Reset' }))
  expect(screen.getByRole('textbox', { name: 'Order number' })).toHaveValue('')
  expect(await screen.findByText('No orders yet')).toBeVisible()
})
