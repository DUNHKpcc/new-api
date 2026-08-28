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
  buildNotificationFeed,
  getAnnouncementNotificationKey,
  migrateLegacyAnnouncementReadKeys,
} from '../notification-feed'

describe('notification feed ordering', () => {
  test('keeps every unread notification above read content', () => {
    const feed = buildNotificationFeed({
      discountNotice: '',
      notice: 'Scheduled maintenance',
      lastReadDiscountNotice: '',
      lastReadNotice: 'Scheduled maintenance',
      readAnnouncementKeys: [
        getAnnouncementNotificationKey({
          id: 1,
          content: 'Resolved incident',
          publishDate: '2026-07-30T12:00:00.000Z',
        }),
      ],
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
      feed.slice(0, 2).map((item) => item.content),
      ['Newest unread update', 'Older unread update']
    )
    assert.equal(feed[2]?.source, 'notice')
  })

  test('treats edited notice and id-less announcement content as new', () => {
    const initialFeed = buildNotificationFeed({
      discountNotice: '',
      notice: 'Initial notice',
      lastReadDiscountNotice: '',
      lastReadNotice: 'Initial notice',
      readAnnouncementKeys: [],
      announcements: [{ content: 'Initial timeline item' }],
    })
    const initialAnnouncement = initialFeed.find(
      (item) => item.source === 'announcement'
    )
    const updatedFeed = buildNotificationFeed({
      discountNotice: '',
      notice: 'Updated notice',
      lastReadDiscountNotice: '',
      lastReadNotice: 'Initial notice',
      readAnnouncementKeys: [initialAnnouncement?.key ?? ''],
      announcements: [{ content: 'Updated timeline item' }],
    })

    assert.equal(updatedFeed[0]?.source, 'notice')
    assert.equal(updatedFeed[0]?.unread, true)
    assert.equal(updatedFeed[1]?.unread, true)
    assert.notEqual(initialAnnouncement?.key, updatedFeed[1]?.key)
  })

  test('treats an edited identified announcement as unread', () => {
    const initialFeed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [],
      announcements: [{ id: 1, content: 'Initial timeline item' }],
    })
    const updatedFeed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [initialFeed[0]?.key ?? ''],
      announcements: [{ id: 1, content: 'Updated timeline item' }],
    })

    assert.notEqual(initialFeed[0]?.key, updatedFeed[0]?.key)
    assert.equal(updatedFeed[0]?.unread, true)
  })

  test('migrates legacy identified announcement reads without replaying them', () => {
    const announcement = { id: 1, content: 'Existing timeline item' }
    const legacyFeed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: ['id:1'],
      announcements: [announcement],
    })
    const migratedKeys = migrateLegacyAnnouncementReadKeys(
      ['id:1'],
      [announcement]
    )
    const updatedFeed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: migratedKeys,
      announcements: [{ ...announcement, content: 'Updated timeline item' }],
    })

    assert.equal(legacyFeed[0]?.unread, false)
    assert.deepEqual(migratedKeys, [
      getAnnouncementNotificationKey(announcement),
    ])
    assert.equal(updatedFeed[0]?.unread, true)
  })

  test('keeps an unread discount notice above other unread notifications', () => {
    const feed = buildNotificationFeed({
      discountNotice: 'GPT-5 is 20% off through August 31.',
      notice: 'Scheduled maintenance',
      announcements: [{ id: 1, content: 'New model available' }],
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [],
    })

    assert.equal(feed[0]?.source, 'discount')
    assert.equal(feed[0]?.unread, true)
    assert.equal(feed[0]?.content, 'GPT-5 is 20% off through August 31.')
  })

  test('treats an edited discount notice as unread', () => {
    const feed = buildNotificationFeed({
      discountNotice: 'GPT-5 is now 30% off.',
      notice: '',
      announcements: [],
      lastReadDiscountNotice: 'GPT-5 is 20% off.',
      lastReadNotice: '',
      readAnnouncementKeys: [],
    })

    assert.equal(feed[0]?.source, 'discount')
    assert.equal(feed[0]?.unread, true)
  })

  test('preserves lottery image and winning information in the unread feed', () => {
    const feed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      announcements: [],
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [],
      readLotteryKeys: [],
      lotteries: [
        {
          id: 'summer-draw',
          title: 'Summer draw',
          content: 'Join before Friday.',
          winnerInfo: 'Winner #42',
          image: 'data:image/webp;base64,UklGRg==',
          publishDate: '2026-08-09T08:00:00Z',
        },
      ],
    })

    assert.equal(feed[0]?.source, 'lottery')
    assert.match(feed[0]?.key ?? '', /^lottery:id:summer-draw:/)
    assert.equal(feed[0]?.title, 'Summer draw')
    assert.equal(feed[0]?.winnerInfo, 'Winner #42')
    assert.equal(feed[0]?.image, 'data:image/webp;base64,UklGRg==')
    assert.equal(feed[0]?.unread, true)
  })

  test('keeps global notifications independent with their own read keys', () => {
    const notification = {
      id: 'maintenance',
      title: 'Global maintenance',
      content: 'The service will restart once.',
      publishDate: '2026-08-10T08:00:00Z',
    }
    const key = `global:${getAnnouncementNotificationKey(notification)}`
    const feed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      announcements: [notification],
      globalNotifications: [notification],
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [getAnnouncementNotificationKey(notification)],
      readGlobalNotificationKeys: [key],
    })

    const timeline = feed.find((item) => item.source === 'announcement')
    const global = feed.find((item) => item.source === 'global')
    assert.equal(timeline?.unread, false)
    assert.equal(global?.key, key)
    assert.equal(global?.unread, false)
    assert.equal(global?.title, 'Global maintenance')
  })

  test('treats newly published winning information as unread', () => {
    const lottery = {
      id: 'summer-draw',
      title: 'Summer draw',
      content: 'Join before Friday.',
      winnerInfo: '',
      image: 'data:image/webp;base64,UklGRg==',
      publishDate: '2026-08-09T08:00:00Z',
    }
    const initialFeed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      announcements: [],
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [],
      lotteries: [lottery],
    })
    const updatedFeed = buildNotificationFeed({
      discountNotice: '',
      notice: '',
      announcements: [],
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [],
      readLotteryKeys: [initialFeed[0]?.key ?? ''],
      lotteries: [{ ...lottery, winnerInfo: 'Winner #42' }],
    })

    assert.notEqual(initialFeed[0]?.key, updatedFeed[0]?.key)
    assert.equal(updatedFeed[0]?.unread, true)
  })
})
