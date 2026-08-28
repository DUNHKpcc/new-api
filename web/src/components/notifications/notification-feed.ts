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
import type { LotteryItem } from '@/features/lottery/types'

export type AnnouncementNotification = {
  id?: number | string
  kind?: string
  type?: string
  title?: string
  content?: string
  extra?: string
  image?: string
  publishDate?: string | Date
}

export type NotificationFeedItem = {
  key: string
  source: 'discount' | 'notice' | 'announcement' | 'global' | 'lottery'
  kind?: string
  type?: string
  title?: string
  content: string
  extra?: string
  winnerInfo?: string
  image?: string
  publishDate?: string | Date
  unread: boolean
}

type BuildNotificationFeedOptions = {
  discountNotice: string
  notice: string
  announcements: AnnouncementNotification[]
  globalNotifications?: AnnouncementNotification[]
  lastReadDiscountNotice: string
  lastReadNotice: string
  readAnnouncementKeys: string[]
  readGlobalNotificationKeys?: string[]
  lotteries?: LotteryItem[]
  readLotteryKeys?: string[]
}

const notificationSourcePriority: Record<
  NotificationFeedItem['source'],
  number
> = {
  discount: 0,
  notice: 1,
  lottery: 2,
  announcement: 3,
  global: 2,
}

export function getLotteryNotificationKey(lottery: LotteryItem): string {
  const fingerprint = hashString(
    JSON.stringify({
      title: lottery.title.trim(),
      content: lottery.content.trim(),
      winnerInfo: lottery.winnerInfo.trim(),
      publishDate: lottery.publishDate,
    })
  )
  if (lottery.id) return `lottery:id:${lottery.id}:${fingerprint}`

  return `lottery:hash:${fingerprint}`
}

function hashString(input: string): string {
  let hash = 0
  if (!input) return '0'

  for (let index = 0; index < input.length; index += 1) {
    const character = input.charCodeAt(index)
    hash = (hash << 5) - hash + character
    hash |= 0
  }

  return hash.toString(36)
}

export function getAnnouncementNotificationKey(
  announcement: AnnouncementNotification
): string {
  const fingerprint = JSON.stringify({
    kind: announcement.kind ?? '',
    publishDate: announcement.publishDate ?? '',
    content: announcement.content?.trim() ?? '',
    extra: announcement.extra?.trim() ?? '',
    image: announcement.image ?? '',
    type: announcement.type ?? '',
  })
  const revision = hashString(fingerprint)

  if (announcement.id !== undefined && announcement.id !== null) {
    return `id:${announcement.id}:${revision}`
  }

  return `hash:${revision}`
}

function getLegacyAnnouncementNotificationKey(
  announcement: AnnouncementNotification
): string | null {
  if (announcement.id === undefined || announcement.id === null) return null
  return `id:${announcement.id}`
}

export function migrateLegacyAnnouncementReadKeys(
  readKeys: string[],
  announcements: AnnouncementNotification[]
): string[] {
  const legacyKeys = new Set(readKeys.filter((key) => /^id:[^:]+$/.test(key)))
  if (legacyKeys.size === 0) return readKeys

  const migratedKeys = new Set(readKeys.filter((key) => !legacyKeys.has(key)))
  for (const announcement of announcements) {
    const legacyKey = getLegacyAnnouncementNotificationKey(announcement)
    if (legacyKey && legacyKeys.has(legacyKey)) {
      migratedKeys.add(getAnnouncementNotificationKey(announcement))
    }
  }

  return [...migratedKeys]
}

