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

import type { TopupSummary, UserWalletData } from '../../types'
import { WalletStatsCard } from '../wallet-stats-card'

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const user: UserWalletData = {
  id: 1,
  username: 'wallet-user',
  quota: 500000,
  used_quota: 100000,
  request_count: 20,
  aff_quota: 0,
  aff_history_quota: 0,
  aff_count: 0,
  group: 'default',
}

const summary: TopupSummary = {
  basis: 'provider_verified_payment',
  covered_providers: ['epay'],
  default_currency: 'CNY',
  totals: [
    {
      payment_provider: 'epay',
      currency: 'CNY',
      amount_minor: '1234',
    },
  ],
}

function renderStats(options?: {
  topupSummaryError?: boolean
  topupSummaryLoading?: boolean
}) {
  return renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <WalletStatsCard
        user={user}
        topupSummary={options?.topupSummaryLoading ? null : summary}
        topupSummaryError={options?.topupSummaryError}
        topupSummaryLoading={options?.topupSummaryLoading}
      />
    </I18nextProvider>
  )
}

describe('wallet stats', () => {
  test('shows verified payment total in a responsive four-stat grid', () => {
    const markup = renderStats()

    assert.match(markup, /Verified Topup Total/)
    assert.match(markup, /12\.34/)
    assert.match(markup, /grid-cols-2/)
    assert.match(markup, /md:grid-cols-4/)
  })

  test('shows unavailable marker when the summary request fails', () => {
    const markup = renderStats({ topupSummaryError: true })

    assert.match(markup, />--</)
  })

  test('keeps existing statistics visible while the topup summary loads', () => {
    const markup = renderStats({ topupSummaryLoading: true })

    assert.match(markup, /Current Balance/)
    assert.match(markup, /Total Usage/)
    assert.match(markup, /data-topup-summary-loading="true"/)
    assert.equal(markup.match(/data-slot="skeleton"/g)?.length, 1)
  })
})
