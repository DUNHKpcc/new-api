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
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, expect, it, vi } from 'vitest'

import zh from '@/i18n/locales/zh.json'
import { api } from '@/lib/api'

import { EmailDeliveryDialog } from '../components/delivery-dialog'
import { EmailDraftDialog } from '../components/draft-dialog'
import { EmailQueueDialog } from '../components/queue-dialog'
import type { EmailCampaign } from '../types'

const campaign: EmailCampaign = {
  id: 7,
  subject: 'Maintenance notice',
  body: 'First line\n<script>literal</script>',
  category: 'platform',
  group: '',
  status: 'draft',
  created_by: 1,
  created_at: 100,
  updated_at: 100,
  delivery_counts: {
    total: 0,
    pending: 0,
    sending: 0,
    retry: 0,
    sent: 0,
    failed: 0,
    skipped: 0,
    uncertain: 0,
    cancelled: 0,
  },
}
const clients: QueryClient[] = []
function renderWorkflow(component: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(client)
  return render(
    <QueryClientProvider client={client}>{component}</QueryClientProvider>
  )
}
afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

it('rejects empty drafts and multibyte bodies over the byte limit without submitting', async () => {
  const post = vi.spyOn(api, 'post')
  renderWorkflow(
    <EmailDraftDialog onClose={() => undefined} onSaved={() => undefined} />
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save draft' }))
  expect(await screen.findByText('Enter an email subject')).toBeVisible()
  expect(screen.getByLabelText('Email subject')).toHaveAttribute(
    'aria-invalid',
    'true'
  )
  fireEvent.change(screen.getByLabelText('Email subject'), {
    target: { value: 'Offer' },
  })
  fireEvent.change(screen.getByLabelText('Email message'), {
    target: { value: '中'.repeat(6667) },
  })
  await userEvent.click(screen.getByRole('button', { name: 'Save draft' }))
  expect(
    await screen.findByText('Email message must be at most 20000 UTF-8 bytes')
  ).toBeVisible()
  expect(post).not.toHaveBeenCalled()
})

it('saves a valid draft without sending and keeps a rejected draft open', async () => {
  const onSaved = vi.fn()
  const onClose = vi.fn()
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValueOnce({
      data: { success: false, message: 'SMTP permission denied' },
    })
    .mockResolvedValueOnce({ data: { success: true, data: campaign } })
  renderWorkflow(<EmailDraftDialog onClose={onClose} onSaved={onSaved} />)
  fireEvent.change(screen.getByLabelText('Email subject'), {
    target: { value: '  Maintenance notice  ' },
  })
  fireEvent.change(screen.getByLabelText('Email message'), {
    target: { value: 'Planned maintenance.' },
  })
  await userEvent.selectOptions(screen.getByLabelText('Category'), 'platform')
  await userEvent.click(screen.getByRole('button', { name: 'Save draft' }))
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Failed to save email draft'
  )
  expect(onSaved).not.toHaveBeenCalled()
  expect(onClose).not.toHaveBeenCalled()
  await userEvent.click(screen.getByRole('button', { name: 'Save draft' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledOnce())
  expect(post).toHaveBeenLastCalledWith('/api/email-campaign/', {
    subject: 'Maintenance notice',
    body: 'Planned maintenance.',
    category: 'platform',
    group: '',
  })
})

it('previews escaped content and recipient count before explicit queue confirmation', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { eligible: 2 } },
  })
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { ...campaign, status: 'queued' } },
  })
  const onQueued = vi.fn()
  renderWorkflow(
    <EmailQueueDialog
      campaign={campaign}
      onClose={() => undefined}
      onQueued={onQueued}
      sendingReady
    />
  )
  expect(await screen.findByText('Eligible recipients: 2')).toBeVisible()
  expect(screen.getByText(/<script>literal<\/script>/)).toBeVisible()
  expect(document.querySelector('script')).toBeNull()
  expect(post).not.toHaveBeenCalled()
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm and queue' })
  )
  await waitFor(() => expect(onQueued).toHaveBeenCalledOnce())
  expect(post).toHaveBeenCalledWith('/api/email-campaign/7/queue')
})

it('blocks queueing when recipient preview fails', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: false, message: 'Preview unavailable' },
  })
  const post = vi.spyOn(api, 'post')
  renderWorkflow(
    <EmailQueueDialog
      campaign={campaign}
      onClose={() => undefined}
      onQueued={() => undefined}
      sendingReady
    />
  )
  expect(await screen.findByText('Failed to preview recipients')).toBeVisible()
  expect(
    screen.getByRole('button', { name: 'Confirm and queue' })
  ).toBeDisabled()
  expect(post).not.toHaveBeenCalled()
})

it('blocks empty audiences and missing SMTP configuration', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { eligible: 0 } },
  })
  renderWorkflow(
    <EmailQueueDialog
      campaign={campaign}
      onClose={() => undefined}
      onQueued={() => undefined}
      sendingReady={false}
    />
  )
  await screen.findByText('Eligible recipients: 0')
  expect(
    screen.getByRole('button', { name: 'Confirm and queue' })
  ).toBeDisabled()
  expect(screen.getByRole('alert')).toHaveTextContent('Configure SMTP')
})

it('paginates delivery records on the server and resets the page when status changes', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        items: [
          {
            id: 1,
            campaign_id: 7,
            user_id: 2,
            email: 'person@example.com',
            status: 'uncertain',
            attempts: 1,
            next_attempt_at: 0,
            last_error: 'Connection interrupted',
            sent_at: 0,
            created_at: 100,
            updated_at: 100,
          },
        ],
        total: 21,
        p: 1,
        page_size: 20,
      },
    },
  })
  renderWorkflow(
    <EmailDeliveryDialog campaign={campaign} onClose={() => undefined} />
  )
  expect(await screen.findByText('person@example.com')).toBeVisible()
  expect(screen.getByText('Connection interrupted')).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: 'Go to next page' }))
  await waitFor(() =>
    expect(get).toHaveBeenLastCalledWith('/api/email-campaign/7/deliveries', {
      params: { p: 2, page_size: 20, status: '' },
    })
  )
  await userEvent.selectOptions(
    screen.getByRole('combobox', { name: 'Delivery status' }),
    'uncertain'
  )
  await waitFor(() =>
    expect(get).toHaveBeenLastCalledWith('/api/email-campaign/7/deliveries', {
      params: { p: 1, page_size: 20, status: 'uncertain' },
    })
  )
  expect(
    screen.getByText(
      'Accepted by SMTP does not guarantee inbox delivery. Uncertain deliveries are not retried automatically.'
    )
  ).toBeVisible()
})

it('labels pending delivery as awaiting send in Chinese without reusing approval wording', async () => {
  const i18n = createInstance()
  await i18n.init({ lng: 'zh', resources: { zh } })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        items: [
          {
            id: 1,
            campaign_id: 7,
            user_id: 2,
            email: 'waiting@example.com',
            status: 'pending',
            attempts: 0,
            next_attempt_at: 0,
            last_error: '',
            sent_at: 0,
            created_at: 100,
            updated_at: 100,
          },
        ],
        total: 1,
        p: 1,
        page_size: 20,
      },
    },
  })
  renderWorkflow(
    <I18nextProvider i18n={i18n}>
      <EmailDeliveryDialog campaign={campaign} onClose={() => undefined} />
    </I18nextProvider>
  )
  expect(
    await screen.findByRole('row', { name: /waiting@example.com/ })
  ).toHaveTextContent('待发送')
  expect(screen.getByRole('option', { name: '待发送' })).toHaveValue('pending')
  expect(i18n.t('Pending')).toBe('待确认')
})
