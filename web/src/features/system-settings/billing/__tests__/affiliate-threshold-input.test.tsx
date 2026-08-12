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
import assert from 'node:assert/strict'
import { after, test } from 'node:test'

import { Window } from 'happy-dom'
import type { ReactNode } from 'react'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLInputElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
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

const { act, Component, StrictMode } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { AffiliateSettingsSection } =
  await import('../affiliate-settings-section')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Qualification threshold (minor units)':
          'Qualification threshold (minor units)',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type TestErrorBoundaryState = {
  error: Error | null
}

class TestErrorBoundary extends Component<
  { children: ReactNode },
  TestErrorBoundaryState
> {
  state: TestErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): TestErrorBoundaryState {
    return { error }
  }

  render() {
    if (this.state.error) {
      return <output data-render-error>{this.state.error.message}</output>
    }
    return this.props.children
  }
}

after(() => {
  domWindow.close()
})

test('updates the qualification threshold from mixed-character input in StrictMode', async () => {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })

  await act(async () => {
    root.render(
      <StrictMode>
        <TestErrorBoundary>
          <QueryClientProvider client={queryClient}>
            <I18nextProvider i18n={i18n}>
              <AffiliateSettingsSection defaultValue='{"qualification_threshold_minor":10000}' />
            </I18nextProvider>
          </QueryClientProvider>
        </TestErrorBoundary>
      </StrictMode>
    )
  })

  const thresholdLabel = [...container.querySelectorAll('label')].find(
    (label) =>
      label.textContent?.includes('Qualification threshold (minor units)')
  )
  const thresholdInput =
    thresholdLabel?.querySelector<HTMLInputElement>('input')
  assert.ok(thresholdInput)

  const valueSetter = Object.getOwnPropertyDescriptor(
    domWindow.HTMLInputElement.prototype,
    'value'
  )?.set
  assert.ok(valueSetter)

  await act(async () => {
    valueSetter.call(thresholdInput, '20a000')
    thresholdInput.dispatchEvent(
      new domWindow.Event('input', { bubbles: true }) as unknown as Event
    )
  })

  assert.equal(container.querySelector('[data-render-error]'), null)
  const updatedThresholdInput = container.querySelector<HTMLInputElement>(
    'input[inputmode="numeric"]'
  )
  assert.ok(updatedThresholdInput)
  assert.equal(updatedThresholdInput.value, '20000')

  await act(async () => root.unmount())
  container.remove()
  queryClient.clear()
})
