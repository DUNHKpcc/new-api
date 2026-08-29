/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { after, beforeEach, describe, test } from 'node:test'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'Node',
  'Element',
  'Event',
  'KeyboardEvent',
  'PointerEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const i18next = (await import('i18next')).default
const { initReactI18next, I18nextProvider } = await import('react-i18next')
const { useAuthStore } = await import('@/stores/auth-store')
const { PccAgentTicket } = await import('../pcc-agent-ticket')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

await i18next.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: false,
  resources: {
    en: {
      translation: {
        'Limited offer': 'Limited offer',
        'PccAgent $150 credit offer': 'PccAgent $150 credit offer',
        'PccAgent credit claimed': 'PccAgent credit claimed',
        'PccAgent credit claim complete': 'PccAgent credit claim complete',
        'Official PccAgent offer': 'Official PccAgent offer',
        'AI API credits': 'AI API credits',
        'Claim {{amount}} in AI API credits':
          'Claim {{amount}} in AI API credits',
        'Use PccAgent to unlock your credit':
          'Use PccAgent to unlock your credit',
        'One-time claim': 'One-time claim',
        'Tear to claim': 'Tear to claim',
        'Drag right to tear and claim': 'Drag right to tear and claim',
        'Claimed successfully': 'Claimed successfully',
        'Close PccAgent offer': 'Close PccAgent offer',
      },
    },
  },
})

type RenderedTicket = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
}

async function renderTicket(): Promise<RenderedTicket> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  const queryClient = new QueryClient()
  queryClient.setQueryData(['status'], {
    global_notifications_enabled: true,
    global_notifications: [
      {
        id: 150,
        kind: 'pcc-agent-ticket',
        title: 'PccAgent $150 credit offer',
        content: 'Use PccAgent to unlock your credit',
        publishDate: '2026-08-28T00:00:00Z',
        type: 'success',
      },
    ],
  })

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18next}>
          <PccAgentTicket />
        </I18nextProvider>
      </QueryClientProvider>
    )
  })

  return { container, root }
}

async function unmountTicket(rendered: RenderedTicket) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

function resetAuth() {
  useAuthStore.getState().auth.reset('complete')
  useAuthStore.getState().auth.setUser({
    id: 42,
    username: 'ticket-tester',
    role: 1,
  })
}

describe('PccAgent ticket interaction', () => {
  beforeEach(() => {
    domWindow.localStorage.clear()
    document.body.replaceChildren()
    resetAuth()
  })

  after(() => {
    domWindow.close()
  })

  test('claims the credit after dragging the ticket tail past the threshold', async () => {
    const rendered = await renderTicket()
    const stub = rendered.container.querySelector<HTMLButtonElement>(
      '.pcc-agent-ticket__stub'
    )

    assert.ok(stub)
    await act(async () => {
      stub.dispatchEvent(
        new PointerEvent('pointerdown', {
          bubbles: true,
          button: 0,
          clientX: 100,
          pointerId: 7,
        })
      )
      stub.dispatchEvent(
        new PointerEvent('pointermove', {
          bubbles: true,
          clientX: 240,
          pointerId: 7,
        })
      )
      stub.dispatchEvent(
        new PointerEvent('pointerup', {
          bubbles: true,
          clientX: 240,
          pointerId: 7,
        })
      )
    })

    const ticket = rendered.container.querySelector('.pcc-agent-ticket')
    assert.equal(ticket?.getAttribute('data-state'), 'claimed')
    assert.equal(
      rendered.container
        .querySelector('.pcc-agent-ticket__claim-panel')
        ?.getAttribute('aria-hidden'),
      'false'
    )

    await unmountTicket(rendered)
  })

  test('snaps the ticket tail back when the drag is released early', async () => {
    const rendered = await renderTicket()
    const stub = rendered.container.querySelector<HTMLButtonElement>(
      '.pcc-agent-ticket__stub'
    )

    assert.ok(stub)
    await act(async () => {
      stub.dispatchEvent(
        new PointerEvent('pointerdown', {
          bubbles: true,
          button: 0,
          clientX: 100,
          pointerId: 8,
        })
      )
      stub.dispatchEvent(
        new PointerEvent('pointermove', {
          bubbles: true,
          clientX: 130,
          pointerId: 8,
        })
      )
      stub.dispatchEvent(
        new PointerEvent('pointerup', {
          bubbles: true,
          clientX: 130,
          pointerId: 8,
        })
      )
    })

    const ticket = rendered.container.querySelector('.pcc-agent-ticket')
    assert.equal(ticket?.getAttribute('data-state'), 'ready')
    assert.equal(
      rendered.container
        .querySelector('.pcc-agent-ticket__claim-panel')
        ?.getAttribute('aria-hidden'),
      'true'
    )

    await unmountTicket(rendered)
  })

  test('supports keyboard claiming and suppresses the ticket after remount', async () => {
    const first = await renderTicket()
    const stub = first.container.querySelector<HTMLButtonElement>(
      '.pcc-agent-ticket__stub'
    )

    assert.ok(stub)
    await act(async () => {
      stub.dispatchEvent(
        new KeyboardEvent('keydown', {
          bubbles: true,
          key: 'Enter',
        })
      )
    })
    assert.equal(
      first.container
        .querySelector('.pcc-agent-ticket')
        ?.getAttribute('data-state'),
      'claimed'
    )
    await unmountTicket(first)

    const second = await renderTicket()
    assert.equal(second.container.querySelector('.pcc-agent-ticket'), null)
    await unmountTicket(second)
  })

  test('does not render again after an anonymous visitor signs in', async () => {
    useAuthStore.getState().auth.reset('complete')
    const visitor = await renderTicket()
    assert.ok(visitor.container.querySelector('.pcc-agent-ticket'))
    await unmountTicket(visitor)

    resetAuth()
    const authenticated = await renderTicket()
    assert.equal(
      authenticated.container.querySelector('.pcc-agent-ticket'),
      null
    )
    await unmountTicket(authenticated)
  })
})
