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
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { OrderCard } from '../components/order-card'
import type { GuardianOrder, GuardianOrderStatus } from '../types'

function order(status: GuardianOrderStatus): GuardianOrder {
  return {
    id: 7,
    trade_no: 'SVCUSR1NOabc1761500000',
    service_id: 4,
    service_title: 'VPS dev machine',
    amount_cents: 59900,
    payment_method: 'alipay',
    status,
    route: 'vps',
    os: 'macos',
    contact_type: 'telegram',
    contact: '@guardian_user',
    note: '',
    admin_note: '',
    created_at: 1761500000,
    paid_at: 0,
    delivered_at: 0,
    updated_at: 1761500000,
  }
}

const clients: QueryClient[] = []
function renderCard(status: GuardianOrderStatus) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  return render(
    <QueryClientProvider client={client}>
      <OrderCard order={order(status)} />
    </QueryClientProvider>
  )
}

afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

it('reopens the epay cashier for the same order when continuing payment', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      success: true,
      message: '',
      data: {
        trade_no: 'SVCUSR1NOabc1761500000',
        pay_url: 'https://pay.example.com/submit.php',
        pay_params: { pid: '1001', sign: 'signed' },
      },
    },
  })
  const submit = vi
    .spyOn(HTMLFormElement.prototype, 'submit')
    .mockImplementation(() => undefined)
  renderCard('pending_payment')

  await userEvent.click(
    screen.getByRole('button', { name: 'Continue payment' })
  )

  await waitFor(() => expect(submit).toHaveBeenCalledTimes(1))
  expect(post).toHaveBeenCalledWith('/api/guardian/orders/7/pay')
  const form = submit.mock.contexts[0] as HTMLFormElement
  expect(form.action).toBe('https://pay.example.com/submit.php')
})

it('cancels the order only after the buyer confirms', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, message: '', data: order('cancelled') },
  })
  renderCard('pending_payment')

  await userEvent.click(screen.getByRole('button', { name: 'Cancel order' }))
  await userEvent.click(screen.getByRole('button', { name: 'Keep order' }))
  expect(post).not.toHaveBeenCalled()

  await userEvent.click(screen.getByRole('button', { name: 'Cancel order' }))
  const dialog = await screen.findByRole('alertdialog')
  await userEvent.click(
    within(dialog).getByRole('button', { name: 'Cancel order' })
  )

  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/guardian/orders/7/cancel')
  )
  await waitFor(() =>
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  )
})

it('offers no payment or cancellation once the order is paid', () => {
  renderCard('paid')

  expect(
    screen.queryByRole('button', { name: 'Continue payment' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Cancel order' })
  ).not.toBeInTheDocument()
})

it.each(['delivered', 'refunded', 'expired', 'cancelled'] as const)(
  'hides contact editing for %s orders',
  (status) => {
    renderCard(status)
    expect(
      screen.queryByRole('button', { name: 'Edit contact and note' })
    ).not.toBeInTheDocument()
  }
)

it('edits existing contact details and sends them to the own-order endpoint', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true, data: order('paid') } })
  renderCard('paid')
  await userEvent.click(
    screen.getByRole('button', { name: 'Edit contact and note' })
  )
  expect(screen.getByLabelText('Contact')).toHaveValue('@guardian_user')
  await userEvent.clear(screen.getByLabelText('Contact'))
  await userEvent.type(screen.getByLabelText('Contact'), '@updated')
  await userEvent.type(screen.getByLabelText('Note (optional)'), 'updated note')
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith('/api/guardian/orders/7/contact', {
      contact_type: 'telegram',
      contact: '@updated',
      note: 'updated note',
    })
  )
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
})
