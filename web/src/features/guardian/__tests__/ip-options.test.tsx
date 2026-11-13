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

import { GuardianServiceDialog } from '../components/admin/service-dialog'
import type { GuardianService } from '../types'

const product: GuardianService = {
  id: 9,
  title: 'Environment setup',
  subtitle: '',
  description: '',
  features: [],
  price_cents: 19900,
  delivery_note: '',
  badge: '',
  route: 'vps',
  enabled: true,
  sort_order: 0,
  created_at: 1,
  updated_at: 1,
}
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

it('lets administrators add, edit, disable and remove IP options', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true, data: product } })
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })
  const saved = vi.fn()
  const view = render(
    <QueryClientProvider client={client}>
      <GuardianServiceDialog
        service={product}
        onClose={() => undefined}
        onSaved={saved}
      />
    </QueryClientProvider>
  )
  await userEvent.click(screen.getByRole('button', { name: 'Add IP option' }))
  await userEvent.type(
    screen.getByLabelText('Name', { exact: true }),
    'Residential IP'
  )
  await userEvent.type(
    screen.getAllByLabelText('Description', { exact: true })[0],
    'One month'
  )
  await userEvent.type(
    within(screen.getByRole('group', { name: 'IP add-ons' })).getByLabelText(
      'Monthly price (CNY)',
      { exact: true }
    ),
    '50.25'
  )
  await userEvent.type(screen.getByLabelText('Quarterly price (CNY)'), '140')
  await userEvent.type(screen.getByLabelText('Annual price (CNY)'), '500')
  await userEvent.click(
    screen.getByRole('switch', { name: 'Require IP for this product' })
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(saved).toHaveBeenCalledOnce())
  const payload = put.mock.calls[0][1] as GuardianService
  expect(payload.ip_enabled).toBe(true)
  expect(payload.ip_options).toEqual([
    {
      id: expect.any(String),
      name: 'Residential IP',
      description: 'One month',
      price_cents: 5025,
      quarter_price_cents: 14000,
      annual_price_cents: 50000,
      enabled: true,
    },
  ])
  view.unmount()
  render(
    <QueryClientProvider client={client}>
      <GuardianServiceDialog
        service={{ ...product, ip_options: payload.ip_options }}
        onClose={() => undefined}
        onSaved={saved}
      />
    </QueryClientProvider>
  )
  await userEvent.click(
    screen.getByRole('switch', { name: 'List this IP option' })
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(put).toHaveBeenCalledTimes(2))
  expect((put.mock.calls[1][1] as GuardianService).ip_options?.[0]).toEqual({
    ...payload.ip_options?.[0],
    enabled: false,
  })
  await userEvent.click(screen.getByRole('button', { name: 'Remove' }))
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(put).toHaveBeenCalledTimes(3))
  expect((put.mock.calls[2][1] as GuardianService).ip_options).toEqual([])
  client.clear()
})
