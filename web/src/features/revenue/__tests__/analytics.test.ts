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
  buildRevenueFlowGraph,
  buildRevenueTimeline,
  calculateRevenueProfit,
  combineRevenueTotals,
  decimalAmountToMinor,
  minorAmountToDecimal,
  nonNegativeDecimalAmountToMinor,
  sumMinorAmounts,
} from '../lib/analytics'
import type { ExternalRevenueSummary, PlatformRevenueSummary } from '../types'

const platform: PlatformRevenueSummary = {
  totals: [
    { currency: 'CNY', amount_minor: '12000', count: 2, verified_count: 1 },
  ],
  by_payment_method: [
    {
      payment_method: 'wxpay',
      currency: 'CNY',
      amount_minor: '12000',
      count: 2,
    },
  ],
  timeline: [],
  default_currency: 'CNY',
  start_time: 1,
  end_time: 2,
  timezone_offset: 480,
  normalization_basis: 'verified_settlement_or_recorded_order_amount',
}

const external: ExternalRevenueSummary = {
  totals: [{ currency: 'CNY', amount_minor: '3450', count: 1 }],
  by_source: [
    {
      source: 'xianyu',
      source_label: '',
      currency: 'CNY',
      amount_minor: '3450',
      count: 1,
    },
  ],
  timeline: [],
  start_time: 1,
  end_time: 2,
  timezone_offset: 480,
}

describe('revenue analytics', () => {
  test('combines platform and external amounts only within the same currency', () => {
    const totals = combineRevenueTotals(platform, {
      ...external,
      totals: [
        ...(external.totals ?? []),
        { currency: 'USD', amount_minor: '500', count: 1 },
      ],
    })

    assert.deepEqual(totals, [
      {
        currency: 'CNY',
        platformAmountMinor: '12000',
        externalAmountMinor: '3450',
        totalAmountMinor: '15450',
        platformCount: 2,
        externalCount: 1,
      },
      {
        currency: 'USD',
        platformAmountMinor: '0',
        externalAmountMinor: '500',
        totalAmountMinor: '500',
        platformCount: 0,
        externalCount: 1,
      },
    ])
  })

  test('treats null summary arrays as empty data instead of throwing', () => {
    const emptyPlatform: PlatformRevenueSummary = {
      ...platform,
      totals: null,
      by_payment_method: null,
      timeline: null,
    }
    const emptyExternal: ExternalRevenueSummary = {
      ...external,
      totals: null,
      by_source: null,
      timeline: null,
    }

    assert.deepEqual(combineRevenueTotals(emptyPlatform, emptyExternal), [])
    assert.deepEqual(
      buildRevenueFlowGraph(
        emptyPlatform,
        emptyExternal,
        'CNY',
        { platform: 'Platform', external: 'External' },
        'amount'
      ),
      { nodes: [], links: [] }
    )
  })

  test('builds the reused flow graph from both revenue sources', () => {
    const graph = buildRevenueFlowGraph(
      platform,
      external,
      'CNY',
      { platform: 'Platform', external: 'External' },
      'amount'
    )

    const nodeIds = new Set(graph.nodes.map((node) => node.id))
    assert.ok(nodeIds.has('root:platform'))
    assert.ok(nodeIds.has('root:external'))
    assert.ok(nodeIds.has('platform:wxpay'))
    assert.ok(nodeIds.has('external:xianyu:xianyu'))
    assert.ok(nodeIds.has('currency:CNY'))
    assert.equal(graph.links.length, 4)
  })

  test('converts decimal form amounts without floating-point rounding', () => {
    assert.equal(decimalAmountToMinor('10.05'), '1005')
    assert.equal(decimalAmountToMinor('0'), null)
    assert.equal(decimalAmountToMinor('2.345'), null)
    assert.equal(minorAmountToDecimal('1005'), '10.05')
  })

  test('accepts zero and rejects unsafe manual cost inputs', () => {
    assert.equal(nonNegativeDecimalAmountToMinor('0'), '0')
    assert.equal(nonNegativeDecimalAmountToMinor('10.05'), '1005')
    assert.equal(nonNegativeDecimalAmountToMinor('-1'), null)
    assert.equal(nonNegativeDecimalAmountToMinor('1.001'), null)
    assert.equal(nonNegativeDecimalAmountToMinor('10000000000000000.01'), null)
  })

  test('calculates signed net income and margin in minor units', () => {
    assert.deepEqual(calculateRevenueProfit('10000', '2500'), {
      revenueAmountMinor: '10000',
      costAmountMinor: '2500',
      netAmountMinor: '7500',
      marginPercent: 75,
    })
    assert.equal(calculateRevenueProfit('0', '100').marginPercent, null)
    assert.equal(calculateRevenueProfit('100', '250').netAmountMinor, '-150')
  })

  test('sums persisted cost amounts without floating-point rounding', () => {
    assert.equal(sumMinorAmounts(['1005', '245', 'not-a-number']), '1250')
  })

  test('fills daily revenue buckets and switches the chart metric', () => {
    const startTime = Math.floor(Date.parse('2026-08-01T00:00:00Z') / 1000)
    const endTime = Math.floor(Date.parse('2026-08-04T00:00:00Z') / 1000)
    const platformWithTimeline: PlatformRevenueSummary = {
      ...platform,
      start_time: startTime,
      end_time: endTime,
      timezone_offset: 0,
      timeline: [
        {
          date: '2026-08-01',
          payment_method: 'wxpay',
          currency: 'CNY',
          amount_minor: '1000',
          count: 1,
        },
        {
          date: '2026-08-01',
          payment_method: 'alipay',
          currency: 'CNY',
          amount_minor: '250',
          count: 2,
        },
      ],
    }
    const externalWithTimeline: ExternalRevenueSummary = {
      ...external,
      start_time: startTime,
      end_time: endTime,
      timezone_offset: 0,
      timeline: [
        {
          date: '2026-08-03',
          source: 'xianyu',
          source_label: 'Xianyu',
          currency: 'CNY',
          amount_minor: '500',
          count: 1,
        },
      ],
    }

    const amountTimeline = buildRevenueTimeline(
      platformWithTimeline,
      externalWithTimeline,
      'CNY',
      'amount'
    )
    assert.equal(amountTimeline.length, 6)
    assert.deepEqual(
      amountTimeline.filter((item) => item.date === '2026-08-01'),
      [
        {
          date: '2026-08-01',
          source: 'platform',
          amountMinor: '1250',
          count: 3,
          value: 12.5,
        },
        {
          date: '2026-08-01',
          source: 'external',
          amountMinor: '0',
          count: 0,
          value: 0,
        },
      ]
    )
    const countTimeline = buildRevenueTimeline(
      platformWithTimeline,
      externalWithTimeline,
      'CNY',
      'count'
    )
    assert.equal(
      countTimeline.find(
        (item) => item.date === '2026-08-01' && item.source === 'platform'
      )?.value,
      3
    )
    assert.equal(
      countTimeline.find(
        (item) => item.date === '2026-08-02' && item.source === 'external'
      )?.value,
      0
    )
  })
})
