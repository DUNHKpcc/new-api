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

import { ResourceDownloadsGrid } from '../resource-downloads-grid'

describe('resource downloads grid', () => {
  test('renders resources in a responsive grid with safe download links', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      resources: {
        en: {
          translation: {
            Download: 'Download',
          },
        },
      },
    })

    const markup = renderToStaticMarkup(
      createElement(
        I18nextProvider,
        { i18n },
        createElement(ResourceDownloadsGrid, {
          items: [
            {
              id: 'desktop-client',
              name: 'Desktop Client',
              description: 'Stable release',
              url: 'https://example.com/client',
              thumbnail: 'data:image/webp;base64,UklGRg==',
            },
            {
              id: 'command-line-tools',
              name: 'Command Line Tools',
              description: '',
              url: 'https://example.com/cli',
              thumbnail: 'data:image/webp;base64,UklGRg==',
            },
          ],
        })
      )
    )

    assert.match(markup, /data-layout="responsive-resource-grid"/)
    assert.match(markup, /repeat\(auto-fill,minmax\(min\(100%,17rem\),1fr\)\)/)
    assert.equal((markup.match(/aspect-video/g) ?? []).length, 2)
    assert.equal((markup.match(/<article/g) ?? []).length, 2)
    assert.match(markup, /target="_blank"/)
    assert.match(markup, /rel="noopener noreferrer"/)
    assert.match(markup, /loading="lazy"/)
  })
})
