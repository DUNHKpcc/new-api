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
*/

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { resolveWeChatLoginMode } from '../oauth'

describe('resolveWeChatLoginMode', () => {
  test('selects direct OAuth only when the explicit direct contract is ready', () => {
    assert.equal(
      resolveWeChatLoginMode({
        wechat_login: true,
        wechat_direct_oauth: true,
        wechat_server_bridge: false,
        wechat_app_id: 'wx-app-id',
      }),
      'direct'
    )
  })

  test('selects the legacy server bridge when it is the only ready contract', () => {
    assert.equal(
      resolveWeChatLoginMode({
        wechat_login: true,
        wechat_direct_oauth: false,
        wechat_server_bridge: true,
      }),
      'server'
    )
  })

  test('does not advertise a path when status says login is unavailable', () => {
    assert.equal(
      resolveWeChatLoginMode({
        wechat_login: false,
        wechat_direct_oauth: false,
        wechat_server_bridge: false,
      }),
      null
    )
  })

  test('keeps legacy status responses compatible', () => {
    assert.equal(
      resolveWeChatLoginMode({
        wechat_login: true,
        wechat_app_id: 'legacy-app-id',
      }),
      'direct'
    )
    assert.equal(
      resolveWeChatLoginMode({
        wechat_login: true,
      }),
      'server'
    )
  })

  test('does not treat a whitespace-only app id as direct OAuth', () => {
    assert.equal(
      resolveWeChatLoginMode({
        wechat_login: true,
        wechat_app_id: '   ',
      }),
      'server'
    )
  })
})
