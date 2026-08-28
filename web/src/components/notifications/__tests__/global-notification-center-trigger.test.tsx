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

const { GlobalNotificationCenter } =
  await import('../global-notification-center')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: false,
  resources: { en: { translation: {} } },
})

function renderCenter() {
  return renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <GlobalNotificationCenter />
    </I18nextProvider>
  )
}

describe('floating notification center trigger', () => {
  after(() => mock.restore())

  test('reuses the unread bell alert animation class', () => {
    notificationItems = [
      {
        key: 'notice',
        source: 'notice',
        content: 'Scheduled maintenance',
        unread: true,
      },
    ]

    const markup = renderCenter()

    assert.match(markup, /data-floating-action="notification"/)
    assert.match(markup, /text-destructive topbar-alert-icon/)
  })
})
