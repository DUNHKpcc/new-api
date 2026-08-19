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
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import {
  Ban,
  ChevronLeft,
  ChevronRight,
  Pencil,
  Plus,
  Search,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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

import { getExternalRevenue, voidExternalRevenue } from '../api'
import type { ExternalRevenueRecord } from '../types'
import { ExternalRevenueDialog } from './external-revenue-dialog'

function sourceName(
  record: ExternalRevenueRecord,
  translate: (key: string) => string
) {
  if (record.source === 'other') return record.source_label
  const keys = {
    xianyu: 'Xianyu',
    wechat: 'WeChat',
    alipay: 'Alipay',
  }
  return translate(keys[record.source])
}

export function ExternalRevenueList() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [source, setSource] = useState('all')
  const [status, setStatus] = useState('active')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRecord, setEditingRecord] =
    useState<ExternalRevenueRecord | null>(null)
  const [voidingRecord, setVoidingRecord] =
    useState<ExternalRevenueRecord | null>(null)
  const [voiding, setVoiding] = useState(false)
  const query = useQuery({
    queryKey: ['revenue', 'external-list', page, keyword, source, status],
    queryFn: () =>
      getExternalRevenue({
        page,
        pageSize: 20,
        keyword,
        source: source === 'all' ? undefined : source,
        status: status === 'all' ? undefined : status,
      }),
    placeholderData: keepPreviousData,
  })
  const records = query.data?.data?.items ?? []
  const total = query.data?.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / 20))

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['revenue'] })
  }

  const confirmVoid = async () => {
    if (!voidingRecord) return
    setVoiding(true)
    try {
      await voidExternalRevenue(voidingRecord.id, voidingRecord.version)
      toast.success(t('External revenue voided'))
      setVoidingRecord(null)
      refresh()
    } catch {
      // The shared API client presents server and network errors.
    } finally {
      setVoiding(false)
    }
  }

  let tableContent: ReactNode
  if (query.isLoading) {
    tableContent = (
      <div className='space-y-2 p-4'>
        {['one', 'two', 'three', 'four', 'five', 'six'].map((key) => (
          <Skeleton key={key} className='h-12 w-full' />
        ))}
      </div>
    )
  } else if (query.isError) {
    tableContent = (
      <div className='text-destructive p-8 text-center text-sm'>
        {t('Failed to load external revenue')}
      </div>
    )
  } else if (records.length === 0) {
    tableContent = (
      <div className='text-muted-foreground p-12 text-center text-sm'>
        {t('No external revenue records')}
      </div>
    )
  } else {
    tableContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Source')}</TableHead>
            <TableHead>{t('External order number')}</TableHead>
            <TableHead>{t('Amount')}</TableHead>
            <TableHead>{t('Received time')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead className='text-right'>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {records.map((record) => (
            <TableRow key={record.id}>
              <TableCell className='font-medium'>
                {sourceName(record, t)}
              </TableCell>
              <TableCell>
                <div className='max-w-48 truncate font-mono text-xs'>
                  {record.external_order_no || '-'}
                </div>
              </TableCell>
              <TableCell className='font-semibold'>
                {formatMinorCurrency(record.amount_minor, record.currency)}
              </TableCell>
              <TableCell>{formatTimestampToDate(record.occurred_at)}</TableCell>
              <TableCell>
                <Badge
                  variant={record.status === 'active' ? 'secondary' : 'outline'}
                >
                  {record.status === 'active' ? t('Active') : t('Voided')}
                </Badge>
              </TableCell>
              <TableCell>
                <div className='flex justify-end gap-1'>
                  <Button
                    size='icon-sm'
                    variant='ghost'
                    disabled={record.status === 'voided'}
                    onClick={() => {
                      setEditingRecord(record)
                      setDialogOpen(true)
                    }}
                    aria-label={t('Edit')}
                    title={t('Edit')}
                  >
                    <Pencil />
                  </Button>
                  <Button
                    size='icon-sm'
                    variant='ghost'
                    disabled={record.status === 'voided'}
                    onClick={() => setVoidingRecord(record)}
                    aria-label={t('Void')}
                    title={t('Void')}
                  >
                    <Ban />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-col gap-2 lg:flex-row lg:items-center'>
        <form
          className='relative flex-1'
          onSubmit={(event) => {
            event.preventDefault()
            setPage(1)
            setKeyword(searchInput.trim())
          }}
        >
          <Search className='text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
          <Input
            className='pl-9'
            value={searchInput}
            onChange={(event) => setSearchInput(event.currentTarget.value)}
            placeholder={t('Search order number or note')}
          />
        </form>
        <div className='grid grid-cols-2 gap-2 sm:flex'>
          <Select
            value={source}
            onValueChange={(value) => {
              if (!value) return
              setPage(1)
              setSource(value)
            }}
          >
            <SelectTrigger className='w-full sm:w-36' aria-label={t('Source')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>{t('All sources')}</SelectItem>
              <SelectItem value='xianyu'>{t('Xianyu')}</SelectItem>
              <SelectItem value='wechat'>{t('WeChat')}</SelectItem>
              <SelectItem value='alipay'>{t('Alipay')}</SelectItem>
              <SelectItem value='other'>{t('Other')}</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={status}
            onValueChange={(value) => {
              if (!value) return
              setPage(1)
              setStatus(value)
            }}
          >
            <SelectTrigger className='w-full sm:w-32' aria-label={t('Status')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>{t('All statuses')}</SelectItem>
              <SelectItem value='active'>{t('Active')}</SelectItem>
              <SelectItem value='voided'>{t('Voided')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button
          onClick={() => {
            setEditingRecord(null)
            setDialogOpen(true)
          }}
        >
          <Plus />
          {t('Add external revenue')}
        </Button>
      </div>

      <Card className='gap-0 py-0'>
        <CardContent className='px-0'>{tableContent}</CardContent>
      </Card>

      <div className='flex items-center justify-between gap-3'>
        <p className='text-muted-foreground text-xs'>
          {t('{{count}} records', { count: total })}
        </p>
        <div className='flex items-center gap-2'>
          <Button
            size='icon-sm'
            variant='outline'
            disabled={page <= 1}
            onClick={() => setPage((value) => value - 1)}
            aria-label={t('Previous page')}
          >
            <ChevronLeft />
          </Button>
          <span className='text-muted-foreground min-w-16 text-center text-xs'>
            {page} / {totalPages}
          </span>
          <Button
            size='icon-sm'
            variant='outline'
            disabled={page >= totalPages}
            onClick={() => setPage((value) => value + 1)}
            aria-label={t('Next page')}
          >
            <ChevronRight />
          </Button>
        </div>
      </div>

      <ExternalRevenueDialog
        open={dialogOpen}
        record={editingRecord}
        onOpenChange={setDialogOpen}
        onSaved={refresh}
      />
      <AlertDialog
        open={Boolean(voidingRecord)}
        onOpenChange={(open) => !open && setVoidingRecord(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Void external revenue?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'The record will be kept for audit history but excluded from revenue totals.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={voiding}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction disabled={voiding} onClick={confirmVoid}>
              {t('Void')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
