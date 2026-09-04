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
import { describe, test } from 'node:test'

import { createInstance } from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

const { mock } = await import(`bun:${'test'}`)

mock.module('@tanstack/react-query', () => ({
  useQuery: () => ({
    data: {
      success: true,
      data: {
        models: [
          {
            model_name: 'status-model',
            avg_latency_ms: 900,
            success_rate: 98,
            avg_tps: 42,
            recent_success_rates: [100, 92, 98],
            request_count: 100,
          },
        ],
      },
    },
    isLoading: false,
  }),
}))

const { SystemStatusPanel } = await import('../system-status-panel')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: false,
  resources: { en: { translation: {} } },
  returnNull: false,
})

function renderPanel() {
  return renderToStaticMarkup(
    createElement(I18nextProvider, { i18n }, createElement(SystemStatusPanel))
  )
}

describe('system status panel', () => {
  test('renders as a standalone announcement-sized panel', () => {
    const markup = renderPanel()

    assert.match(markup, /System status/)
    assert.doesNotMatch(markup, /Model call status/)
    assert.doesNotMatch(markup, /Minor blips in the leading models/)
    assert.match(markup, /data-system-status-panel="true"/)
    assert.match(markup, /class="relative h-72"/)
    assert.equal(
      (markup.match(/data-performance-status-segment="true"/g) ?? []).length,
      72
    )
  })
})
