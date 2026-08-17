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
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface NotificationState {
  // Last read discount notification content signature (full trimmed message)
  lastReadDiscountNotice: string
  // Last read Notice content signature (full trimmed message)
  lastReadNotice: string
  // Array of read announcement keys (id or content hash)
  readAnnouncementKeys: string[]
  // Array of read lottery keys (id or content hash)
  readLotteryKeys: string[]
  // Timestamp of last "Close Today" action
  closedUntilDate: string | null

  // Actions
  markDiscountNoticeRead: (noticeContent: string) => void
  markNoticeRead: (noticeContent: string) => void
  markAnnouncementsRead: (keys: string[]) => void
  replaceAnnouncementReadKeys: (keys: string[]) => void
  markLotteriesRead: (keys: string[]) => void
  setClosedUntilDate: (date: string | null) => void
  isAnnouncementRead: (key: string) => boolean
  isNoticeClosed: () => boolean
}

/**
 * Notification store for tracking read status across the notification center.
 * Persists to localStorage to maintain state across sessions
 */
export const useNotificationStore = create<NotificationState>()(
  persist(
    (set, get) => ({
      lastReadDiscountNotice: '',
      lastReadNotice: '',
      readAnnouncementKeys: [],
      readLotteryKeys: [],
      closedUntilDate: null,

      markDiscountNoticeRead: (noticeContent: string) => {
        set({ lastReadDiscountNotice: noticeContent.trim() })
      },

      markNoticeRead: (noticeContent: string) => {
        // Persist the full trimmed content so edits beyond 100 chars register
        const normalizedContent = noticeContent.trim()
        set({ lastReadNotice: normalizedContent })
      },

      markAnnouncementsRead: (keys: string[]) => {
        set((state) => ({
          readAnnouncementKeys: [
            ...new Set([...state.readAnnouncementKeys, ...keys]),
          ],
        }))
      },

      replaceAnnouncementReadKeys: (keys: string[]) => {
        set({ readAnnouncementKeys: [...new Set(keys)] })
      },

      markLotteriesRead: (keys: string[]) => {
        set((state) => ({
          readLotteryKeys: [...new Set([...state.readLotteryKeys, ...keys])],
        }))
      },

      setClosedUntilDate: (date: string | null) => {
        set({ closedUntilDate: date })
      },

      isAnnouncementRead: (key: string) => {
        return get().readAnnouncementKeys.includes(key)
      },

      isNoticeClosed: () => {
        const { closedUntilDate } = get()
        if (!closedUntilDate) return false

        const today = new Date().toDateString()
        return closedUntilDate === today
      },
    }),
    {
      name: 'notification-storage',
      partialize: (state) => ({
        lastReadDiscountNotice: state.lastReadDiscountNotice,
        lastReadNotice: state.lastReadNotice,
        readAnnouncementKeys: state.readAnnouncementKeys,
        readLotteryKeys: state.readLotteryKeys,
        closedUntilDate: state.closedUntilDate,
      }),
    }
  )
)
