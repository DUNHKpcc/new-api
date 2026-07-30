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

import type { NotificationFeedItem } from '../notification-feed'
import { partitionNotificationItems } from '../notification-sections'

describe('system announcements popover', () => {
  test('keeps notices and timeline announcements in separate tabs', () => {
    const items: NotificationFeedItem[] = [
      {
        key: 'discount',
        source: 'discount',
        content: 'Discount',
        unread: true,
      },
      {
        key: 'notice',
        source: 'notice',
        content: 'System notice',
        unread: true,
      },
      {
        key: 'announcement',
        source: 'announcement',
        content: 'Timeline update',
        unread: false,
      },
    ]

    const sections = partitionNotificationItems(items)

    assert.deepEqual(
      sections.notice.map((item) => item.key),
      ['discount', 'notice']
    )
    assert.deepEqual(
      sections.timeline.map((item) => item.key),
      ['announcement']
    )
  })
})
