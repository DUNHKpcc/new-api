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
import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import type { Lottery, LotteryEntry, LotteryItemsResponse } from './types'

export async function getLotteryItems(): Promise<LotteryItemsResponse> {
  const response = await api.get<LotteryItemsResponse>('/api/lottery-items')
  return requireServerSuccess(response.data)
}

interface Response<T> {
  success: boolean
  message: string
  data: T
}
export async function listLotteries(before?: number) {
  return requireServerSuccess(
    (
      await api.get<Response<Lottery[]>>('/api/lotteries-admin', {
        params: { before },
      })
    ).data
  ).data
}
export async function saveLottery(lottery: Partial<Lottery>) {
  return requireServerSuccess(
    (await api.post<Response<Lottery>>('/api/lotteries-admin', lottery)).data
  ).data
}
export async function changeLottery(id: number, action: 'publish' | 'cancel') {
  return requireServerSuccess(
    (await api.post<Response<never>>(`/api/lotteries-admin/${id}/${action}`))
      .data
  )
}
export async function joinLottery(id: number) {
  return requireServerSuccess(
    (await api.post<Response<never>>(`/api/lotteries/${id}/join`)).data
  )
}
export async function getLotteryEntry(id: number) {
  return requireServerSuccess(
    (await api.get<Response<LotteryEntry | null>>(`/api/lotteries/${id}/self`))
      .data
  ).data
}
export async function listWinners(id: number) {
  return requireServerSuccess(
    (
      await api.get<Response<LotteryEntry[]>>(
        `/api/lotteries-admin/${id}/winners`
      )
    ).data
  ).data
}
export async function updateWinner(
  id: number,
  entry: number,
  action: 'fulfill' | 'retry'
) {
  return requireServerSuccess(
    (
      await api.put<Response<never>>(
        `/api/lotteries-admin/${id}/winners/${entry}`,
        { action }
      )
    ).data
  )
}
