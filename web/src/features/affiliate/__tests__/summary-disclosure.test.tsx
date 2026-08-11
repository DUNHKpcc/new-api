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
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
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

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { AffiliateDetailsDisclosure } = await import('../index')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Collapse: 'Collapse',
        'View details': 'View details',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('affiliate commission summary disclosure', () => {
  after(() => domWindow.close())

  test('hides secondary metrics until the user expands details', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <AffiliateDetailsDisclosure>
            <span>Commission details content</span>
          </AffiliateDetailsDisclosure>
        </I18nextProvider>
      )
    })

    const button = container.querySelector('button')
    assert.ok(button)
    assert.equal(button.textContent?.includes('View details'), true)
    assert.equal(button.getAttribute('aria-expanded'), 'false')
    assert.equal(
      container.textContent?.includes('Commission details content'),
      false
    )

    await act(async () => button.click())

    assert.equal(button.textContent?.includes('Collapse'), true)
    assert.equal(button.getAttribute('aria-expanded'), 'true')
    assert.equal(
      container.textContent?.includes('Commission details content'),
      true
    )

    await act(async () => root.unmount())
    container.remove()
  })
})
