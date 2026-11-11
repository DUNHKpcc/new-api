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
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState, type ReactNode } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { GuardianResellerAccessDialog } from '../components/admin/reseller-access-dialog'
import { GuardianResellerList } from '../components/admin/reseller-list'
import { ResellerActionDialog } from '../components/reseller/action-dialog'
import { ResellerOffersPanel } from '../components/reseller/offers'
import { ResellerRecords } from '../components/reseller/records'
import { GuardianResellerCenter } from '../reseller'
import { GuardianServices } from '../services'

const clients: QueryClient[] = []
function renderPanel(node: ReactNode) {
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
  clients.forEach((c) => c.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

it('displays the complete storefront URL and copies the same link', async () => {
  const user = userEvent.setup()
  const copy = vi.spyOn(navigator.clipboard, 'writeText')
  const responses: Record<string, unknown> = {
    '/api/guardian/reseller': {
      user_id: 3,
      status: 'active',
      code: 'DEMO-AGENT',
    },
    '/api/guardian/reseller/balance': {},
    '/api/guardian/reseller/offers': {
      services: [],
      offers: [],
      setting: { fee_bps: 60 },
    },
  }
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: { success: true, data: responses[url] },
  }))
  const router = createRouter({
    routeTree: createRootRoute({ component: GuardianResellerCenter }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  renderPanel(<RouterProvider router={router} />)
  const url = `${window.location.origin}/guardian/services?reseller=DEMO-AGENT`
  expect(await screen.findByRole('link', { name: url })).toHaveAttribute(
    'href',
    url
  )
  await user.click(screen.getByRole('button', { name: 'Copy storefront link' }))
  expect(copy).toHaveBeenCalledWith(url)
})

it('keeps the reseller in desktop and mobile storefront navigation', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: { services: [], pay_methods: [], purchase_enabled: false },
    },
  })
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => <GuardianServices resellerCode='DEMO-AGENT' />,
    }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  renderPanel(<RouterProvider router={router} />)
  const pricing = await screen.findByRole('link', { name: 'Pricing' })
  const mac = screen.getByRole('link', { name: 'Overseas Mac mini' })
  expect(pricing).toHaveAttribute(
    'href',
    expect.stringContaining('reseller=DEMO-AGENT')
  )
  expect(mac).toHaveAttribute(
    'href',
    expect.stringContaining('reseller=DEMO-AGENT')
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Open section menu' })
  )
  const mobileMac = await screen.findByRole('menuitem', {
    name: 'Overseas Mac mini',
  })
  expect(mobileMac).toHaveAttribute(
    'href',
    expect.stringContaining('reseller=DEMO-AGENT')
  )
})

it('refreshes the cached reseller list after granting access', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 10000 } },
  })
  clients.push(client)
  client.setQueryData(['guardian', 'admin', 'resellers'], [])
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/guardian/admin/resellers'
          ? [
              {
                user_id: 3,
                username: 'new-agent',
                display_name: '',
                status: 'active',
              },
            ]
          : null,
    },
  }))
  vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: { user_id: 3, status: 'active' } },
  })
  function AccessThenList() {
    const [saved, setSaved] = useState(false)
    return saved ? (
      <GuardianResellerList />
    ) : (
      <GuardianResellerAccessDialog
        user={{ id: 3, username: 'new-agent' }}
        onClose={() => setSaved(true)}
      />
    )
  }
  render(
    <QueryClientProvider client={client}>
      <AccessThenList />
    </QueryClientProvider>
  )
  await userEvent.type(await screen.findByLabelText('Reason'), 'Approved')
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText('new-agent')).toBeVisible()
})

it('shows prices but prevents editing when reseller sales are suspended', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        services: [{ id: 1, title: 'Mac mini', route: 'mac_mini' }],
        offers: [
          {
            service_id: 1,
            granted: true,
            listed: true,
            supply_cents: 20000,
            retail_cents: 26000,
          },
        ],
        setting: { fee_bps: 60 },
      },
    },
  })
  renderPanel(<ResellerOffersPanel suspended />)
  expect(await screen.findByRole('button', { name: 'Edit' })).toBeDisabled()
  expect(screen.getByText('¥260')).toBeVisible()
})

