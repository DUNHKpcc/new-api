/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const PCC_AGENT_TICKET_STORAGE_PREFIX = 'pcc-agent-ticket:v1:seen'
const PCC_AGENT_TICKET_BROWSER_STORAGE_KEY = `${PCC_AGENT_TICKET_STORAGE_PREFIX}:browser`

export type TicketStorage = Pick<Storage, 'getItem' | 'setItem'>

export function getPccAgentTicketStorageKey(
  userId: number | null | undefined
): string {
  if (userId === null || userId === undefined) {
    return `${PCC_AGENT_TICKET_STORAGE_PREFIX}:visitor`
  }

  return `${PCC_AGENT_TICKET_STORAGE_PREFIX}:user:${userId}`
}

export function getPccAgentTicketBrowserStorageKey(): string {
  return PCC_AGENT_TICKET_BROWSER_STORAGE_KEY
}

export function readPccAgentTicketSeen(
  storage: TicketStorage | null | undefined,
  storageKey: string
): boolean {
  if (!storage) return false

  try {
    return storage.getItem(storageKey) === 'true'
  } catch {
    return false
  }
}

export function markPccAgentTicketSeen(
  storage: TicketStorage | null | undefined,
  storageKey: string
): boolean {
  if (!storage) return false

  try {
    storage.setItem(storageKey, 'true')
    return true
  } catch {
    return false
  }
}
