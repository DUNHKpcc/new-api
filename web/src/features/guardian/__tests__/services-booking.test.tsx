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
import { useAuthStore } from '@/stores/auth-store'

import { GuardianServiceDialog } from '../components/admin/service-dialog'
import { ServicesSection } from '../components/services-section'
import type { GuardianCatalog } from '../types'

const navigate = vi.fn()
vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  useNavigate: () => navigate,
}))

const catalog: GuardianCatalog = {
  services: [
    {
      id: 7,
      title: 'VM route setup',
      subtitle: '',
      description: '',
      features: ['Omarchy guest'],
      price_cents: 39900,
      delivery_note: '',
      badge: '',
      route: 'vm',
      enabled: true,
      sort_order: 0,
      created_at: 1,
      updated_at: 1,
    },
  ],
  pay_methods: [{ type: 'alipay', name: 'Alipay' }],
  purchase_enabled: true,
}

afterEach(() => {
  cleanup()
  navigate.mockReset()
  useAuthStore.getState().auth.reset()
  vi.restoreAllMocks()
})

function renderSection(props: {
  requestedServiceId?: number
  onRequestHandled?: () => void
}) {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, message: '', data: catalog },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ServicesSection
        requestedServiceId={props.requestedServiceId}
        onRequestHandled={props.onRequestHandled ?? (() => undefined)}
      />
    </QueryClientProvider>
  )
}

it('sends a signed-out visitor to sign in and back to the services page for that service', async () => {
  renderSection({})

  await userEvent.click(
    await screen.findByRole('button', { name: 'Book this service' })
  )

  expect(navigate).toHaveBeenCalledWith({
    to: '/sign-in',
    search: { redirect: '/guardian/services?service=7' },
  })
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
})

it('reopens checkout for the requested service once the visitor is signed in', async () => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'alice', role: 1 })
  const onRequestHandled = vi.fn()
  renderSection({ requestedServiceId: 7, onRequestHandled })

  expect(
    await screen.findByRole('dialog', { name: 'VM route setup' })
  ).toBeVisible()
  await waitFor(() => expect(onRequestHandled).toHaveBeenCalledTimes(1))
})

it('shows sales to guests, including zero, and hides an unavailable count', async () => {
  catalog.services[0].sold_count = 1280
  renderSection({})
  expect((await screen.findByText('1,280')).parentElement).toHaveTextContent(
    'Sold: 1,280'
  )
  cleanup()
  catalog.services[0].sold_count = 0
  renderSection({})
  expect((await screen.findByText('0')).parentElement).toHaveTextContent(
    'Sold: 0'
  )
  cleanup()
  delete catalog.services[0].sold_count
  renderSection({})
  await screen.findByRole('button', { name: 'Book this service' })
  expect(screen.queryByText(/^Sold:/)).not.toBeInTheDocument()
})

it('previews the base sales adjustment and saves zero without submitting derived totals', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true, data: {} } })
  const onSaved = vi.fn()
  render(
    <QueryClientProvider client={new QueryClient()}>
      <GuardianServiceDialog
        service={{
          ...catalog.services[0],
          base_sales: 5,
          actual_sales: 12,
          sold_count: 17,
        }}
        onClose={() => undefined}
        onSaved={onSaved}
      />
    </QueryClientProvider>
  )
  expect(screen.getByText(/Actual sales: 12/)).toBeVisible()
  expect(screen.getByText(/Displayed sales: 17/)).toBeVisible()
  const input = screen.getByRole('spinbutton', { name: 'Base sales' })
  await userEvent.clear(input)
  await userEvent.type(input, '-1')
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(
    await screen.findByText('Enter a whole number from 0 to 1,000,000,000')
  ).toBeVisible()
  expect(put).not.toHaveBeenCalled()
  await userEvent.clear(input)
  await userEvent.type(input, '0')
  expect(screen.getByText(/Displayed sales: 12/)).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledOnce())
  expect(put.mock.calls[0][1]).toMatchObject({ base_sales: 0 })
  expect(put.mock.calls[0][1]).not.toHaveProperty('actual_sales')
  expect(put.mock.calls[0][1]).not.toHaveProperty('sold_count')
})