it('shows an explicit empty state when no products are granted', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { services: [], offers: [], setting: null } },
  })
  renderPanel(<ResellerOffersPanel />)
  expect(await screen.findByText('No authorized products')).toBeVisible()
  expect(
    screen.getByText('The administrator must configure the payment fee first.')
  ).toBeVisible()
})

it('does not submit invalid money and retains the dialog after a failed operation', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockRejectedValue(new Error('Reconciliation rejected'))
  const close = vi.fn()
  renderPanel(
    <ResellerActionDialog
      action={{
        title: 'Reconcile',
        fields: [
          { name: 'fee', label: 'Payment fee', kind: 'money', value: '-1' },
        ],
        submit: (v) => api.put('/fixture', v),
      }}
      onClose={close}
    />
  )
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  expect(await screen.findByText('Enter a valid amount in CNY')).toBeVisible()
  expect(put).not.toHaveBeenCalled()
  await userEvent.clear(screen.getByLabelText('Payment fee'))
  await userEvent.type(screen.getByLabelText('Payment fee'), '1.56')
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith('/fixture', { fee: '1.56' })
  )
  expect(close).not.toHaveBeenCalled()
  expect(screen.getByRole('dialog')).toBeVisible()
})

it('requires a completed-transfer reference and displays the net payout before recording it', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        items: [
          {
            id: 9,
            reseller_id: 3,
            amount_cents: 5000,
            fee_cents: 50,
            status: 'processing',
            account: 'test-payee',
            payee_name: 'Test',
            reference: null,
          },
        ],
        total: 1,
      },
    },
  })
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true, data: null } })
  renderPanel(<ResellerRecords admin withdrawals />)
  await userEvent.click(
    await screen.findByRole('button', { name: 'Mark paid' })
  )
  expect(
    screen.getByText(
      'Record only an actual successful transfer of ¥49.50. This button does not send money.'
    )
  ).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  expect(put).not.toHaveBeenCalled()
  await userEvent.type(
    screen.getByLabelText('Transfer result reference'),
    'external-9'
  )
  await userEvent.type(
    screen.getByLabelText('Reason or verification record'),
    'Recipient and payment checked'
  )
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/guardian/admin/reseller-withdrawals/9',
      expect.objectContaining({
        status: 'paid',
        reference: 'external-9',
        fee_cents: 50,
      }),
      undefined
    )
  )
})

it('does not present unpaid orders as earned income', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        items: [
          {
            order_id: 1,
            trade_no: 'UNPAID-1',
            service_title: 'Mac mini',
            amount_cents: 26000,
            supply_cents: 20000,
            estimated_fee_cents: 156,
            actual_fee_cents: 0,
            fee_confirmed_at: 0,
            paid_at: 0,
            status: 'pending_payment',
            refund_state: '',
          },
        ],
        total: 1,
      },
    },
  })
  renderPanel(<ResellerRecords />)
  expect(await screen.findByRole('cell', { name: '—' })).toBeVisible()
  expect(screen.queryByText('¥58.44')).not.toBeInTheDocument()
})

it('selects a named reseller before editing their product supply prices', async () => {
  const { GuardianResellerAdminPanel } =
    await import('../components/admin/reseller-panel')
  const responses: Record<string, unknown> = {
    '/api/guardian/admin/resellers': [
      {
        user_id: 7,
        username: 'alice-agent',
        display_name: 'Alice',
        status: 'active',
      },
    ],
    '/api/guardian/admin/resellers/7/offers': {
      services: [{ id: 11, title: 'Deployment', route: 'vps' }],
      offers: [],
      setting: { fee_bps: 60 },
    },
    '/api/guardian/admin/reseller-setting': { fee_bps: 60 },
  }
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: { success: true, data: responses[url] ?? { items: [], total: 0 } },
  }))
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  renderPanel(<GuardianResellerAdminPanel />)
  expect(
    screen.queryByRole('button', {
      name: 'Transfer future renewal attribution',
    })
  ).not.toBeInTheDocument()
  await userEvent.type(
    await screen.findByRole('textbox', { name: 'Search resellers' }),
    'alice'
  )
  await userEvent.click(
    await screen.findByRole('button', { name: 'Products and supply prices' })
  )
  expect(await screen.findByText('Deployment')).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
  await userEvent.selectOptions(
    screen.getByLabelText('Authorized for sale'),
    'true'
  )
  await userEvent.clear(screen.getByLabelText('Price (CNY)'))
  await userEvent.type(screen.getByLabelText('Price (CNY)'), '200')
  await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/guardian/admin/resellers/7/offers/11',
      { granted: true, supply_cents: 20000 },
      undefined
    )
  )
})

