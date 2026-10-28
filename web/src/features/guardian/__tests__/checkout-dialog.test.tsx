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
import type { ReactNode } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { CheckoutDialog } from '../components/checkout-dialog'
import type { GuardianService } from '../types'

const navigate = vi.fn()
vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  useNavigate: () => navigate,
}))

const service: GuardianService = {
  id: 4,
  title: 'VPS dev machine',
  subtitle: 'Turn a VPS into a Claude Code workstation',
  description: 'First paragraph.\n\nSecond paragraph.',
  features: ['Ubuntu 24.04'],
  price_cents: 59900,
  delivery_note: '1-3 days',
  badge: '',
  route: 'vps',
  enabled: true,
  sort_order: 0,
  created_at: 1,
  updated_at: 1,
}
const payMethods = [
  { type: 'alipay', name: 'Alipay' },
  { type: 'wxpay', name: 'WeChat Pay' },
]

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
  navigate.mockReset()
  vi.restoreAllMocks()
})

it('creates the order with the chosen service and opens the epay form on submit', async () => {
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
  const onClose = vi.fn()
  renderDialog(
    <CheckoutDialog
      service={service}
      payMethods={payMethods}
      onClose={onClose}
    />
  )

  await userEvent.type(screen.getByLabelText('Contact'), '@guardian_user')
  await userEvent.click(screen.getByRole('radio', { name: 'WeChat Pay' }))
  await userEvent.click(screen.getByRole('button', { name: /^Pay / }))

  await waitFor(() => expect(submit).toHaveBeenCalledTimes(1))
  expect(post).toHaveBeenCalledWith('/api/guardian/orders', {
    service_id: 4,
    payment_method: 'wxpay',
    route: 'vps',
    os: 'macos',
    contact_type: 'telegram',
    contact: '@guardian_user',
    note: '',
  })
  const form = submit.mock.contexts[0] as HTMLFormElement
  expect(form.action).toBe('https://pay.example.com/submit.php')
  expect(form.method).toBe('post')
  expect(
    Object.fromEntries(
      new FormData(form) as unknown as Iterable<[string, string]>
    )
  ).toEqual({ pid: '1001', sign: 'signed' })
  expect(onClose).toHaveBeenCalled()
  expect(navigate).toHaveBeenCalledWith({ to: '/guardian/orders' })
})

it('blocks submission and flags the note when it contains a subscription link', async () => {
  const post = vi.spyOn(api, 'post')
  renderDialog(
    <CheckoutDialog
      service={service}
      payMethods={payMethods}
      onClose={() => undefined}
    />
  )

  await userEvent.type(screen.getByLabelText('Contact'), '@guardian_user')
  await userEvent.type(
    screen.getByLabelText('Note (optional)'),
    'my node is vmess://abcdef'
  )
  await userEvent.click(screen.getByRole('button', { name: /^Pay / }))

  expect(
    await screen.findByText(
      'Do not include passwords, keys or subscription links'
    )
  ).toBeVisible()
  expect(screen.getByLabelText('Note (optional)')).toHaveAttribute(
    'aria-invalid',
    'true'
  )
  expect(post).not.toHaveBeenCalled()
})

it('preselects the route the service was built for', () => {
  renderDialog(
    <CheckoutDialog
      service={{ ...service, route: 'vm' }}
      payMethods={payMethods}
      onClose={() => undefined}
    />
  )
  expect(screen.getByLabelText('Route')).toHaveValue('vm')
  expect(screen.getByRole('radio', { name: 'Alipay' })).toBeChecked()
})
