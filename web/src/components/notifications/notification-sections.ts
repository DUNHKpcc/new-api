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
import type { NotificationFeedItem } from './notification-feed'

type NotificationSections = {
  notice: NotificationFeedItem[]
  timeline: NotificationFeedItem[]
  lottery: NotificationFeedItem[]
}

export function partitionNotificationItems(
  items: NotificationFeedItem[]
): NotificationSections {
  return {
    notice: items.filter(
      (item) => item.source === 'discount' || item.source === 'notice'
    ),
    timeline: items.filter((item) => item.source === 'announcement'),
    lottery: items.filter((item) => item.source === 'lottery'),
  }
}
