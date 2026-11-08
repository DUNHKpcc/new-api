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
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { ThemeProvider } from '@/context/theme-provider'
import { api } from '@/lib/api'
import { Route as legacyStatisticsRoute } from '@/routes/guardian/account-statistics'
import { Route as servicesRoute } from '@/routes/guardian/services'

import { GuardianAccountStatusDialog } from '../components/admin/account-status-dialog'
import { GuardianHistoricalAccountsForm } from '../components/admin/historical-accounts-panel'
import { GuardianServices } from '../services'
import type { GuardianAccountStatistics } from '../types'

const clients: QueryClient[] = []
afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

function renderStatistics(data: GuardianAccountStatistics) {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: url === '/api/guardian/account-statistics' ? data : [],
    },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/guardian/services',
    validateSearch: servicesRoute.options.validateSearch,
    component: function ServicesTestRoute(): ReactElement {
      const search = route.useSearch()
      return <GuardianServices category={search.category} />
    },
  })
  const legacyRoute = createRoute({
    getParentRoute: () => root,
    path: '/guardian/account-statistics',
    beforeLoad: legacyStatisticsRoute.options.beforeLoad as () => never,
  })
  const router = createRouter({
    routeTree: root.addChildren([route, legacyRoute]),
    history: createMemoryHistory({
      initialEntries: ['/guardian/account-statistics'],
    }),
  })
  render(
    <QueryClientProvider client={client}>
      <ThemeProvider>
        <RouterProvider router={router} />
      </ThemeProvider>
    </QueryClientProvider>
  )
  return client
}

const empty: GuardianAccountStatistics = {
  historical_users: 0,
  healthy_users: 0,
  all_issues_users: 0,
  purchased_users: 5,
  recorded_users: 0,
  banned_users: 0,
  phone_required_users: 0,
  kyc_required_users: 0,
  updated_at: 0,
}

it('distinguishes unrecorded purchasers from confirmed zero issues', async () => {
  renderStatistics(empty)
  expect(await screen.findByText('Not recorded: 5')).toBeVisible()
  expect(within(screen.getByText('Purchasers:')).getByText('5')).toHaveClass(
    'font-semibold',
    'text-[var(--g-accent)]'
  )
  expect(screen.getByRole('link', { name: /^Claude-Guardian$/ })).toHaveClass(
    'bg-[var(--g-accent-wash)]',
    'text-[var(--g-ink)]'
  )
  expect(screen.getByText('No account statuses recorded yet')).toBeVisible()
  expect(screen.getAllByText('—')).toHaveLength(3)
})

it('redirects the old URL into the pricing tabs and switches statistics in place', async () => {
  renderStatistics(empty)
  const statusTab = await screen.findByRole('tab', {
    name: 'Purchased user account status',
  })
  expect(statusTab).toHaveAttribute('aria-selected', 'true')
  expect(
    screen.queryByRole('button', {
      name: 'Click to view purchased user account status',
    })
  ).not.toBeInTheDocument()
  await screen.findByRole('region', { name: 'Account status matrix' })
  await userEvent.click(screen.getByRole('tab', { name: 'Environment setup' }))
  expect(
    screen.queryByRole('region', { name: 'Account status matrix' })
  ).not.toBeInTheDocument()
  expect(
    screen.getByRole('tab', { name: 'Environment setup' })
  ).toHaveAttribute('aria-selected', 'true')
  await userEvent.click(statusTab)
  expect(
    await screen.findByRole('region', { name: 'Account status matrix' })
  ).toBeVisible()
  expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
})

it('shows independent counts and updates when fresh statistics arrive', async () => {
  const client = renderStatistics({
    ...empty,
    recorded_users: 2,
    banned_users: 1,
    phone_required_users: 2,
    updated_at: 1700000000,
  })
  const label = await screen.findByText('Account banned')
  expect(
    within(label.parentElement as HTMLElement).getByText('1')
  ).toBeVisible()
  expect(
    within(
      screen.getByText('Phone verification required')
        .parentElement as HTMLElement
    ).getByText('2')
  ).toBeVisible()
  expect(
    within(
      screen.getByText('KYC required').parentElement as HTMLElement
    ).getByText('0')
  ).toBeVisible()
  client.setQueryData(['guardian', 'account-statistics'], {
    ...empty,
    recorded_users: 2,
    updated_at: 1700000010,
  })
  await waitFor(() =>
    expect(
      within(label.parentElement as HTMLElement).getByText('0')
    ).toBeVisible()
  )
})

it('saves all three conditions independently and allows clearing recovered conditions', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true, data: {} } })
  const client = new QueryClient()
  clients.push(client)
  const saved = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <GuardianAccountStatusDialog
        user={{
          user_id: 7,
          username: 'buyer',
          banned: true,
          phone_required: false,
          kyc_required: false,
          updated_at: 1700000000,
        }}
        onClose={() => undefined}
        onSaved={saved}
      />
    </QueryClientProvider>
  )
  await userEvent.click(screen.getByRole('switch', { name: 'Account banned' }))
  await userEvent.click(
    screen.getByRole('switch', { name: 'Phone verification required' })
  )
  await userEvent.click(screen.getByRole('switch', { name: 'KYC required' }))
  await userEvent.selectOptions(
    screen.getByLabelText('Historical user link'),
    'banned_only'
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(saved).toHaveBeenCalledOnce())
  expect(put).toHaveBeenCalledWith('/api/guardian/admin/account-statuses/7', {
    banned: false,
    phone_required: true,
    kyc_required: true,
    historical_category: 'banned_only',
  })
})

