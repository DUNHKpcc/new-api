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

import { buildWeChatOAuthUrl } from '../oauth'

describe('WeChat website OAuth URL', () => {
  test('uses the Open Platform QR authorization contract and local callback', () => {
    const result = new URL(
      buildWeChatOAuthUrl('wx-app-id', 'csrf-state', 'https://api.example.com')
    )

    assert.equal(
      `${result.origin}${result.pathname}`,
      'https://open.weixin.qq.com/connect/qrconnect'
    )
    assert.equal(result.searchParams.get('appid'), 'wx-app-id')
    assert.equal(
      result.searchParams.get('redirect_uri'),
      'https://api.example.com/oauth/wechat'
    )
    assert.equal(result.searchParams.get('response_type'), 'code')
    assert.equal(result.searchParams.get('scope'), 'snsapi_login')
    assert.equal(result.searchParams.get('state'), 'csrf-state')
    assert.equal(result.hash, '#wechat_redirect')
  })
})
