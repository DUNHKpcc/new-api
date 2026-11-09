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
import { z } from 'zod'

export const resellerMoneySchema = z
  .string()
  .regex(/^(0|[1-9]\d{0,5})(\.\d{1,2})?$/, 'Enter a valid amount in CNY')
  .refine((v) => Number(v) <= 100000, 'Amount is too large')
export const withdrawalSchema = z.object({
  amount: resellerMoneySchema.refine(
    (v) => Number(v) >= 1,
    'Minimum withdrawal is CNY 1'
  ),
  account: z.string().trim().min(1, 'Required').max(128),
  payee_name: z.string().trim().min(1, 'Required').max(128),
})
export type WithdrawalForm = z.infer<typeof withdrawalSchema>
