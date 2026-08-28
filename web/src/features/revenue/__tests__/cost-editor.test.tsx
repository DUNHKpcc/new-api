/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

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

import { RevenueCostEditor } from '../components/revenue-cost-editor'

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

describe('revenue cost editor', () => {
  test('shows category details and a persisted total for the selected month', () => {
    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <RevenueCostEditor
          month='2026-08'
          currency='CNY'
          records={[
            {
              id: 1,
              month: '2026-08',
              category: 'server',
              description: 'Production host',
              amount_minor: '1250',
              currency: 'CNY',
              version: 1,
              created_by: 1,
              updated_by: 1,
              create_time: 1,
              update_time: 1,
            },
          ]}
          totalAmountMinor='1250'
          loading={false}
          error={false}
          onSaved={async () => undefined}
        />
      </I18nextProvider>
    )

    assert.match(markup, /data-revenue-cost-editor="true"/)
    assert.match(markup, /Operating costs/)
    assert.match(markup, /Production host/)
    assert.match(markup, /Server/)
    assert.match(markup, /CNY&nbsp;12\.50|CNY 12\.50|12\.50/)
    assert.match(markup, /aria-label="Edit"/)
    assert.match(markup, /aria-label="Expand"/)
    assert.match(markup, /aria-expanded="false"/)
    assert.match(markup, /data-revenue-cost-scroll="true"/)
    assert.match(markup, /class="[^"]*max-h-56[^"]*overflow-auto/)
  })
})
