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
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { RankingAdjustmentRow } from '../ranking-adjustment-row'

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
  interpolation: { escapeValue: false },
})

describe('ranking adjustment row', () => {
  test('shows live, added, and final values with exact token counts', () => {
    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <RankingAdjustmentRow
          modelName='model-a'
          liveTokens={240_000}
          addedTokens={10_000}
          enabled
          onAdjustmentChange={() => {}}
        />
      </I18nextProvider>
    )

    assert.match(markup, />240K</)
    assert.match(markup, />240000</)
    assert.match(markup, /value="10000"/)
    assert.match(markup, />250K</)
    assert.match(markup, />250000</)
    assert.match(markup, /aria-label="Added tokens for model-a"/)
  })

  test('keeps mobile field labels visible without horizontal scrolling', () => {
    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <RankingAdjustmentRow
          modelName='a-very-long-ranking-model-name'
          liveTokens={1}
          addedTokens={0}
          enabled
          onAdjustmentChange={() => {}}
        />
      </I18nextProvider>
    )

    assert.match(markup, />Added tokens</)
    assert.match(markup, />Final display</)
    assert.doesNotMatch(markup, /overflow-x-auto/)
  })
})
