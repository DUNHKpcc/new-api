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
export type LotteryItem = {
  lottery?: Lottery
  id: string
  title: string
  content: string
  winnerInfo: string
  image: string
  publishDate: string
}

export type LotteryItemsResponse = {
  success: boolean
  message: string
  data?: LotteryItem[]
}

export interface Lottery {
  id: number
  title: string
  content: string
  prize: string
  winner_count: number
  draw_mode: 'full' | 'time' | 'either'
  target_count: number
  draw_at: number
  require_recharge: boolean
  min_invites: number
  required_group: string
  email_notify: boolean
  status: 'draft' | 'active' | 'drawn' | 'cancelled'
  entry_count: number
  created_at: number
  drawn_at: number
}
export interface LotteryEntry {
  id: number
  lottery_id: number
  user_id: number
  winner: boolean
  fulfilled: boolean
  email_status: string
  email_attempts: number
}
