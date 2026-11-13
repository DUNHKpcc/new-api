import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { GuardianAfterSalesTermsPanel } from '../components/admin/after-sales-terms-panel'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

it('preserves unsaved terms during refetch and accepts normalized saved terms', async () => {
  const terms = { deployment: 'Original terms', mac: 'Mac terms' }
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: terms },
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: { ...terms, deployment: 'New terms' } },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <GuardianAfterSalesTermsPanel />
    </QueryClientProvider>
  )
  const input = await screen.findByLabelText('Environment deployment')
  await waitFor(() => expect(input).toHaveValue('Original terms'))
  await userEvent.clear(input)
  await userEvent.type(input, ' New terms ')
  act(() => {
    client.setQueryData(['guardian', 'admin-after-sales-terms'], {
      deployment: 'Background update',
      mac: 'Mac terms',
    })
  })
  expect(input).toHaveValue(' New terms ')
  vi.mocked(api.get).mockResolvedValue({
    data: { success: true, data: { ...terms, deployment: 'New terms' } },
  })
  await userEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(put).toHaveBeenCalledOnce())
  await waitFor(() => expect(input).toHaveValue('New terms'))
  expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  client.clear()
})