it('explains how to enable reseller access when the product selector has no resellers', async () => {
  const { GuardianResellerAdminPanel } =
    await import('../components/admin/reseller-panel')
  const responses: Record<string, unknown> = {
    '/api/guardian/admin/resellers': [],
    '/api/guardian/admin/reseller-setting': { fee_bps: 60 },
  }
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: { success: true, data: responses[url] ?? { items: [], total: 0 } },
  }))
  renderPanel(<GuardianResellerAdminPanel />)
  expect(
    await screen.findByText(
      'No resellers yet. Enable Guardian reseller access in User management first.'
    )
  ).toBeVisible()
  expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
})

it('previews the selected reseller tier before applying it', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: [
        {
          user_id: 7,
          username: 'alice-agent',
          display_name: 'Alice',
          status: 'active',
          tier: 0,
          independent_products: 2,
          unavailable_products: 0,
        },
      ],
    },
  })
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      success: true,
      data: {
        token: 'preview-token',
        resellers: 1,
        independent: 2,
        unavailable: 0,
        before: [],
        after: [],
      },
    },
  })
  renderPanel(<GuardianResellerList />)
  await userEvent.click(
    await screen.findByRole('checkbox', { name: 'Select alice-agent' })
  )
  const tiers = screen.getAllByRole('combobox', { name: 'Reseller tier' })
  await userEvent.selectOptions(tiers[1], '2')
  await userEvent.click(screen.getByRole('button', { name: 'Set tier' }))
  expect(await screen.findByRole('alertdialog')).toHaveTextContent(
    'Independent prices and existing orders stay unchanged.'
  )
  expect(post).toHaveBeenCalledTimes(1)
  expect(post.mock.calls[0][1]).toEqual({ tier: 2, reseller_ids: [7] })
  await userEvent.click(screen.getByRole('button', { name: 'Continue' }))
  await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
  expect(post.mock.calls[1][1]).toEqual({
    tier: 2,
    reseller_ids: [7],
    preview_token: 'preview-token',
  })
})

it('edits multiple tier prices and preserves fields when preview fails', async () => {
  const { GuardianTierPrices } = await import('../components/admin/tier-prices')
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: url.endsWith('/services')
        ? [{ id: 1, title: 'Deployment', route: 'vps' }]
        : [],
    },
  }))
  const post = vi.spyOn(api, 'post').mockRejectedValue(new Error('offline'))
  renderPanel(<GuardianTierPrices />)
  const input = await screen.findByLabelText('Single / monthly · CNY')
  await userEvent.type(input, '120.50')
  await userEvent.click(
    screen.getByRole('checkbox', { name: 'Authorized for sale' })
  )
  await userEvent.click(screen.getByRole('button', { name: 'Preview changes' }))
  await waitFor(() => expect(post).toHaveBeenCalledOnce())
  expect(input).toHaveValue('120.50')
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  expect(post.mock.calls[0][1]).toMatchObject({
    tier: 1,
    prices: [{ service_id: 1, granted: true, supply_cents: 12050 }],
  })
})

it('excludes unassigned resellers from the following-tier filter', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: [
        {
          user_id: 7,
          username: 'unassigned-agent',
          status: 'active',
          tier: 0,
          independent_products: 0,
        },
        {
          user_id: 8,
          username: 'tier-agent',
          status: 'active',
          tier: 1,
          independent_products: 0,
        },
      ],
    },
  })
  renderPanel(<GuardianResellerList />)
  await screen.findByRole('checkbox', { name: 'Select unassigned-agent' })
  await userEvent.selectOptions(
    screen.getByRole('combobox', { name: 'Supply source' }),
    'tier'
  )
  expect(
    screen.queryByRole('checkbox', { name: 'Select unassigned-agent' })
  ).not.toBeInTheDocument()
  expect(
    screen.getByRole('checkbox', { name: 'Select tier-agent' })
  ).toBeVisible()
})
