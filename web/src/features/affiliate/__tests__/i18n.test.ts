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

import fr from '@/i18n/locales/fr.json'
import ja from '@/i18n/locales/ja.json'
import ru from '@/i18n/locales/ru.json'
import vi from '@/i18n/locales/vi.json'
import zhTW from '@/i18n/locales/zh-TW.json'
import zh from '@/i18n/locales/zh.json'

describe('affiliate translations', () => {
  test('does not fall back to raw English for affiliate labels', () => {
    const locales = [fr, ja, ru, vi, zhTW, zh]
    const keys = [
      'Affiliate',
      'Lifetime',
      'Follow system threshold',
      'Allow without threshold',
      'Deny affiliate access',
      'Affiliate commissions',
      'Commission amount',
      'Purchased balance',
      'Cash withdrawal value',
      'Balance reward',
      'Alipay cash withdrawal',
      'Reversal reason',
      'Reverse affiliate commission',
      'Reverse commission',
    ] as const

    for (const locale of locales) {
      for (const key of keys) {
        assert.ok(locale.translation[key])
        assert.notEqual(locale.translation[key], key)
      }
    }
  })

  test('uses explicit Chinese labels for commission accounting states', () => {
    assert.equal(zh.translation['Balance rewards available'], '可转入余额奖励')
    assert.equal(zh.translation['Balance rewards pending'], '待确认余额奖励')
    assert.equal(zh.translation['Lifetime balance rewards'], '累计有效余额奖励')
    assert.equal(zh.translation['Balance reward debt'], '待抵扣余额奖励')
  })

  test('distinguishes cash withdrawal from balance reward calculation', () => {
    assert.equal(zh.translation['Actual payment'], '实际支付')
    assert.equal(zh.translation['Purchased balance'], '购买额度')
    assert.equal(zh.translation['Cash withdrawal value'], '现金提现金额')
    assert.equal(zh.translation['Balance reward'], '转入余额奖励')
    assert.equal(
      zh.translation[
        'Cash withdrawal uses the verified payment amount; transfer to balance uses the purchased balance.'
      ],
      '现金提现按已核验实付金额计算，转入余额按本次购买额度计算。'
    )
  })

  test('explains affiliate access choices in Chinese', () => {
    assert.equal(zh.translation['Follow system threshold'], '遵循系统充值门槛')
    assert.equal(zh.translation['Allow without threshold'], '允许跳过充值门槛')
    assert.equal(zh.translation['Deny affiliate access'], '禁止代理佣金')
    assert.equal(
      zh.translation[
        'If the user has not met the current recharge threshold, this will lock the affiliate program and pause commission transfers.'
      ],
      '若用户未达到当前充值门槛，保存后代理功能将重新锁定，未转入佣金也会暂停转入。'
    )
  })

  test('explains commission reversal scope and reason visibility in Chinese', () => {
    assert.equal(
      zh.translation[
        "Only the affiliate commission will be reversed. The payment will not be refunded and the referred user's top-up quota will not be changed. If the commission was already transferred, it becomes commission debt and future commissions will offset it first. The reason will be visible to the affiliate and recorded in the admin audit log."
      ],
      '只会冲正这笔代理佣金，不会发起退款，也不会扣除被邀请用户的充值额度。如果佣金已经转入余额，将转为待抵扣佣金额度，后续佣金会优先抵扣。原因会展示给代理本人，并记录到管理员审计日志。'
    )
  })
})
