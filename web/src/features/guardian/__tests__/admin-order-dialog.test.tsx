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
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { GuardianOrderDialog } from '../components/admin/order-dialog'
import type { AdminGuardianOrder } from '../types'

const order: AdminGuardianOrder = {
  id: 9,
  trade_no: 'SVCUSR3NOxyz1761500000',
  user_id: 3,
  username: 'alice',
  service_id: 4,
  service_title: 'VPS dev machine',
  amount_cents: 59900,
  payment_method: 'alipay',
  status: 'paid',
  route: 'vps',
  os: 'linux',
  contact_type: 'telegram',
  contact: '@alice',
  note: '',
  admin_note: '',
  created_at: 1761500000,
  paid_at: 1761500100,
  delivered_at: 0,
  updated_at: 1761500100,
}

const clients: QueryClient[] = []
async function renderDialog(node: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  const router = createRouter({
    routeTree: createRootRoute({ component: () => node }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

function statusOptions() {
  return within(within(screen.getByRole('dialog')).getByLabelText('Status'))
    .getAllByRole('option')
    .map((option) => (option as HTMLOptionElement).value)
}

it('offers only the transitions the backend allows from a paid order', async () => {
  await renderDialog(
    <GuardianOrderDialog
      order={order}
      onClose={() => undefined}
      onSaved={() => undefined}
    />
  )
  expect(statusOptions()).toEqual([
    'paid',
    'scheduled',
    'in_progress',
    'refunded',
  ])
})

it('locks the status of a closed order', async () => {
  await renderDialog(
    <GuardianOrderDialog
      order={{ ...order, status: 'expired' }}
      onClose={() => undefined}
      onSaved={() => undefined}
    />
  )
  expect(screen.getByLabelText('Status')).toBeDisabled()
  expect(statusOptions()).toEqual(['expired'])
})

it('warns that marking refunded does not move money, then saves status and trimmed note', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: {
      success: true,
      message: '',
      data: { ...order, status: 'refunded' },
    },
  })
  const onSaved = vi.fn()
  await renderDialog(
    <GuardianOrderDialog
      order={order}
      onClose={() => undefined}
      onSaved={onSaved}
    />
  )

  await userEvent.selectOptions(screen.getByLabelText('Status'), 'refunded')
  expect(
    screen.getByText(/Marking as refunded does not send money back/)
  ).toBeVisible()
  await userEvent.type(
    screen.getByLabelText('Message to the customer'),
    '  Refunded to Alipay.  '
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))

  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(put).toHaveBeenCalledWith('/api/guardian/admin/orders/9', {
    status: 'refunded',
    admin_note: 'Refunded to Alipay.',
  })
})

it('loads history only when expanded and displays before and after notes', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        total: 1,
        items: [
          {
            id: 1,
            actor_id: 2,
            actor_type: 'admin',
            action: 'admin_updated',
            created_at: 1761500100,
            before: {
              status: 'paid',
              admin_note: 'old note',
              contact_type: 'telegram',
              contact: '@alice',
              note: '',
            },
            after: {
              status: 'scheduled',
              admin_note: 'new note',
              contact_type: 'telegram',
              contact: '@alice',
              note: '',
            },
          },
        ],
      },
    },
  })
  await renderDialog(
    <GuardianOrderDialog
      order={order}
      onClose={() => undefined}
      onSaved={() => undefined}
    />
  )
  expect(get).not.toHaveBeenCalled()
  await userEvent.click(screen.getByRole('button', { name: 'Order history' }))
  expect(await screen.findByText('Before: old note')).toBeInTheDocument()
  expect(screen.getByText('After: new note')).toBeInTheDocument()
  expect(get).toHaveBeenCalledWith('/api/guardian/admin/orders/9/history', {
    params: { p: 1, page_size: 10 },
  })
})

