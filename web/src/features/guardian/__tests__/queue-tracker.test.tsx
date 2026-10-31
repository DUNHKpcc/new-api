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
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'

import { QueueTracker } from '../components/queue-tracker'
import type { GuardianOrder, GuardianOrderStatus } from '../types'

function order(
  status: GuardianOrderStatus,
  queue: Pick<GuardianOrder, 'queue_ahead' | 'queue_start_date'> = {}
): GuardianOrder {
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
    paid_at: 1761500100,
    delivered_at: 0,
    updated_at: 1761500100,
    ...queue,
  }
}

const running = { paused: false, resume_at: 0 }

afterEach(() => cleanup())

it('shows how many orders are ahead and the estimated start day', () => {
  render(
    <QueueTracker
      order={order('paid', { queue_ahead: 3, queue_start_date: '2026-10-31' })}
      queue={running}
    />
  )

  expect(screen.getByRole('status')).toHaveTextContent('3ahead of you')
  expect(screen.getByText('Estimated start October 31')).toBeInTheDocument()
})

it('updates the count when an order ahead finishes', () => {
  const view = render(
    <QueueTracker order={order('paid', { queue_ahead: 3 })} queue={running} />
  )

  view.rerender(
    <QueueTracker order={order('paid', { queue_ahead: 2 })} queue={running} />
  )

  expect(screen.getByRole('status')).toHaveTextContent('2ahead of you')
})

it('collapses a long queue into a count beyond twelve squares', () => {
  render(
    <QueueTracker order={order('paid', { queue_ahead: 15 })} queue={running} />
  )

  expect(screen.getByText('+3')).toBeInTheDocument()
})

it('tells the buyer they are next when nobody is ahead', () => {
  render(
    <QueueTracker
      order={order('scheduled', { queue_ahead: 0 })}
      queue={running}
    />
  )

  expect(screen.getByRole('status')).toHaveTextContent("You're next")
})

it('explains a pause with and without a resume time', () => {
  const resumeAt = new Date(2026, 10, 2, 9, 30).getTime() / 1000
  const view = render(
    <QueueTracker
      order={order('paid', { queue_ahead: 2, queue_start_date: '2026-11-02' })}
      queue={{ paused: true, resume_at: resumeAt }}
    />
  )
  expect(
    screen.getByText(/^Scheduling is paused\. Expected to resume November 2/)
  ).toBeInTheDocument()
  expect(screen.queryByText(/Estimated start/)).not.toBeInTheDocument()

  view.rerender(
    <QueueTracker
      order={order('paid', { queue_ahead: 2 })}
      queue={{ paused: true, resume_at: 0 }}
    />
  )
  expect(screen.getByText('Scheduling is paused for now')).toBeInTheDocument()
})

it('switches to the in-progress message once delivery starts', () => {
  render(<QueueTracker order={order('in_progress')} queue={running} />)

  expect(screen.getByRole('status')).toHaveTextContent(
    'Your setup is in progress'
  )
})

it('renders nothing for orders that are not in the queue', () => {
  render(<QueueTracker order={order('pending_payment')} queue={running} />)

  expect(
    screen.queryByRole('region', { name: 'Delivery queue' })
  ).not.toBeInTheDocument()
})
