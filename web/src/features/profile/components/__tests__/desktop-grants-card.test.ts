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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createInstance } from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { DesktopGrantsCard } from '../desktop-grants-card'

describe('desktop grants empty state', () => {
  test('links to the official Microsoft Store download with its icon', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      resources: {
        en: {
          translation: {
            'Apps you authorize for PccAgent will appear here.':
              'Apps you authorize for PccAgent will appear here.',
            'Authorized desktop devices': 'Authorized desktop devices',
            'Download PccAgent from Microsoft Store':
              'Download PccAgent from Microsoft Store',
            'No authorized desktop devices': 'No authorized desktop devices',
            'Review and revoke PccAgent apps connected to your account.':
              'Review and revoke PccAgent apps connected to your account.',
          },
        },
      },
    })

    const queryClient = new QueryClient()
    queryClient.setQueryData(['profile', 'desktop-grants'], [])

    const markup = renderToStaticMarkup(
      createElement(
        I18nextProvider,
        { i18n },
        createElement(
          QueryClientProvider,
          { client: queryClient },
          createElement(DesktopGrantsCard)
        )
      )
    )

    assert.match(
      markup,
      /href="https:\/\/apps\.microsoft\.com\/detail\/9pf5ff13cbhp\?hl=zh-CN&amp;gl=CN"/
    )
    assert.match(markup, /target="_blank"/)
    assert.match(markup, /rel="noopener noreferrer"/)
    assert.match(markup, /data-icon="microsoft-store"/)
    assert.match(markup, /Download PccAgent from Microsoft Store/)
  })
})
