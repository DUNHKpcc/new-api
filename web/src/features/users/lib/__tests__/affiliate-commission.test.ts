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

import {
  isAffiliateCommissionReversible,
  normalizeAffiliateCommissionReverseReason,
} from '../affiliate-commission'

describe('affiliate commission reversal policy', () => {
  test('allows every unsettled or transferred commission status', () => {
    assert.equal(isAffiliateCommissionReversible('pending'), true)
    assert.equal(isAffiliateCommissionReversible('available'), true)
    assert.equal(isAffiliateCommissionReversible('transferred'), true)
  })

  test('does not allow an already reversed commission', () => {
    assert.equal(isAffiliateCommissionReversible('reversed'), false)
  })

  test('trims a valid audit reason before submission', () => {
    assert.equal(
      normalizeAffiliateCommissionReverseReason('  payment disputed  '),
      'payment disputed'
    )
  })

  test('rejects blank and over-limit audit reasons', () => {
    assert.equal(normalizeAffiliateCommissionReverseReason('   '), null)
    assert.equal(
      normalizeAffiliateCommissionReverseReason('x'.repeat(256)),
      null
    )
  })
})
