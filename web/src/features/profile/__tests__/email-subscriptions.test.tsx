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
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { EmailSubscriptionsCard } from '../components/email-subscriptions-card'

const clients: QueryClient[] = []
function renderSubscriptions() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  render(
    <QueryClientProvider client={client}>
      <EmailSubscriptionsCard />
    </QueryClientProvider>
  )
}
afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})
it('persists the changed category while retaining the other server preference', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { promotion: false, platform: true } },
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: { promotion: true, platform: true } },
  })
  renderSubscriptions()
  const promotion = await screen.findByRole('switch', {
    name: 'Promotional emails',
  })
  expect(promotion).not.toBeChecked()
  expect(screen.getByRole('switch', { name: 'Platform updates' })).toBeChecked()
  await userEvent.click(promotion)
  await waitFor(() => expect(promotion).toBeChecked())
  expect(put).toHaveBeenCalledWith('/api/user/self/email-subscriptions', {
    promotion: true,
    platform: true,
  })
})
it('retains the saved preference when a server rejects the update', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { promotion: false, platform: true } },
  })
  vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: false, message: 'Save rejected' },
  })
  renderSubscriptions()
  const platform = await screen.findByRole('switch', {
    name: 'Platform updates',
  })
  await userEvent.click(platform)
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Failed to save email preferences'
  )
  expect(platform).toBeChecked()
})
