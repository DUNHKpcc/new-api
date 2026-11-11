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
import { useMutation, useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DataTablePage,
  DataTableBulkActions,
  useDataTable,
} from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { listAdminGuardianServices } from '../../api'
import {
  resellerRead,
  resellerWrite,
  type GuardianResellerSummary,
} from '../../reseller-api'
import { ResellerOffersPanel } from '../reseller/offers'
import {
  TierSelect,
  TierConfirmation,
  type TierChange,
  type TierPreview,
} from './tier-prices'

export function GuardianResellerList() {
  const { t } = useTranslation()
  const [tier, setTier] = useState(-1)
  const [source, setSource] = useState('')
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [serviceIds, setServiceIds] = useState<number[]>([])
  const services = useQuery({
    queryKey: ['guardian', 'admin-services'],
    queryFn: listAdminGuardianServices,
    enabled: restoreOpen,
  })
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [targetTier, setTargetTier] = useState(1)
  const [detail, setDetail] = useState<GuardianResellerSummary | null>(null)
  const [confirmation, setConfirmation] = useState<{
    change: TierChange
    preview: TierPreview
  } | null>(null)
  const query = useQuery({
    queryKey: ['guardian', 'admin', 'resellers'],
    queryFn: () => resellerRead<GuardianResellerSummary[]>('/admin/resellers'),
  })
  const preview = useMutation({
    mutationFn: (change: TierChange) =>
      resellerWrite<TierPreview>('/admin/tier-supplies', change, 'post'),
    onSuccess: (result, change) => setConfirmation({ change, preview: result }),
  })
  const data = useMemo(
    () =>
      (query.data ?? []).filter(
        (r) =>
          (tier === -1 || (r.tier ?? 0) === tier) &&
          (!source ||
            (source === 'independent'
              ? (r.independent_products ?? 0) > 0
              : (r.tier ?? 0) > 0 && (r.independent_products ?? 0) === 0)) &&
          (!status || r.status === status) &&
          `${r.username} ${r.display_name} ${r.user_id}`
            .toLowerCase()
            .includes(search.toLowerCase())
      ),
    [query.data, tier, source, status, search]
  )
  const columns = useMemo<ColumnDef<GuardianResellerSummary>[]>(
    () => [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            aria-label={t('Select all')}
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            aria-label={`${t('Select')} ${row.original.username}`}
            checked={row.getIsSelected()}
            onCheckedChange={(v) => row.toggleSelected(!!v)}
          />
        ),
      },
      {
        accessorKey: 'username',
        header: t('Reseller'),
        cell: ({ row }) => (
          <span>
            {row.original.display_name || row.original.username}
            <small className='block'>
              {row.original.username} · #{row.original.user_id}
            </small>
          </span>
        ),
      },
      {
        accessorKey: 'tier',
        header: t('Reseller tier'),
        cell: ({ row }) =>
          row.original.tier
            ? t('Tier {{tier}}', { tier: row.original.tier })
            : t('Unassigned'),
      },
      {
        accessorKey: 'status',
        header: t('Status'),
        cell: ({ row }) =>
          row.original.status === 'active' ? t('Active') : t('Suspended'),
      },
      {
        accessorKey: 'independent_products',
        header: t('Independent prices'),
        cell: ({ row }) => row.original.independent_products ?? 0,
      },
      {
        accessorKey: 'unavailable_products',
        header: t('Retail price issues'),
        cell: ({ row }) => row.original.unavailable_products ?? 0,
      },
      {
        id: 'actions',
        header: t('Actions'),
        cell: ({ row }) => (
          <Button variant='outline' onClick={() => setDetail(row.original)}>
            {t('Products and supply prices')}
          </Button>
        ),
      },
    ],
    [t]
  )
  const { table } = useDataTable({
    data,
    columns,
    enableRowSelection: true,
    getRowId: (r) => String(r.user_id),
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  return (
    <div className='space-y-4'>
      {query.isError && <ErrorState onRetry={() => void query.refetch()} />}
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={query.isPending}
        paginationInFooter={false}
        emptyTitle={t(
          'No resellers yet. Enable Guardian reseller access in User management first.'
        )}
        showMobileBulkActions
        mobileProps={{ enableRowSelection: true }}
        toolbar={
          <div className='flex flex-wrap gap-3'>
            <Input
              aria-label={t('Search resellers')}
              placeholder={t('Search resellers')}
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                table.resetRowSelection()
                table.setPageIndex(0)
              }}
            />
            <NativeSelect
              aria-label={t('Supply source')}
              value={source}
              onChange={(e) => {
                setSource(e.target.value)
                table.resetRowSelection()
                table.setPageIndex(0)
              }}
            >
              <NativeSelectOption value=''>
                {t('All sources')}
              </NativeSelectOption>
              <NativeSelectOption value='independent'>
                {t('Independent prices')}
              </NativeSelectOption>
              <NativeSelectOption value='tier'>
                {t('Following tier')}
              </NativeSelectOption>
            </NativeSelect>
            <TierSelect
              all
              value={tier}
              onChange={(v) => {
                setTier(v)
                table.resetRowSelection()
                table.setPageIndex(0)
              }}
            />
            <NativeSelect
              aria-label={t('Status')}
              value={status}
              onChange={(e) => {
                setStatus(e.target.value)
                table.resetRowSelection()
                table.setPageIndex(0)
              }}
            >
              <NativeSelectOption value=''>
                {t('All statuses')}
              </NativeSelectOption>
              <NativeSelectOption value='active'>
                {t('Active')}
              </NativeSelectOption>
              <NativeSelectOption value='suspended'>
                {t('Suspended')}
              </NativeSelectOption>
            </NativeSelect>
          </div>
        }
        bulkActions={
          <DataTableBulkActions
            table={table}
            entityName={t('Resellers')}
            placement='inline'
          >
            <TierSelect value={targetTier} onChange={setTargetTier} />
            <Button
              disabled={preview.isPending}
              onClick={() =>
                preview.mutate({
                  tier: targetTier,
                  reseller_ids: table
                    .getFilteredSelectedRowModel()
                    .rows.map((r) => r.original.user_id),
                })
              }
            >
              {t('Set tier')}
            </Button>
            <Button
              variant='outline'
              onClick={() => {
                setServiceIds([])
                setRestoreOpen(true)
              }}
            >
              {t('Restore tier prices')}
            </Button>
          </DataTableBulkActions>
        }
      />
      {restoreOpen && (
        <Dialog
          open
          title={t('Restore tier prices')}
          onOpenChange={(open) => {
            if (!open) setRestoreOpen(false)
          }}
        >
          <p className='mb-3'>
            {t('Selected independent prices will be replaced by tier prices.')}
          </p>
          {services.isError && (
            <ErrorState onRetry={() => void services.refetch()} />
          )}
          <div className='space-y-3'>
            {services.data?.map((s) => (
              <label key={s.id} className='flex items-center gap-2'>
                <Checkbox
                  checked={serviceIds.includes(s.id)}
                  onCheckedChange={(v) =>
                    setServiceIds(
                      v
                        ? [...serviceIds, s.id]
                        : serviceIds.filter((id) => id !== s.id)
                    )
                  }
                />
                {s.title}
              </label>
            ))}
          </div>
          <Button
            className='mt-4'
            disabled={!serviceIds.length || preview.isPending}
            onClick={() => {
              preview.mutate(
                {
                  tier: targetTier,
                  reseller_ids: table
                    .getFilteredSelectedRowModel()
                    .rows.map((r) => r.original.user_id),
                  service_ids: serviceIds,
                  restore: true,
                },
                { onSuccess: () => setRestoreOpen(false) }
              )
            }}
          >
            {t('Preview changes')}
          </Button>
        </Dialog>
      )}
      {detail && (
        <Dialog
          open
          title={detail.display_name || detail.username}
          description={
            detail.tier
              ? t('Tier {{tier}}', { tier: detail.tier })
              : t('Unassigned')
          }
          contentClassName='sm:max-w-4xl'
          onOpenChange={(open) => {
            if (!open) setDetail(null)
          }}
        >
          <ResellerOffersPanel userId={detail.user_id} tier={detail.tier} />
        </Dialog>
      )}
      {confirmation && (
        <TierConfirmation
          {...confirmation}
          onClose={() => setConfirmation(null)}
          onSaved={() => {
            setConfirmation(null)
            table.resetRowSelection()
          }}
        />
      )}
    </div>
  )
}
