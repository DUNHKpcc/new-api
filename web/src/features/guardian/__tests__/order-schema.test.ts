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
import { describe, expect, it } from 'vitest'

import { guardianOrderSchema } from '../lib/schema'

const validOrder = {
  payment_method: 'alipay',
  route: 'vps',
  os: 'macos',
  contact_type: 'telegram',
  contact: '@guardian_user',
  note: 'Need help with Realm forwarding on an existing VPS.',
}

describe('guardianOrderSchema', () => {
  it('accepts a contact handle and a plain note', () => {
    expect(guardianOrderSchema.safeParse(validOrder).success).toBe(true)
  })

  it.each([
    ['a proxy subscription link', 'vless://uuid@203.0.113.7:443?security=tls'],
    ['a private key', '-----BEGIN OPENSSH PRIVATE KEY-----\nabc'],
    ['a password line', 'root password: hunter2'],
    ['a Chinese password line', 'SSH 密码：hunter2'],
    ['an API key', 'sk-ant-api03-abcdefghijklmnopqrstuv'],
  ])('rejects a note that contains %s', (_label, note) => {
    const result = guardianOrderSchema.safeParse({ ...validOrder, note })
    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.path).toEqual(['note'])
  })

  it('rejects an empty contact after trimming', () => {
    const result = guardianOrderSchema.safeParse({
      ...validOrder,
      contact: '   ',
    })
    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe('Enter how we can reach you')
  })
})