export function getNotificationPreview(
  content: string,
  maxLength: number = 140
): string {
  const plainText = content
    .replaceAll(/<[^>]*>/g, ' ')
    .replaceAll(/[`#*_~>[\]()]/g, ' ')
    .replaceAll(/\s+/g, ' ')
    .trim()

  if (plainText.length <= maxLength) return plainText
  return `${plainText.slice(0, maxLength).trimEnd()}...`
}

export function getUnreadNotificationItems(
  items: readonly NotificationFeedItem[],
  loading: boolean
): NotificationFeedItem[] {
  if (loading) return []
  return items.filter((item) => item.unread)
}

export function buildNotificationFeed(
  options: BuildNotificationFeedOptions
): NotificationFeedItem[] {
  const items: Array<NotificationFeedItem & { originalIndex: number }> = []
  const normalizedDiscountNotice = options.discountNotice.trim()
  const normalizedNotice = options.notice.trim()

  if (normalizedDiscountNotice) {
    items.push({
      key: 'discount',
      source: 'discount',
      content: normalizedDiscountNotice,
      unread: normalizedDiscountNotice !== options.lastReadDiscountNotice,
      originalIndex: items.length,
    })
  }

  if (normalizedNotice) {
    items.push({
      key: 'notice',
      source: 'notice',
      content: normalizedNotice,
      unread: normalizedNotice !== options.lastReadNotice,
      originalIndex: items.length,
    })
  }

  const readAnnouncementKeys = new Set(options.readAnnouncementKeys)
  for (const announcement of options.announcements) {
    const key = getAnnouncementNotificationKey(announcement)
    const legacyKey = getLegacyAnnouncementNotificationKey(announcement)
    items.push({
      key,
      source: 'announcement',
      type: announcement.type,
      title: announcement.title,
      content: announcement.content?.trim() ?? '',
      extra: announcement.extra?.trim() || undefined,
      image: announcement.image,
      publishDate: announcement.publishDate,
      unread:
        !readAnnouncementKeys.has(key) &&
        (!legacyKey || !readAnnouncementKeys.has(legacyKey)),
      originalIndex: items.length,
    })
  }

  const readGlobalNotificationKeys = new Set(
    options.readGlobalNotificationKeys ?? []
  )
  for (const notification of options.globalNotifications ?? []) {
    const key = `global:${getAnnouncementNotificationKey(notification)}`
    items.push({
      key,
      source: 'global',
      kind: notification.kind,
      type: notification.type,
      title: notification.title,
      content: notification.content?.trim() ?? '',
      extra: notification.extra?.trim() || undefined,
      image: notification.image,
      publishDate: notification.publishDate,
      unread: !readGlobalNotificationKeys.has(key),
      originalIndex: items.length,
    })
  }

  const readLotteryKeys = new Set(options.readLotteryKeys ?? [])
  for (const lottery of options.lotteries ?? []) {
    const key = getLotteryNotificationKey(lottery)
    items.push({
      key,
      source: 'lottery',
      title: lottery.title.trim(),
      content: lottery.content.trim(),
      winnerInfo: lottery.winnerInfo.trim() || undefined,
      image: lottery.image,
      publishDate: lottery.publishDate,
      unread: !readLotteryKeys.has(key),
      originalIndex: items.length,
    })
  }

  items.sort((left, right) => {
    if (left.unread !== right.unread) return left.unread ? -1 : 1
    if (left.source !== right.source) {
      return (
        notificationSourcePriority[left.source] -
        notificationSourcePriority[right.source]
      )
    }

    const leftTime = left.publishDate
      ? new Date(left.publishDate).getTime()
      : Number.NaN
    const rightTime = right.publishDate
      ? new Date(right.publishDate).getTime()
      : Number.NaN
    if (Number.isFinite(leftTime) && Number.isFinite(rightTime)) {
      return rightTime - leftTime
    }

    return left.originalIndex - right.originalIndex
  })

  return items.map((item) => ({
    key: item.key,
    source: item.source,
    kind: item.kind,
    type: item.type,
    title: item.title,
    content: item.content,
    extra: item.extra,
    winnerInfo: item.winnerInfo,
    image: item.image,
    publishDate: item.publishDate,
    unread: item.unread,
  }))
}
