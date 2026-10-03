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

export const emailDraftSchema = z.object({
  subject: z
    .string()
    .trim()
    .min(1, 'Enter an email subject')
    .max(200, 'Email subject must be at most 200 characters'),
  body: z
    .string()
    .trim()
    .min(1, 'Enter an email message')
    .refine(
      (value) => new TextEncoder().encode(value).length <= 20000,
      'Email message must be at most 20000 UTF-8 bytes'
    ),
  category: z.enum(['promotion', 'platform']),
  group: z.string().trim().max(64),
})
export const emailRateSchema = z.object({
  rate_per_minute: z.number().int().min(1).max(120),
})
