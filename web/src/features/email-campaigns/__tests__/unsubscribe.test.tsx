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
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { EmailUnsubscribe } from '../unsubscribe'

const clients: QueryClient[] = []
async function renderUnsubscribe(token: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  const root = createRootRoute({
    component: () => (
      <QueryClientProvider client={client}>
        <EmailUnsubscribe token={token} />
      </QueryClientProvider>
    ),
  })
  const router = createRouter({
    routeTree: root,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
}
afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})
it('only reads preferences on opening and unsubscribes after an explicit confirmation', async () => {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/email/unsubscribe'
          ? { category: 'promotion', unsubscribed: false }
          : {},
    },
  }))
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      success: true,
      data: { category: 'promotion', unsubscribed: true },
    },
  })
  await renderUnsubscribe('test-token')
  const button = await screen.findByRole('button', {
    name: 'Confirm unsubscribe',
  })
  expect(post).not.toHaveBeenCalled()
  expect(document.body).not.toHaveTextContent('test-token')
  expect(
    screen.queryByRole('link', { name: /Sign in/i })
  ).not.toBeInTheDocument()
  await userEvent.click(button)
  expect(await screen.findByRole('status')).toHaveTextContent(
    'You are unsubscribed'
  )
  expect(post).toHaveBeenCalledWith(
    '/api/email/unsubscribe',
    { confirm: true },
    { params: { token: 'test-token' } }
  )
})
it('shows an invalid link state without requesting a missing token', async () => {
  const get = vi
    .spyOn(api, 'get')
    .mockResolvedValue({ data: { success: true, data: {} } })
  await renderUnsubscribe('')
  expect(await screen.findByText('Invalid unsubscribe link')).toBeVisible()
  expect(get.mock.calls.some(([url]) => url === '/api/email/unsubscribe')).toBe(
    false
  )
  expect(
    screen.queryByRole('button', { name: 'Confirm unsubscribe' })
  ).not.toBeInTheDocument()
})
it('shows retryable failure when the unsubscribe POST fails', async () => {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/email/unsubscribe'
          ? { category: 'platform', unsubscribed: false }
          : {},
    },
  }))
  vi.spyOn(api, 'post').mockRejectedValue(new Error('Unavailable'))
  await renderUnsubscribe('test-token')
  await userEvent.click(
    await screen.findByRole('button', { name: 'Confirm unsubscribe' })
  )
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Unsubscribe failed'
  )
  await waitFor(() =>
    expect(
      screen.getByRole('button', { name: 'Confirm unsubscribe' })
    ).toBeEnabled()
  )
})
