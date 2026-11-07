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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DataTablePagination,
  StaticDataTable,
  useDataTable,
} from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import { formatTimestampToDate } from '@/lib/format'

import { listGuardianAccountStatuses } from '../../api'
import { GUARDIAN_QUERY_KEYS } from '../../constants'
import { useAccountStatusLabels } from '../../hooks/use-account-status-labels'
import type { GuardianAccountStatus } from '../../types'
import { GuardianAccountStatusDialog } from './account-status-dialog'

const EMPTY_USERS: GuardianAccountStatus[] = []

export function GuardianAccountStatusPanel() {
  const { t } = useTranslation()
  const labels = useAccountStatusLabels()
  const client = useQueryClient()
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 20 })
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [editing, setEditing] = useState<GuardianAccountStatus | null>(null)
  const users = useQuery({
    queryKey: [
      ...GUARDIAN_QUERY_KEYS.adminAccountStatuses,
      pagination,
      keyword,
    ],
    queryFn: () =>
      listGuardianAccountStatuses(
        pagination.pageIndex + 1,
        pagination.pageSize,
        keyword
      ),
    refetchInterval: 30000,
  })
  const { table } = useDataTable({
    data: users.data?.items ?? EMPTY_USERS,
    columns: [],
    totalCount: users.data?.total ?? 0,
    manualPagination: true,
    columnFilters: [],
    pagination,
    onPaginationChange: setPagination,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  return (
    <TitledCard
      title={t('Purchased user account status')}
      description={t(
        'Register Claude account conditions for purchasers. Each user has one record across all Guardian orders.'
      )}
      disableHoverEffect
    >
      <form
        className='mb-4 flex gap-2'
        onSubmit={(event) => {
          event.preventDefault()
          setKeyword(keywordInput.trim())
          setPagination((previous) => ({ ...previous, pageIndex: 0 }))
        }}
      >
        <Input
          aria-label={t('Username')}
          placeholder={t('Username')}
          value={keywordInput}
          onChange={(event) => setKeywordInput(event.target.value)}
          className='max-w-64'
        />
        <Button type='submit' variant='outline'>
          {t('Search')}
        </Button>
      </form>
      {users.isPending && <LoadingState />}
      {users.isError && (
        <ErrorState
          title={t('Failed to load account status')}
          onRetry={() => void users.refetch()}
        />
      )}
      {users.isSuccess && users.data.items.length === 0 && (
        <EmptyState title={t('No purchasers found')} className='min-h-40' />
      )}
      {users.isSuccess && users.data.items.length > 0 && (
        <StaticDataTable tableClassName='min-w-[760px]'>
          <TableHeader>
            <TableRow>
              <TableHead>{t('User')}</TableHead>
              {labels.map((item) => (
                <TableHead key={item.key}>{item.label}</TableHead>
              ))}
              <TableHead>{t('Updated at')}</TableHead>
              <TableHead>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.data.items.map((user) => (
              <TableRow key={user.user_id}>
                <TableCell>
                  {user.username} (#{user.user_id})
                </TableCell>
                {labels.map((item) => {
                  let value = t('Not recorded')
                  if (user.updated_at !== 0) {
                    value = user[item.key] ? t('Yes') : t('No')
                  }
                  return <TableCell key={item.key}>{value}</TableCell>
                })}
                <TableCell>
                  {user.updated_at
                    ? formatTimestampToDate(user.updated_at)
                    : '—'}
                </TableCell>
                <TableCell>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => setEditing(user)}
                  >
                    {t('Record account status')}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </StaticDataTable>
      )}
      {users.isSuccess && <DataTablePagination table={table} compact />}
      {editing && (
        <GuardianAccountStatusDialog
          user={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            void client.invalidateQueries({
              queryKey: GUARDIAN_QUERY_KEYS.adminAccountStatuses,
            })
            void client.invalidateQueries({
              queryKey: GUARDIAN_QUERY_KEYS.accountStatistics,
            })
          }}
        />
      )}
    </TitledCard>
  )
}
