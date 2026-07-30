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

import { buildNotificationFeed } from '../notification-feed'

describe('notification feed ordering', () => {
  test('keeps every unread notification above read content', () => {
    const feed = buildNotificationFeed({
      notice: 'Scheduled maintenance',
      lastReadNotice: 'Scheduled maintenance',
      readAnnouncementKeys: ['id:1'],
      announcements: [
        {
          id: 1,
          content: 'Resolved incident',
          publishDate: '2026-07-30T12:00:00.000Z',
        },
        {
          id: 2,
          content: 'Older unread update',
          publishDate: '2026-07-28T12:00:00.000Z',
        },
        {
          id: 3,
          content: 'Newest unread update',
          publishDate: '2026-07-30T13:00:00.000Z',
        },
      ],
    })

    assert.deepEqual(
      feed.map((item) => item.unread),
      [true, true, false, false]
    )
    assert.deepEqual(
      feed.slice(0, 2).map((item) => item.key),
      ['id:3', 'id:2']
    )
    assert.equal(feed[2]?.source, 'notice')
  })

  test('treats edited notice and id-less announcement content as new', () => {
    const initialFeed = buildNotificationFeed({
      notice: 'Initial notice',
      lastReadNotice: 'Initial notice',
      readAnnouncementKeys: [],
      announcements: [{ content: 'Initial timeline item' }],
    })
    const initialAnnouncement = initialFeed.find(
      (item) => item.source === 'announcement'
    )
    const updatedFeed = buildNotificationFeed({
      notice: 'Updated notice',
      lastReadNotice: 'Initial notice',
      readAnnouncementKeys: [initialAnnouncement?.key ?? ''],
      announcements: [{ content: 'Updated timeline item' }],
    })

    assert.equal(updatedFeed[0]?.source, 'notice')
    assert.equal(updatedFeed[0]?.unread, true)
    assert.equal(updatedFeed[1]?.unread, true)
    assert.notEqual(initialAnnouncement?.key, updatedFeed[1]?.key)
  })
})
