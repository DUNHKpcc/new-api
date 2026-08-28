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
import { useQuery } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo } from 'react'

import {
  buildNotificationFeed,
  getAnnouncementNotificationKey,
  getLotteryNotificationKey,
  getUnreadNotificationItems,
  migrateLegacyAnnouncementReadKeys,
  type AnnouncementNotification,
  type NotificationFeedItem,
} from '@/components/notifications/notification-feed'
import { getLotteryItems } from '@/features/lottery/api'
import { useStatus } from '@/hooks/use-status'
import { getNotice } from '@/lib/api'
import { useNotificationStore } from '@/stores/notification-store'

const NOTIFICATION_REFRESH_INTERVAL_MS = 60 * 1000

/**
 * Hook to manage all notification center sources.
 * Provides unread counts and read status management
 */
export function useNotifications() {
  // Fetch Notice from API
  const { data: noticeResponse, isLoading: noticeLoading } = useQuery({
    queryKey: ['notice'],
    queryFn: getNotice,
    staleTime: 1000 * 60 * 5, // 5 minutes
    refetchInterval: NOTIFICATION_REFRESH_INTERVAL_MS,
  })
  const { data: lotteryResponse, isLoading: lotteryLoading } = useQuery({
    queryKey: ['lottery-items'],
    queryFn: getLotteryItems,
    staleTime: 1000 * 60 * 5,
    refetchInterval: NOTIFICATION_REFRESH_INTERVAL_MS,
  })

  // Fetch Announcements from status
  const { status, loading: statusLoading } = useStatus({
    refetchInterval: NOTIFICATION_REFRESH_INTERVAL_MS,
  })
  const announcementsEnabled = status?.announcements_enabled ?? false
  const globalNotificationsEnabled =
    status?.global_notifications_enabled ?? false
  const announcements = useMemo(() => {
    if (!announcementsEnabled) return []
    return ((status?.announcements || []) as AnnouncementNotification[]).slice(
      0,
      20
    )
  }, [announcementsEnabled, status?.announcements])
  const globalNotifications = useMemo(() => {
    if (!globalNotificationsEnabled) return []
    return (
      (status?.global_notifications || []) as AnnouncementNotification[]
    ).slice(0, 20)
  }, [globalNotificationsEnabled, status?.global_notifications])
  const lotteries = useMemo(
    () => (lotteryResponse?.success ? lotteryResponse.data || [] : []),
    [lotteryResponse]
  )

  // Notification store
  const {
    lastReadDiscountNotice,
    lastReadNotice,
    readAnnouncementKeys,
    readGlobalNotificationKeys,
    readLotteryKeys,
    markDiscountNoticeRead,
    markNoticeRead,
    markAnnouncementsRead,
    markGlobalNotificationsRead,
    replaceAnnouncementReadKeys,
    markLotteriesRead,
  } = useNotificationStore()

  // Extract notice content
  const noticeContent = noticeResponse?.success
    ? (noticeResponse.data || '').trim()
    : ''
  const discountNoticeContent = status?.discount_notice?.trim() ?? ''

  useEffect(() => {
    if (statusLoading || !announcementsEnabled) return

    const migratedKeys = migrateLegacyAnnouncementReadKeys(
      readAnnouncementKeys,
      announcements
    )
    if (migratedKeys !== readAnnouncementKeys) {
      replaceAnnouncementReadKeys(migratedKeys)
    }
  }, [
    announcements,
    announcementsEnabled,
    readAnnouncementKeys,
    replaceAnnouncementReadKeys,
    statusLoading,
  ])

  const items = useMemo(
    () =>
      buildNotificationFeed({
        discountNotice: discountNoticeContent,
        notice: noticeContent,
        announcements,
        globalNotifications,
        lastReadDiscountNotice,
        lastReadNotice,
        readAnnouncementKeys,
        readGlobalNotificationKeys,
        lotteries,
        readLotteryKeys,
      }),
    [
      announcements,
      globalNotifications,
      discountNoticeContent,
      lastReadDiscountNotice,
      lastReadNotice,
      noticeContent,
      readAnnouncementKeys,
      readGlobalNotificationKeys,
      readLotteryKeys,
      lotteries,
    ]
  )
  const loading = noticeLoading || lotteryLoading || statusLoading
  const unreadItems = useMemo(
    () => getUnreadNotificationItems(items, loading),
    [items, loading]
  )

  const markAsRead = useCallback(
    (item: NotificationFeedItem) => {
      if (!item.unread) return

      if (item.source === 'discount') {
        markDiscountNoticeRead(item.content)
        return
      }

      if (item.source === 'notice') {
        markNoticeRead(item.content)
        return
      }

      if (item.source === 'lottery') {
        markLotteriesRead([item.key])
        return
      }

      if (item.source === 'global') {
        markGlobalNotificationsRead([item.key])
        return
      }

      markAnnouncementsRead([item.key])
    },
    [
      markAnnouncementsRead,
      markGlobalNotificationsRead,
      markDiscountNoticeRead,
      markLotteriesRead,
      markNoticeRead,
    ]
  )

  const markAsReadByKey = useCallback(
    (key: string) => {
      const item = items.find((candidate) => candidate.key === key)
      if (item) markAsRead(item)
    },
    [items, markAsRead]
  )

  const markAllAsRead = useCallback(() => {
    if (loading) return

    if (discountNoticeContent) {
      markDiscountNoticeRead(discountNoticeContent)
    }

    if (noticeContent) {
      markNoticeRead(noticeContent)
    }

    if (announcements.length > 0) {
      markAnnouncementsRead(
        announcements.map((announcement) =>
          getAnnouncementNotificationKey(announcement)
        )
      )
    }
    if (globalNotifications.length > 0) {
      markGlobalNotificationsRead(
        globalNotifications.map(
          (notification) =>
            `global:${getAnnouncementNotificationKey(notification)}`
        )
      )
    }
    if (lotteries.length > 0) {
      markLotteriesRead(lotteries.map(getLotteryNotificationKey))
    }
  }, [
    announcements,
    globalNotifications,
    discountNoticeContent,
    lotteries,
    markAnnouncementsRead,
    markGlobalNotificationsRead,
    markDiscountNoticeRead,
    markLotteriesRead,
    markNoticeRead,
    noticeContent,
    loading,
  ])

  return {
    items,
    loading,
    unreadItems,
    unreadCount: unreadItems.length,
    markAsRead,
    markAsReadByKey,
    markAllAsRead,
  }
}