it('shows payment anomaly amounts and repeated callback count', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        total: 1,
        items: [
          {
            id: 1,
            expected_cents: 59900,
            paid_money: '0.01',
            payment_method: 'alipay',
            occurrences: 3,
            first_seen_at: 1761500100,
            last_seen_at: 1761500200,
          },
        ],
      },
    },
  })
  await renderDialog(
    <GuardianOrderDialog
      order={order}
      onClose={() => undefined}
      onSaved={() => undefined}
    />
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Payment anomalies' })
  )
  expect(
    await screen.findByText(/Callback amount.*0.01 CNY/)
  ).toBeInTheDocument()
  expect(screen.getByText(/Occurrences.*3/)).toBeInTheDocument()
})

it('manages direct and reseller orders together and reconciles the selected reseller order', async () => {
  const { GuardianOrdersPanel } =
    await import('../components/admin/orders-panel')
  const sale = {
    order_id: 10,
    reseller_id: 7,
    trade_no: 'RESELLER-10',
    amount_cents: 26000,
    supply_cents: 20000,
    estimated_fee_cents: 156,
    actual_fee_cents: 0,
    fee_confirmed_at: 0,
    paid_at: 1761500100,
    refund_state: '',
  }
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/guardian/admin/orders/10/reseller'
          ? sale
          : {
              items: [
                order,
                {
                  ...order,
                  id: 10,
                  trade_no: sale.trade_no,
                  reseller_code: 'agent-seven',
                  amount_cents: sale.amount_cents,
                },
              ],
              total: 2,
            },
    },
  }))
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  await renderDialog(<GuardianOrdersPanel />)
  const resellerRow = await screen.findByRole('row', { name: /RESELLER-10/ })
  expect(within(resellerRow).getByText('Reseller')).toBeVisible()
  const directRow = screen.getByRole('row', {
    name: new RegExp(order.trade_no),
  })
  expect(within(directRow).getByText('Platform direct')).toBeVisible()
  await userEvent.click(
    within(resellerRow).getByRole('button', { name: 'Manage' })
  )
  expect(await screen.findByText('¥200')).toBeVisible()
  expect(screen.getByText('¥58.44')).toBeVisible()
  expect(statusOptions()).not.toContain('refunded')
  expect(get).toHaveBeenCalledWith('/api/guardian/admin/orders/10/reseller')
  await userEvent.click(screen.getByRole('button', { name: 'Reconcile' }))
  await userEvent.type(
    screen.getByLabelText('Channel statement reference'),
    'statement-10'
  )
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/guardian/admin/reseller-orders/10/fee',
      { fee_cents: 156, reference: 'statement-10' },
      undefined
    )
  )
  expect(
    await screen.findByRole('dialog', { name: order.service_title })
  ).toBeVisible()
})

it('records a reseller refund inside the order and closes its stale status form', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        order_id: order.id,
        reseller_id: 7,
        trade_no: order.trade_no,
        amount_cents: 26000,
        supply_cents: 20000,
        estimated_fee_cents: 156,
        actual_fee_cents: 156,
        fee_confirmed_at: 1761500200,
        paid_at: order.paid_at,
        refund_state: 'processing',
      },
    },
  })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  const onClose = vi.fn()
  const onSaved = vi.fn()
  await renderDialog(
    <GuardianOrderDialog
      order={{ ...order, reseller_code: 'agent-seven' }}
      onClose={onClose}
      onSaved={onSaved}
    />
  )
  await userEvent.click(
    await screen.findByRole('button', { name: 'Record completed refund' })
  )
  expect(
    screen.getByText('Confirm actual customer refund: ¥258.44')
  ).toBeVisible()
  await userEvent.type(
    screen.getByLabelText('Refund transaction reference'),
    'refund-9'
  )
  await userEvent.type(screen.getByLabelText('Reason'), 'Customer request')
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
  expect(onSaved).toHaveBeenCalledOnce()
  expect(post).toHaveBeenCalledWith(
    '/api/guardian/admin/reseller-orders/9/refund',
    {
      action: 'complete',
      returned_fee_cents: 0,
      reference: 'refund-9',
      reason: 'Customer request',
    },
    undefined
  )
})
