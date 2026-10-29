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

import { GUARDIAN_MAX_FEATURES } from '../constants'
import { yuanToCents } from './price'

/**
 * Catches the credential shapes people most often paste by mistake. The order
 * form must never carry them; remote access is arranged out of band.
 */
const SECRET_PATTERN =
  /(-----BEGIN [A-Z ]*PRIVATE KEY-----|ssh-(rsa|ed25519)\s+AAAA|\b(vmess|vless|trojan|ss|ssr|hysteria2?|tuic):\/\/|\bsk-[A-Za-z0-9_-]{16,}|(\b(password|passwd|pwd)|密码|密碼)\s*[:：=])/i

export function looksLikeSecret(value: string): boolean {
  return SECRET_PATTERN.test(value)
}

const SECRET_MESSAGE = 'Do not include passwords, keys or subscription links'

export const guardianOrderSchema = z.object({
  payment_method: z.string().min(1, 'Choose a payment method'),
  route: z.enum(['local', 'vm', 'vps', 'unsure']),
  os: z.enum(['macos', 'windows', 'linux', 'other']),
  contact_type: z.enum(['telegram', 'qq', 'wechat', 'email']),
  contact: z
    .string()
    .trim()
    .min(1, 'Enter how we can reach you')
    .max(128, 'Contact must be at most 128 characters')
    .refine((value) => !looksLikeSecret(value), SECRET_MESSAGE),
  note: z
    .string()
    .trim()
    .max(1000, 'Note must be at most 1000 characters')
    .refine((value) => !looksLikeSecret(value), SECRET_MESSAGE),
})
export type GuardianOrderForm = z.infer<typeof guardianOrderSchema>

export function splitFeatureLines(value: string): string[] {
  return value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

export const guardianServiceSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, 'Enter a service name')
    .max(128, 'Service name must be at most 128 characters'),
  subtitle: z
    .string()
    .trim()
    .max(255, 'Subtitle must be at most 255 characters'),
  description: z
    .string()
    .trim()
    .max(5000, 'Description must be at most 5000 characters'),
  features: z
    .string()
    .refine(
      (value) => splitFeatureLines(value).length <= GUARDIAN_MAX_FEATURES,
      'Use at most 12 specification lines'
    )
    .refine(
      (value) => splitFeatureLines(value).every((line) => line.length <= 120),
      'Each specification line must be at most 120 characters'
    ),
  price: z
    .string()
    .refine(
      (value) => yuanToCents(value) !== null,
      'Enter a price between 0.01 and 100000 yuan'
    ),
  delivery_note: z
    .string()
    .trim()
    .max(128, 'Delivery note must be at most 128 characters'),
  badge: z.string().trim().max(32, 'Badge must be at most 32 characters'),
  route: z.enum(['any', 'local', 'vm', 'vps']),
  enabled: z.boolean(),
  sort_order: z
    .number({ message: 'Enter a whole number' })
    .int('Enter a whole number')
    .min(-9999)
    .max(9999),
})
export type GuardianServiceForm = z.infer<typeof guardianServiceSchema>

export const guardianContactSchema = guardianOrderSchema.pick({
  contact_type: true,
  contact: true,
  note: true,
})
