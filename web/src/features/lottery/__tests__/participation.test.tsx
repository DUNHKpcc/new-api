import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { LotteryAdmin } from '../admin'
import { LotteryCard } from '../lottery-card'
import type { Lottery } from '../types'

const lottery: Lottery = {
  id: 1,
  title: 'Test lottery',
  content: 'Rules',
  prize: 'A gift',
  winner_count: 1,
  draw_mode: 'full',
  target_count: 10,
  draw_at: 0,
  require_recharge: false,
  min_invites: 0,
  required_group: '',
  email_notify: false,
  status: 'active',
  entry_count: 2,
  created_at: 1,
  drawn_at: 0,
}
const initial = useAuthStore.getState()
afterEach(() => {
  cleanup()
  useAuthStore.setState(initial, true)
  vi.restoreAllMocks()
})
function setup() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: Infinity },
      mutations: { retry: false },
    },
  })
  useAuthStore.setState({
    auth: {
      ...initial.auth,
      user: { id: 7, username: 'reader', email: 'reader@example.com', role: 1 },
    },
  })
  return client
}

test('joining uses the selected lottery and shows the recorded entry', async () => {
  const client = setup()
  client.setQueryData(['lottery-entry', 1, 7, 'active'], null)
  vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: { id: 3, lottery_id: 1, user_id: 7, winner: false },
    },
  })
  render(
    <QueryClientProvider client={client}>
      <LotteryCard lottery={lottery} />
    </QueryClientProvider>
  )
  fireEvent.click(screen.getByRole('button', { name: 'Enter lottery' }))
  await waitFor(() =>
    expect(api.post).toHaveBeenCalledWith('/api/lotteries/1/join')
  )
  expect(await screen.findByText('Entered successfully')).toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Enter lottery' })
  ).not.toBeInTheDocument()
})

test('expired draws cannot be joined even before the next background draw', () => {
  const client = setup()
  client.setQueryData(['lottery-entry', 1, 7, 'active'], null)
  render(
    <QueryClientProvider client={client}>
      <LotteryCard lottery={{ ...lottery, draw_mode: 'time', draw_at: 1 }} />
    </QueryClientProvider>
  )
  expect(screen.getByRole('button', { name: 'Enter lottery' })).toBeDisabled()
})

test('draft editor uses native rules instead of image uploads and saves a timed draw', async () => {
  const client = setup()
  client.setQueryData(['lotteries-admin', undefined], [])
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: [] } })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: { id: 1 } } })
  const { container } = render(
    <QueryClientProvider client={client}>
      <LotteryAdmin />
    </QueryClientProvider>
  )
  fireEvent.click(screen.getByRole('button', { name: 'Add lottery' }))
  expect(container.querySelector('input[type="file"]')).toBeNull()
  fireEvent.change(screen.getByLabelText('Title'), {
    target: { value: 'Test lottery' },
  })
  fireEvent.change(screen.getByLabelText('Prize'), {
    target: { value: 'Gift' },
  })
  fireEvent.change(screen.getByLabelText('Lottery content'), {
    target: { value: 'Free entry' },
  })
  fireEvent.change(screen.getByLabelText('Draw mode'), {
    target: { value: 'time' },
  })
  expect(screen.queryByLabelText('Participant target')).not.toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Draw time'), {
    target: { value: '2099-01-01T12:00' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/api/lotteries-admin',
      expect.objectContaining({
        title: 'Test lottery',
        draw_mode: 'time',
        draw_at: Math.floor(new Date('2099-01-01T12:00').getTime() / 1000),
      })
    )
  )
})

test('published lottery cannot be edited and cancellation requires confirmation', async () => {
  const client = setup()
  client.setQueryData(['lotteries-admin', undefined], [lottery])
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  render(
    <QueryClientProvider client={client}>
      <LotteryAdmin />
    </QueryClientProvider>
  )
  expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Cancel lottery' }))
  expect(await screen.findByRole('alertdialog')).toBeInTheDocument()
  expect(post).not.toHaveBeenCalled()
})
