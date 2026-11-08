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

import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it } from 'vitest'

import { MacMiniDelivery } from '../components/mac-mini-delivery'

afterEach(cleanup)

it('explains payment-first delivery and opens refund details by keyboard', async () => {
  const router = createRouter({
    routeTree: createRootRoute({ component: MacMiniDelivery }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(<RouterProvider router={router} />)
  expect(
    await screen.findByRole('heading', { name: 'Choose, then pay' })
  ).toBeVisible()
  expect(
    screen.getByRole('heading', { name: 'Confirm the rental period' })
  ).toBeVisible()
  expect(screen.getByRole('button', { name: 'UU Remote ↗' })).toHaveAttribute(
    'href',
    'https://uuyc.163.com/'
  )
  expect(screen.getByRole('button', { name: 'My orders' })).toHaveAttribute(
    'href',
    '/guardian/orders'
  )
  const refund = screen.getByRole('button', { name: 'Refund assessment' })
  expect(refund).toHaveAttribute('aria-expanded', 'false')
  refund.focus()
  await userEvent.keyboard('{Enter}')
  expect(refund).toHaveAttribute('aria-expanded', 'true')
  expect(screen.getByText(/For an eligible refund/)).toBeVisible()
  await userEvent.keyboard('{Enter}')
  expect(refund).toHaveAttribute('aria-expanded', 'false')
})
