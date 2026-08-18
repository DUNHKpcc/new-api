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

import { AccountRechargeLink } from '../account-recharge-link'

async function renderAccountRechargeLink() {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'zh',
    fallbackLng: false,
    resources: {
      zh: {
        translation: {
          'Official accounts / recharge': '官方账号/代充',
        },
      },
    },
  })

  return renderToStaticMarkup(
    createElement(I18nextProvider, { i18n }, createElement(AccountRechargeLink))
  )
}

describe('account recharge link', () => {
  test('opens the official payment page with Codex and Claude icons', async () => {
    const markup = await renderAccountRechargeLink()

    assert.match(markup, /href="https:\/\/dpccgaming\.xyz\/payment"/)
    assert.match(markup, /target="_blank"/)
    assert.match(markup, /rel="noopener noreferrer"/)
    assert.match(markup, /data-account-recharge-icon="codex"/)
    assert.match(markup, /data-account-recharge-icon="claude"/)
    assert.match(markup, /data-slot="button"/)
    assert.doesNotMatch(markup, /rounded-full/)
    assert.match(markup, />官方账号\/代充</)
  })
})
