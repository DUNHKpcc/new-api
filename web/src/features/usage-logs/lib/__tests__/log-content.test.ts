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

import { translateLogContent } from '../format'

describe('usage log content localization', () => {
  test('translates stable affiliate and payment log messages', () => {
    const translations: Record<string, string> = {
      'Affiliate commission created': '代理佣金已生成',
      'Online top-up completed': '在线充值已到账',
    }
    const t = (key: string) => translations[key] ?? key

    assert.equal(
      translateLogContent('Affiliate commission created', t),
      '代理佣金已生成'
    )
    assert.equal(
      translateLogContent('Online top-up completed', t),
      '在线充值已到账'
    )
  })

  test('preserves arbitrary log content instead of treating it as a key', () => {
    let translationCalls = 0
    const t = (key: string) => {
      translationCalls += 1
      return `translated:${key}`
    }

    assert.equal(
      translateLogContent('Provider returned 503', t),
      'Provider returned 503'
    )
    assert.equal(translationCalls, 0)
  })
})
