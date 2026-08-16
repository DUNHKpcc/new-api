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

import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import type { NotificationFeedItem } from '../notification-feed'

const { mock } = await import(`bun:${'test'}`)

let notificationItems: NotificationFeedItem[] = []

mock.module('@/hooks/use-notifications', () => ({
  useNotifications: () => ({
    items: notificationItems,
    loading: false,
    unreadItems: notificationItems.filter((item) => item.unread),
    unreadCount: notificationItems.filter((item) => item.unread).length,
    markAsRead: () => undefined,
    markAsReadByKey: () => undefined,
    markAllAsRead: () => undefined,
  }),
}))

const { NotificationPopover } = await import('../../notification-popover')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: false,
  resources: { en: { translation: {} } },
})

function renderNotificationPopover() {
  return renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <NotificationPopover />
    </I18nextProvider>
  )
}

describe('notification popover topbar trigger', () => {
  after(() => mock.restore())

  test('keeps notification and lottery icons visible at the same time', () => {
    notificationItems = [
      {
        key: 'lottery',
        source: 'lottery',
        title: 'Summer draw',
        content: 'Join now',
        unread: true,
      },
    ]

    const markup = renderNotificationPopover()

    assert.match(markup, /data-notification-entry="header"/)
    assert.match(markup, /aria-label="System Announcements"/)
    assert.match(markup, /data-lottery-active="true"/)
    assert.match(markup, /data-lottery-unread="true"/)
    assert.match(markup, /data-notification-icon="true"/)
    assert.match(markup, /data-lottery-icon="true"/)
    assert.match(markup, /app-header-lottery-alert/)
    assert.match(markup, />Lottery</)
  })

  test('keeps the quiet notification action without lottery content', () => {
    notificationItems = []

    const markup = renderNotificationPopover()

    assert.match(markup, /data-notification-entry="header"/)
    assert.match(markup, /data-notification-icon="true"/)
    assert.doesNotMatch(markup, /data-lottery-icon/)
    assert.doesNotMatch(markup, /data-lottery-active/)
    assert.doesNotMatch(markup, /app-header-lottery-alert/)
    assert.match(markup, /aria-label="System Announcements"/)
  })
})
