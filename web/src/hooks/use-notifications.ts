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
import { useCallback, useMemo } from 'react'

import {
  buildNotificationFeed,
  getAnnouncementNotificationKey,
  getLotteryNotificationKey,
  type AnnouncementNotification,
  type NotificationFeedItem,
} from '@/components/notifications/notification-feed'
import { getLotteryItems } from '@/features/lottery/api'
import { useStatus } from '@/hooks/use-status'
import { getNotice } from '@/lib/api'
import { useNotificationStore } from '@/stores/notification-store'

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
  })
  const { data: lotteryResponse, isLoading: lotteryLoading } = useQuery({
    queryKey: ['lottery-items'],
    queryFn: getLotteryItems,
    staleTime: 1000 * 60 * 5,
  })

  // Fetch Announcements from status
  const { status, loading: statusLoading } = useStatus()
  const announcementsEnabled = status?.announcements_enabled ?? false
  const announcements = useMemo(() => {
    if (!announcementsEnabled) return []
    return ((status?.announcements || []) as AnnouncementNotification[]).slice(
      0,
      20
    )
  }, [announcementsEnabled, status?.announcements])
  const lotteries = useMemo(
    () => (lotteryResponse?.success ? lotteryResponse.data || [] : []),
    [lotteryResponse]
  )

  // Notification store
  const {
    lastReadDiscountNotice,
    lastReadNotice,
    readAnnouncementKeys,
    readLotteryKeys,
    markDiscountNoticeRead,
    markNoticeRead,
    markAnnouncementsRead,
    markLotteriesRead,
  } = useNotificationStore()

  // Extract notice content
  const noticeContent = noticeResponse?.success
    ? (noticeResponse.data || '').trim()
    : ''
  const discountNoticeContent = status?.discount_notice?.trim() ?? ''

  const items = useMemo(
    () =>
      buildNotificationFeed({
        discountNotice: discountNoticeContent,
        notice: noticeContent,
        announcements,
        lastReadDiscountNotice,
        lastReadNotice,
        readAnnouncementKeys,
        lotteries,
        readLotteryKeys,
      }),
    [
      announcements,
      discountNoticeContent,
      lastReadDiscountNotice,
      lastReadNotice,
      noticeContent,
      readAnnouncementKeys,
      readLotteryKeys,
      lotteries,
    ]
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

      markAnnouncementsRead([item.key])
    },
    [
      markAnnouncementsRead,
      markDiscountNoticeRead,
      markLotteriesRead,
      markNoticeRead,
    ]
  )

  const markAllAsRead = useCallback(() => {
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
    if (lotteries.length > 0) {
      markLotteriesRead(lotteries.map(getLotteryNotificationKey))
    }
  }, [
    announcements,
    discountNoticeContent,
    lotteries,
    markAnnouncementsRead,
    markDiscountNoticeRead,
    markLotteriesRead,
    markNoticeRead,
    noticeContent,
  ])

  return {
    items,
    loading: noticeLoading || lotteryLoading || statusLoading,
    unreadCount: items.filter((item) => item.unread).length,
    markAsRead,
    markAllAsRead,
  }
}
