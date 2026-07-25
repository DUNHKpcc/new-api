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

import { isSafeDesktopLoopbackRedirect } from '../loopback-redirect'

describe('desktop authorization loopback redirects', () => {
  test('accepts an exact loopback callback carrying state and a result', () => {
    assert.equal(
      isSafeDesktopLoopbackRedirect(
        'http://127.0.0.1:49152/oauth/callback/0123456789abcdefghijkl?code=one-time&state=expected'
      ),
      true
    )
  })

  test('rejects alternate hosts, privileged ports, and incomplete callbacks', () => {
    const invalid = [
      'http://localhost:49152/oauth/callback/0123456789abcdefghijkl?code=one-time&state=expected',
      'https://127.0.0.1:49152/oauth/callback/0123456789abcdefghijkl?code=one-time&state=expected',
      'http://127.0.0.1:80/oauth/callback/0123456789abcdefghijkl?code=one-time&state=expected',
      'http://127.0.0.1:49152/oauth/callback/short?code=one-time&state=expected',
      'http://127.0.0.1:49152/oauth/callback/0123456789abcdefghijkl?state=expected',
      'http://127.0.0.1:49152/oauth/callback/0123456789abcdefghijkl?code=one-time',
    ]

    for (const value of invalid) {
      assert.equal(isSafeDesktopLoopbackRedirect(value), false, value)
    }
  })
})
