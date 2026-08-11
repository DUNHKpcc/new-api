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

import i18next from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { initReactI18next } from 'react-i18next'

import { AffiliateCommissionList, CashWithdrawalComingSoon } from '../index'
import type { AffiliateCommission } from '../types'

await i18next.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

function commission(id: number): AffiliateCommission {
  return {
    id: String(id),
    referred_user: `de***${String(id).padStart(2, '0')}`,
    paid_amount_minor: '10000',
    paid_currency: 'CNY',
    purchased_quota: '5000000',
    commission_rate_bps: 1000,
    commission_amount_minor: '1000',
    reward_quota: '500000',
    debt_offset_quota: '0',
    status: 'available',
    available_at: 1,
    transferred_at: 0,
    reversed_at: 0,
    created_at: 1,
  }
}

describe('affiliate commission list layout', () => {
  test('keeps long commission history inside a bounded scroll region', () => {
    const markup = renderToStaticMarkup(
      <AffiliateCommissionList
        loading={false}
        items={Array.from({ length: 12 }, (_, index) => commission(index + 1))}
      />
    )

    assert.match(markup, /data-slot="affiliate-commission-scroll-area"/)
    assert.match(markup, /max-h-\[min\(58svh,36rem\)\]/)
    assert.match(markup, /overflow-y-auto/)
    assert.match(markup, /overscroll-contain/)
    assert.match(markup, /Username/)
    assert.match(markup, /Actual payment/)
    assert.match(markup, /Purchased balance/)
    assert.match(markup, /Cash withdrawal value/)
    assert.match(markup, /Balance reward/)
    assert.match(markup, /de\*\*\*12/)
  })

  test('keeps the empty state simple without an inactive scroll surface', () => {
    const markup = renderToStaticMarkup(
      <AffiliateCommissionList loading={false} items={[]} />
    )

    assert.match(markup, /No commissions yet/)
    assert.doesNotMatch(markup, /affiliate-commission-scroll-area/)
  })

  test('presents Alipay withdrawal as unavailable without an active action', () => {
    const markup = renderToStaticMarkup(<CashWithdrawalComingSoon />)

    assert.match(markup, /affiliate-cash-withdrawal-coming-soon/)
    assert.match(markup, /Alipay cash withdrawal/)
    assert.match(markup, /Alipay withdrawal is not available yet/)
    assert.match(markup, /disabled=""/)
  })
})
