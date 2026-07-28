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

import { UsersViewTabs } from '../users-view-tabs'

describe('users view tabs', () => {
  test('exposes the PccAgent management view and marks it selected', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      resources: {
        en: {
          translation: {
            'All Users': 'All Users',
            'PccAgent Authorized Users': 'PccAgent Authorized Users',
            'User view': 'User view',
          },
        },
      },
    })

    const markup = renderToStaticMarkup(
      createElement(
        I18nextProvider,
        { i18n },
        createElement(UsersViewTabs, {
          value: 'pcc_agent',
          onValueChange() {},
        })
      )
    )

    assert.match(markup, /aria-label="User view"/)
    assert.match(markup, />All Users</)
    assert.match(markup, />PccAgent Authorized Users</)
    assert.match(markup, /data-active=""[^>]*>PccAgent Authorized Users</)
  })
})
