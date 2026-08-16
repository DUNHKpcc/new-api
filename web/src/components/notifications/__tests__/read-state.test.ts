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
  getUnreadNotificationItems,
  type NotificationFeedItem,
} from '../notification-feed'

const unreadNotice: NotificationFeedItem = {
  key: 'notice',
  source: 'notice',
  content: 'Scheduled maintenance',
  unread: true,
}

describe('notification read state', () => {
  test('does not expose a floating unread preview while sources are loading', () => {
    assert.deepEqual(getUnreadNotificationItems([unreadNotice], true), [])
  })

  test('exposes only unread items after every source finishes loading', () => {
    const readNotice: NotificationFeedItem = {
      ...unreadNotice,
      unread: false,
    }

    assert.deepEqual(
      getUnreadNotificationItems([readNotice, unreadNotice], false),
      [unreadNotice]
    )
  })
})
