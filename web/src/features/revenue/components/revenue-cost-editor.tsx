/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ChevronDown, Pencil, Plus, ReceiptText } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatMinorCurrency, formatTimestampToDate } from '@/lib/format'

import type {
  RevenueCostCategory,
  RevenueCostMutation,
  RevenueCostRecord,
} from '../types'
import { RevenueCostDialog } from './revenue-cost-dialog'

type RevenueCostEditorProps = {
  month: string
  currency: string
  records: RevenueCostRecord[]
  totalAmountMinor: string
  loading: boolean
  error: boolean
  onSaved: (
    payload: RevenueCostMutation,
    record: RevenueCostRecord | null
  ) => Promise<void>
}

function categoryName(
  category: RevenueCostCategory,
  translate: (key: string) => string
): string {
  const labels: Record<RevenueCostCategory, string> = {
    server: 'Server',
    upstream: 'Upstream resources',
    account: 'Account',
    other: 'Other',
  }
  return translate(labels[category])
}

export function RevenueCostEditor(props: RevenueCostEditorProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRecord, setEditingRecord] = useState<RevenueCostRecord | null>(
    null
  )

  const openCreate = () => {
    setEditingRecord(null)
    setDialogOpen(true)
  }

  const openEdit = (record: RevenueCostRecord) => {
    setEditingRecord(record)
    setDialogOpen(true)
  }

  let recordsContent: ReactNode
  if (props.loading) {
    recordsContent = (
      <div className='space-y-2'>
        {['one', 'two', 'three'].map((key) => (
          <Skeleton key={key} className='h-10 w-full' />
        ))}
      </div>
    )
  } else if (props.records.length === 0) {
    recordsContent = (
      <div className='text-muted-foreground rounded-md border border-dashed p-8 text-center text-sm'>
        {t('No cost records for this month')}
      </div>
    )
  } else {
    recordsContent = (
      <div className='overflow-x-auto'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Cost category')}</TableHead>
              <TableHead>{t('Details')}</TableHead>
              <TableHead>{t('Amount')}</TableHead>
              <TableHead>{t('Updated')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {props.records.map((record) => (
              <TableRow key={record.id}>
                <TableCell className='font-medium'>
                  {categoryName(record.category, t)}
                </TableCell>
                <TableCell>
                  <div className='max-w-72 truncate'>
                    {record.description || '-'}
                  </div>
                </TableCell>
                <TableCell className='font-semibold tabular-nums'>
                  {formatMinorCurrency(record.amount_minor, record.currency)}
                </TableCell>
                <TableCell className='text-muted-foreground text-xs'>
                  {record.update_time
                    ? formatTimestampToDate(record.update_time)
                    : '-'}
                </TableCell>
                <TableCell>
                  <div className='flex justify-end'>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      onClick={() => openEdit(record)}
                      aria-label={t('Edit')}
                      title={t('Edit')}
                    >
                      <Pencil />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    )
  }

  return (
    <>
      <Collapsible
        open={expanded}
        onOpenChange={setExpanded}
        data-revenue-cost-editor='true'
      >
        <Card>
          <CardHeader className='flex flex-col items-stretch gap-3 sm:flex-row sm:items-start sm:justify-between'>
            <div className='min-w-0 space-y-1'>
              <CardTitle className='flex items-center gap-2 text-base'>
                <ReceiptText className='text-muted-foreground h-4 w-4' />
                {t('Operating costs')}
              </CardTitle>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Track server, upstream, account, and other costs for the selected month.'
                )}
              </p>
            </div>
            <div className='flex items-center justify-end gap-1'>
              <div className='bg-muted/50 rounded-md px-3 py-1.5 text-right'>
                <span className='text-muted-foreground block text-[0.7rem]'>
                  {t('Total cost')}
                </span>
                <span className='text-sm font-semibold tabular-nums'>
                  {formatMinorCurrency(props.totalAmountMinor, props.currency)}
                </span>
              </div>
              <Button type='button' size='sm' onClick={openCreate}>
                <Plus />
                {t('Add cost')}
              </Button>
              <CollapsibleTrigger
                render={
                  <Button
                    type='button'
                    size='icon-sm'
                    variant='ghost'
                    aria-label={t(expanded ? 'Collapse' : 'Expand')}
                    title={t(expanded ? 'Collapse' : 'Expand')}
                  />
                }
              >
                <ChevronDown
                  className={`transition-transform ${expanded ? 'rotate-180' : ''}`}
                />
              </CollapsibleTrigger>
            </div>
          </CardHeader>
          <CollapsibleContent keepMounted>
            <CardContent className='space-y-4'>
              {props.error ? (
                <p className='text-destructive text-sm'>
                  {t('Failed to load cost records')}
                </p>
              ) : null}
              <div
                className='max-h-56 overflow-auto'
                data-revenue-cost-scroll='true'
              >
                {recordsContent}
              </div>
            </CardContent>
          </CollapsibleContent>
        </Card>
      </Collapsible>
      <RevenueCostDialog
        open={dialogOpen}
        month={props.month}
        currency={props.currency}
        record={editingRecord}
        onOpenChange={setDialogOpen}
        onSaved={(payload) => props.onSaved(payload, editingRecord)}
      />
    </>
  )
}
