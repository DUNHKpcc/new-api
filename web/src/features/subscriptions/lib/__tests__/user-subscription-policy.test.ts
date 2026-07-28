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

import { getUserSubscriptionActionPolicy } from '../user-subscription-policy'

const now = 1_700_000_000

describe('user subscription action policy', () => {
  test('allows an active PccAgent gift to be invalidated but not reset or deleted', () => {
    const policy = getUserSubscriptionActionPolicy(
      {
        source: 'pcc_agent_gift',
        status: 'active',
        end_time: now + 3600,
      },
      now
    )

    assert.equal(policy.isActive, true)
    assert.equal(policy.showReset, false)
    assert.equal(policy.canReset, false)
    assert.equal(policy.canInvalidate, true)
    assert.equal(policy.canDelete, false)
  })

  test('preserves reset, invalidate and delete actions for an active ordinary subscription', () => {
    const policy = getUserSubscriptionActionPolicy(
      {
        source: 'admin',
        status: 'active',
        end_time: now + 3600,
      },
      now
    )

    assert.equal(policy.showReset, true)
    assert.equal(policy.canReset, true)
    assert.equal(policy.canInvalidate, true)
    assert.equal(policy.canDelete, true)
  })

  test('disables state-changing actions after an ordinary subscription expires', () => {
    const policy = getUserSubscriptionActionPolicy(
      {
        source: 'order',
        status: 'active',
        end_time: now - 1,
      },
      now
    )

    assert.equal(policy.showReset, true)
    assert.equal(policy.canReset, false)
    assert.equal(policy.canInvalidate, false)
    assert.equal(policy.canDelete, true)
  })
})
