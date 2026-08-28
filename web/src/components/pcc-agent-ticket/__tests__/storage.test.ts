/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

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
  getPccAgentTicketStorageKey,
  markPccAgentTicketSeen,
  readPccAgentTicketSeen,
  type TicketStorage,
} from '../pcc-agent-ticket-storage'

function createStorage(initial: Record<string, string> = {}): TicketStorage & {
  values: Record<string, string>
} {
  const values = { ...initial }
  return {
    values,
    getItem: (key) => values[key] ?? null,
    setItem: (key, value) => {
      values[key] = value
    },
  }
}

describe('PccAgent ticket storage', () => {
  test('uses a separate persistent key for each account and visitors', () => {
    assert.equal(
      getPccAgentTicketStorageKey(42),
      'pcc-agent-ticket:v1:seen:user:42'
    )
    assert.equal(
      getPccAgentTicketStorageKey(null),
      'pcc-agent-ticket:v1:seen:visitor'
    )
    assert.notEqual(
      getPccAgentTicketStorageKey(42),
      getPccAgentTicketStorageKey(43)
    )
  })

  test('marks an unseen ticket and reads it on the next visit', () => {
    const storage = createStorage()
    const key = getPccAgentTicketStorageKey(42)

    assert.equal(readPccAgentTicketSeen(storage, key), false)
    assert.equal(markPccAgentTicketSeen(storage, key), true)
    assert.equal(readPccAgentTicketSeen(storage, key), true)
  })

  test('fails closed when browser storage throws', () => {
    const storage: TicketStorage = {
      getItem: () => {
        throw new Error('storage unavailable')
      },
      setItem: () => {
        throw new Error('storage unavailable')
      },
    }
    const key = getPccAgentTicketStorageKey(undefined)

    assert.equal(readPccAgentTicketSeen(storage, key), false)
    assert.equal(markPccAgentTicketSeen(storage, key), false)
  })
})
