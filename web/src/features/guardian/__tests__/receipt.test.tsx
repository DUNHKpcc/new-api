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
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { ReceiptLink } from '../components/receipt-link'
import { GuardianReceiptPage } from '../receipt'
import type { GuardianReceipt } from '../types'

const clients: QueryClient[] = []
afterEach(() => {
  cleanup()
  clients.forEach((c) => c.clear())
  clients.length = 0
  vi.restoreAllMocks()
})
const receipt: GuardianReceipt = {
  order_id: 7,
  number: 'GRC-ORDER-7',
  trade_no: 'ORDER-7',
  service_title: 'Original product',
  billing_period: 'monthly',
  amount_cents: 30000,
  max_cents: 10000,
  ultra_cents: 0,
  payment_method: 'alipay',
  paid_at: 1761500100,
  created_at: 1761500200,
  recipient: 'b***@example.test',
  email_status: 'failed',
  refund_state: '',
  refund_cents: null,
  refunded_at: 0,
}
function show(data = receipt, admin = false) {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: { success: true, data: url.endsWith('/receipt') ? data : null },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => <GuardianReceiptPage orderId={7} admin={admin} />,
    }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}
it('shows original item amounts and submits email to the server-owned recipient', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  show()
  expect(await screen.findByText('GRC-ORDER-7')).toBeVisible()
  const frame = screen.getByTitle('Payment receipt') as HTMLIFrameElement
  const document = new DOMParser().parseFromString(frame.srcdoc, 'text/html')
  expect(document.body.textContent).toContain('¥200')
  expect(document.body.textContent).toContain('¥100')
  expect(document.body.textContent).toContain('¥300')
  expect(document.body.textContent).not.toContain('Send to account email')
  const printWindow = frame.contentWindow
  if (!printWindow) throw new Error('Receipt preview window missing')
  const print = vi
    .spyOn(printWindow, 'print')
    .mockImplementation(() => undefined)
  await userEvent.click(
    screen.getByRole('button', { name: 'Print / Save PDF' })
  )
  expect(print).toHaveBeenCalledOnce()
  await userEvent.click(
    screen.getByRole('button', { name: 'Send to account email' })
  )
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/api/guardian/orders/7/receipt/email',
      {},
      undefined
    )
  )
})
it('keeps the original payment visible after a refund and disables mail without an account email', async () => {
  show({
    ...receipt,
    recipient: '',
    email_status: 'no_email',
    refund_state: 'completed',
    refund_cents: 29800,
  })
  const frame = (await screen.findByTitle(
    'Payment receipt'
  )) as HTMLIFrameElement
  const document = new DOMParser().parseFromString(frame.srcdoc, 'text/html')
  expect(document.body.textContent).toContain('¥298')
  expect(document.body.textContent).toContain('¥300')
  expect(
    screen.getByRole('button', { name: 'Send to account email' })
  ).toBeDisabled()
})

it('returns administrators to order management', async () => {
  show(receipt, true)
  await screen.findByTitle('Payment receipt')
  expect(screen.getByRole('button', { name: 'Admin area' })).toHaveAttribute(
    'href',
    '/guardian-admin'
  )
  expect(api.get).toHaveBeenCalledWith('/api/guardian/admin/orders/7/receipt')
})

it('opens receipts through the router in the current tab', async () => {
  const root = createRootRoute()
  const index = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => <ReceiptLink orderId={7} admin />,
  })
  const destination = createRoute({
    getParentRoute: () => root,
    path: '/guardian/receipt',
    validateSearch: (search: Record<string, unknown>) => ({
      order: Number(search.order),
      admin: search.admin === true,
    }),
    component: () => <p>Receipt destination</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([index, destination]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(<RouterProvider router={router} />)
  const link = await screen.findByRole('button', {
    name: 'View payment receipt',
  })
  expect(link).not.toHaveAttribute('target')
  await userEvent.click(link)
  expect(await screen.findByText('Receipt destination')).toBeVisible()
  expect(router.state.location.search).toMatchObject({ order: 7, admin: true })
})
