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
import { after, describe, test } from 'node:test'

const { mock } = await import(`bun:${'test'}`)

let requestedUrl = ''
mock.module('@/lib/api', () => ({
  api: {
    get: async (url: string) => {
      requestedUrl = url
      return { data: { success: true, data: { items: [], total: 0 } } }
    },
  },
}))

const { getAllBillingHistory } = await import('../api')

describe('admin billing history API', () => {
  after(() => mock.restore())

  test('sends the selected completion month with the existing list request', async () => {
    await getAllBillingHistory(2, 20, 'order-1', {
      startTime: 1_722_556_800,
      endTime: 1_725_148_800,
    })

    assert.equal(
      requestedUrl,
      '/api/user/topup?p=2&page_size=20&keyword=order-1&start_time=1722556800&end_time=1725148800'
    )
  })
})
