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

import type { TopupRecord } from '@/features/wallet/types'

import {
  getOrderUser,
  isSubscriptionOrderTradeNo,
  resolveOrderSubscription,
  type PlatformOrderContext,
} from '../lib/platform-orders'

const baseOrder: TopupRecord = {
  id: 1,
  user_id: 42,
  amount: 0,
  money: 99,
  trade_no: 'SUBUSR42NOexample',
  payment_method: 'stripe',
  payment_provider: 'stripe',
  create_time: 1_700_000_000,
  complete_time: 1_700_000_010,
  status: 'success',
}

const context: PlatformOrderContext = {
  users: {
    '42': { id: 42, username: 'alice', display_name: 'Alice' },
  },
  subscriptions: {
    '42': [
      {
        subscription: {
          id: 7,
          user_id: 42,
          plan_id: 3,
          status: 'active',
          source: 'order',
          start_time: 1_700_000_010,
          end_time: 1_700_100_000,
          amount_total: 1000,
          amount_used: 0,
          created_at: 1_700_000_011,
        },
      },
    ],
  },
  planTitles: { '3': 'Pro monthly' },
}

describe('platform order enrichment', () => {
  test('recognizes the subscription trade number formats used by providers', () => {
    assert.equal(isSubscriptionOrderTradeNo('SUBUSR42NOabc'), true)
    assert.equal(isSubscriptionOrderTradeNo('sub_ref_abc'), true)
    assert.equal(isSubscriptionOrderTradeNo('WAFFO_PANCAKE_SUB-42-1-abc'), true)
    assert.equal(isSubscriptionOrderTradeNo('wallet-topup-42'), false)
  })

  test('uses the existing user and subscription resources for row labels', () => {
    assert.deepEqual(getOrderUser(baseOrder, context), {
      id: 42,
      username: 'alice',
      display_name: 'Alice',
    })
    assert.deepEqual(resolveOrderSubscription(baseOrder, context), {
      planId: 3,
      title: 'Pro monthly',
      matched: true,
    })
  })

  test('keeps ordinary wallet topups without a fabricated subscription', () => {
    const walletOrder = { ...baseOrder, trade_no: 'wallet-42' }
    assert.equal(resolveOrderSubscription(walletOrder, context), undefined)
  })
})
