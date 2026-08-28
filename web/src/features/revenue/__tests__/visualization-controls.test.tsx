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

import { RevenueVisualization } from '../components/revenue-visualization'
import type { PlatformRevenueSummary } from '../types'

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const platform: PlatformRevenueSummary = {
  totals: [
    { currency: 'CNY', amount_minor: '5600', count: 3, verified_count: 3 },
  ],
  by_payment_method: [
    {
      payment_method: 'alipay',
      currency: 'CNY',
      amount_minor: '5600',
      count: 3,
    },
  ],
  timeline: [],
  default_currency: 'CNY',
  start_time: 1,
  end_time: 2,
  timezone_offset: 480,
  normalization_basis: 'verified_settlement_or_recorded_order_amount',
}

describe('revenue visualization controls', () => {
  test('gives both segmented controls a lighter border and room around buttons', () => {
    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <RevenueVisualization
          currency='CNY'
          platform={platform}
          external={undefined}
        />
      </I18nextProvider>
    )

    for (const control of ['metric', 'view']) {
      const className = markup.match(
        new RegExp(`class="([^"]+)" data-revenue-${control}-toggle="true"`)
      )?.[1]

      assert.ok(className)
      assert.match(className, /border-border\/60/)
      assert.match(className, /p-1/)
    }

    assert.match(markup, /aria-label="Bar Chart"/)
    assert.match(markup, /aria-label="Area Chart"/)
    assert.match(markup, /aria-label="Flow chart"/)
    assert.match(markup, /aria-pressed="true"[^>]*title="Bar Chart"/)
  })
})
