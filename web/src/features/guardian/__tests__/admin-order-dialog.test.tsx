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
function renderDialog(node: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  return render(
    <QueryClientProvider client={client}>{node}</QueryClientProvider>
  )
}

afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

function statusOptions() {
  return within(screen.getByLabelText('Status'))
    .getAllByRole('option')
    .map((option) => (option as HTMLOptionElement).value)
}

it('offers only the transitions the backend allows from a paid order', () => {
  renderDialog(
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

it('locks the status of a closed order', () => {
  renderDialog(
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
  renderDialog(
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
