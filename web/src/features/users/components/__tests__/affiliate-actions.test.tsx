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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { Row } from '@tanstack/react-table'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { User } from '../../types'
import { DataTableRowActions } from '../data-table-row-actions'
import { UsersProvider } from '../users-provider'

const user: User = {
  id: 2,
  username: 'reseller-user',
  display_name: 'Reseller',
  role: 1,
  status: 1,
  quota: 0,
  used_quota: 0,
  request_count: 0,
  group: 'default',
}
let client: QueryClient

afterEach(() => {
  cleanup()
  client.clear()
  vi.restoreAllMocks()
})

it.each(['Affiliate access', 'Affiliate commissions'])(
  'opens the existing %s dialog alongside Guardian access',
  async (label) => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { items: [], total: 0 } },
    })
    client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={client}>
        <UsersProvider>
          <DataTableRowActions row={{ original: user } as Row<User>} />
        </UsersProvider>
      </QueryClientProvider>
    )
    await userEvent.click(screen.getByRole('button', { name: 'Open menu' }))
    expect(
      await screen.findByRole('menuitem', { name: 'Guardian reseller access' })
    ).toBeVisible()
    expect(
      screen.getByRole('menuitem', { name: 'Affiliate access' })
    ).toBeVisible()
    expect(
      screen.getByRole('menuitem', { name: 'Affiliate commissions' })
    ).toBeVisible()
    await userEvent.click(screen.getByRole('menuitem', { name: label }))
    expect(await screen.findByRole('dialog', { name: label })).toBeVisible()
    if (label === 'Affiliate access') {
      expect(screen.getByRole('combobox')).toHaveTextContent(
        'Follow system threshold'
      )
      await userEvent.click(screen.getByRole('combobox'))
      expect(
        await screen.findByRole('option', { name: 'Follow system threshold' })
      ).toBeVisible()
      await userEvent.click(
        screen.getByRole('option', { name: 'Follow system threshold' })
      )
      expect(screen.getByRole('textbox', { name: 'Reason' })).toBeVisible()
    } else {
      expect(get).toHaveBeenCalledWith('/api/admin/affiliate/commissions', {
        params: { agent_user_id: user.id, page: 1, page_size: 20 },
      })
    }
  }
)