it('includes historical users in status groups without using sales samples', async () => {
  renderStatistics({
    ...empty,
    purchased_users: 110,
    historical_users: 105,
    recorded_users: 109,
    healthy_users: 105,
    all_issues_users: 1,
  })
  expect(await screen.findByText('Historical users: 105')).toBeVisible()
  expect(screen.getByText('(+5)')).toBeVisible()
  const partial = screen.getByText('One or two conditions flagged')
  expect(
    within(partial.parentElement as HTMLElement).getByText('3')
  ).toBeVisible()
  const healthy = screen.getByText('No ban, no KYC, no phone verification')
  expect(
    within(healthy.parentElement as HTMLElement).getByText('105')
  ).toBeVisible()
})

it('renders a continuous seven-row matrix with scrollable overflow and separate counts', async () => {
  renderStatistics({ ...empty, historical_users: 2 })
  const matrix = await screen.findByRole('region', {
    name: 'Account status matrix',
  })
  expect(matrix).toHaveAttribute('tabindex', '0')
  expect(matrix).toHaveClass('overflow-x-auto', 'p-1')
  expect(matrix.firstElementChild).toHaveClass('grid-rows-7', 'grid-flow-col')
  expect(matrix.querySelectorAll('span')).toHaveLength(280)
  expect(
    screen.getByText(
      /Pale empty squares are placeholders, not account records\./
    )
  ).toBeVisible()
})

it('shows the matching status on hover and keyboard focus, including placeholders', async () => {
  renderStatistics({
    ...empty,
    purchased_users: 4,
    recorded_users: 3,
    healthy_users: 1,
    all_issues_users: 1,
    historical_users: 1,
  })
  const matrix = await screen.findByRole('region', {
    name: 'Account status matrix',
  })
  const user = userEvent.setup()
  for (const label of [
    'No ban, no KYC, no phone verification',
    'One or two conditions flagged',
    'All three conditions flagged',
    'Not recorded',
  ]) {
    const cell = within(matrix).getByRole('img', { name: label })
    await user.hover(cell)
    expect(await screen.findByRole('tooltip')).toHaveTextContent(label)
    await user.unhover(cell)
    await waitFor(() =>
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    )
  }
  const placeholder = within(matrix).getAllByRole('img', {
    name: 'Pale empty squares are placeholders, not account records.',
  })[0]
  await user.hover(placeholder)
  expect(await screen.findByRole('tooltip')).toHaveTextContent(
    'Pale empty squares are placeholders, not account records.'
  )
  await user.unhover(placeholder)
  await waitFor(() =>
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  )
  within(matrix)
    .getByRole('img', { name: 'No ban, no KYC, no phone verification' })
    .focus()
  expect(await screen.findByRole('tooltip')).toHaveTextContent(
    'No ban, no KYC, no phone verification'
  )
  await user.keyboard('{Escape}')
  await waitFor(() =>
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  )
})

it('saves eight historical counts, rejects overlapping totals and allows resetting to zero', async () => {
  const initial = {
    total_users: 0,
    banned_only: 0,
    kyc_only: 0,
    phone_only: 0,
    banned_kyc: 0,
    banned_phone: 0,
    kyc_phone: 0,
    all_three: 0,
  }
  const put = vi.spyOn(api, 'put').mockImplementation(async (_url, data) => ({
    data: { success: true, data },
  }))
  const client = new QueryClient()
  clients.push(client)
  render(
    <QueryClientProvider client={client}>
      <GuardianHistoricalAccountsForm initial={initial} baseSales={10} />
    </QueryClientProvider>
  )
  const user = userEvent.setup()
  expect(screen.getAllByRole('spinbutton')).toHaveLength(8)
  for (const field of screen.getAllByRole('spinbutton')) {
    expect(field).toHaveValue(0)
  }
  await user.clear(screen.getByLabelText('Ban only'))
  await user.type(screen.getByLabelText('Ban only'), '1')
  await user.click(screen.getByRole('button', { name: 'Save' }))
  expect(
    await screen.findByText(
      'Combined issue counts cannot exceed historical users'
    )
  ).toBeVisible()
  expect(put).not.toHaveBeenCalled()
  await user.clear(screen.getByLabelText('Historical base users'))
  await user.type(screen.getByLabelText('Historical base users'), '11')
  await user.click(screen.getByRole('button', { name: 'Save' }))
  expect(
    await screen.findByText('Historical users cannot exceed base sales')
  ).toBeVisible()
  expect(put).not.toHaveBeenCalled()
  await user.clear(screen.getByLabelText('Historical base users'))
  await user.type(screen.getByLabelText('Historical base users'), '10')
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/guardian/admin/historical-accounts',
      { ...initial, total_users: 10, banned_only: 1 }
    )
  )
  for (const label of ['Historical base users', 'Ban only']) {
    await user.clear(screen.getByLabelText(label))
    await user.type(screen.getByLabelText(label), '0')
  }
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(put).toHaveBeenLastCalledWith(
      '/api/guardian/admin/historical-accounts',
      initial
    )
  )
})
