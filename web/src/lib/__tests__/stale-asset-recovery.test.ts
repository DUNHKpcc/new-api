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
  attemptStaleAssetRecovery,
  isStaleAssetError,
} from '../stale-asset-recovery'

function createStorage() {
  const values = new Map<string, string>()
  return {
    getItem(key: string) {
      return values.get(key) ?? null
    },
    setItem(key: string, value: string) {
      values.set(key, value)
    },
  }
}

describe('stale frontend asset recovery', () => {
  test('recognizes Rspack route chunk load failures', () => {
    const error = new Error(
      'Loading chunk 9243 failed.\n(missing: /static/js/async/9243.js)'
    )
    error.name = 'ChunkLoadError'

    assert.equal(isStaleAssetError(error), true)
  })

  test('reloads once for the same stale build', () => {
    const storage = createStorage()
    const error = new Error('Loading chunk 9243 failed.')
    error.name = 'ChunkLoadError'
    let reloads = 0

    assert.equal(
      attemptStaleAssetRecovery(error, 'rv.rc29.test', storage, () => {
        reloads += 1
      }),
      true
    )
    assert.equal(
      attemptStaleAssetRecovery(error, 'rv.rc29.test', storage, () => {
        reloads += 1
      }),
      false
    )
    assert.equal(reloads, 1)
  })

  test('does not reload for an ordinary render error', () => {
    let reloads = 0

    assert.equal(
      attemptStaleAssetRecovery(
        new Error('Cannot read properties of undefined'),
        'rv.rc29.test',
        createStorage(),
        () => {
          reloads += 1
        }
      ),
      false
    )
    assert.equal(reloads, 0)
  })
})
