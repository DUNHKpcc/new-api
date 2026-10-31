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

import { GuardianQueuePanel } from '../components/admin/queue-panel'
import type {
  AdminGuardianOrder,
  AdminGuardianQueue,
  AdminGuardianQueueItem,
} from '../types'

function adminOrder(id: number, username: string): AdminGuardianOrder {
  return {
    id,
    trade_no: `SVCUSR${id}NOabc1761500000`,
    service_id: 4,
    service_title: 'VPS dev machine',
    amount_cents: 59900,
    payment_method: 'alipay',
    status: 'paid',
    route: 'vps',
    os: 'macos',
    contact_type: 'telegram',
    contact: '@guardian_user',
    note: '',
    admin_note: '',
    created_at: 1761500000,
    paid_at: 1761500100,
    delivered_at: 0,
    updated_at: 1761500100,
    user_id: id,
    username,
  }
}

function waiting(
  id: number,
  username: string,
  ahead: number
): AdminGuardianQueueItem {
  return {
    ...adminOrder(id, username),
    queue_ahead: ahead,
    queue_start_date: '',
  }
}

const queue: AdminGuardianQueue = {
  setting: { daily_capacity: 0, extra_slots: 0, paused: false, resume_at: 0 },
  paused: false,
  started_today: 1,
  in_progress: [{ ...adminOrder(9, 'busy'), status: 'in_progress' }],
  waiting: [waiting(11, 'alice', 1), waiting(12, 'bob', 2)],
}

const clients: QueryClient[] = []
function renderPanel() {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, message: '', data: queue },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  return render(
    <QueryClientProvider client={client}>
      <GuardianQueuePanel />
    </QueryClientProvider>
  )
}

afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

it('lists orders in progress first, then waiting orders with what each buyer sees', async () => {
  renderPanel()

  const rows = await screen.findAllByRole('row')
  expect(within(rows[1]).getByText('busy')).toBeInTheDocument()
  expect(within(rows[2]).getByText('alice')).toBeInTheDocument()
  expect(within(rows[2]).getByText('1 ahead')).toBeInTheDocument()
  expect(within(rows[3]).getByText('bob')).toBeInTheDocument()
})

it('moves a waiting order up one place', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, message: '', data: null },
  })
  renderPanel()

  const bobRow = await screen.findByRole('row', { name: /bob/ })
  await userEvent.click(within(bobRow).getByRole('button', { name: 'Move up' }))

  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/guardian/admin/queue/12/move', {
      position: 1,
    })
  )
})

it('moves a waiting order to a typed position', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, message: '', data: null },
  })
  renderPanel()

  const aliceRow = await screen.findByRole('row', { name: /alice/ })
  await userEvent.type(
    within(aliceRow).getByRole('spinbutton', { name: 'Move to position' }),
    '2'
  )
  await userEvent.click(within(aliceRow).getByRole('button', { name: 'Move' }))

  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/guardian/admin/queue/11/move', {
      position: 2,
    })
  )
})

it('saves the daily limit and extra slots without a resume time when not paused', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: {
      success: true,
      message: '',
      data: { daily_capacity: 2, extra_slots: 1, paused: false, resume_at: 0 },
    },
  })
  renderPanel()

  const capacity = await screen.findByLabelText('Daily start limit')
  await userEvent.clear(capacity)
  await userEvent.type(capacity, '2')
  const extra = screen.getByLabelText('Extra queue slots')
  await userEvent.clear(extra)
  await userEvent.type(extra, '1')
  expect(screen.queryByText('Resume at')).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))

  await waitFor(() =>
    expect(put).toHaveBeenCalledWith('/api/guardian/admin/queue/settings', {
      daily_capacity: 2,
      extra_slots: 1,
      paused: false,
      resume_at: 0,
    })
  )
})

it('pauses scheduling until resumed manually when no resume time is chosen', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: {
      success: true,
      message: '',
      data: { daily_capacity: 0, extra_slots: 0, paused: true, resume_at: 0 },
    },
  })
  renderPanel()

  await userEvent.click(
    await screen.findByRole('switch', { name: 'Pause scheduling' })
  )
  expect(screen.getByText('Resume at')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))

  await waitFor(() =>
    expect(put).toHaveBeenCalledWith('/api/guardian/admin/queue/settings', {
      daily_capacity: 0,
      extra_slots: 0,
      paused: true,
      resume_at: 0,
    })
  )
})
