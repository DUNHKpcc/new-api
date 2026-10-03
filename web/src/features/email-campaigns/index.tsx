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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  DataTablePagination,
  StaticDataTable,
  useDataTable,
} from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { cancelEmailCampaign, getEmailConfig, listEmailCampaigns } from './api'
import { EmailConfigCard } from './components/config-card'
import { EmailDeliveryDialog } from './components/delivery-dialog'
import { EmailDraftDialog } from './components/draft-dialog'
import { EmailCategoryLabel, EmailStatus } from './components/email-status'
import { EmailQueueDialog } from './components/queue-dialog'
import type { EmailCampaign } from './types'

const EMPTY_CAMPAIGNS: EmailCampaign[] = []
export function EmailCampaigns() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const queryClient = useQueryClient()
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 20 })
  const [editor, setEditor] = useState<EmailCampaign | 'new' | null>(null)
  const [queueCampaign, setQueueCampaign] = useState<EmailCampaign | null>(null)
  const [cancelCampaign, setCancelCampaign] = useState<EmailCampaign | null>(
    null
  )
  const [deliveries, setDeliveries] = useState<EmailCampaign | null>(null)
  const config = useQuery({
    queryKey: ['email-config'],
    queryFn: getEmailConfig,
  })
  const campaigns = useQuery({
    queryKey: ['email-campaigns', 'list', pagination],
    queryFn: () =>
      listEmailCampaigns(pagination.pageIndex + 1, pagination.pageSize),
    refetchInterval: 10000,
  })
  const { table } = useDataTable({
    data: campaigns.data?.items ?? EMPTY_CAMPAIGNS,
    columns: [],
    totalCount: campaigns.data?.total ?? 0,
    manualPagination: true,
    columnFilters: [],
    pagination,
    onPaginationChange: setPagination,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['email-campaigns'] })
  }
  const cancel = useMutation({
    mutationFn: (id: number) => cancelEmailCampaign(id),
    onSuccess: () => {
      toast.success(t('Pending email deliveries cancelled'))
      setCancelCampaign(null)
      refresh()
    },
    onError: (error) =>
      handleServerError(error, t('Failed to cancel email campaign')),
  })
  return (
    <div className='space-y-5'>
      <EmailConfigCard />
      <TitledCard
        title={t('Email campaigns')}
        description={t(
          'Create promotional messages or platform updates, preview recipients and track delivery.'
        )}
        disableHoverEffect
        action={
          <Button onClick={() => setEditor('new')}>
            {t('Create email campaign')}
          </Button>
        }
      >
        <div className='space-y-3' aria-busy={campaigns.isFetching}>
          {campaigns.isPending && <LoadingState />}
          {campaigns.isError && (
            <ErrorState
              title={t('Failed to load email campaigns')}
              onRetry={() => void campaigns.refetch()}
            />
          )}
          {campaigns.isSuccess && (
            <>
              {campaigns.data.items.length ? (
                <StaticDataTable tableClassName='min-w-[780px]'>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Email subject')}</TableHead>
                      <TableHead>{t('Category')}</TableHead>
                      <TableHead>{t('Status')}</TableHead>
                      <TableHead>{t('Delivery progress')}</TableHead>
                      <TableHead>{t('Created')}</TableHead>
                      <TableHead>{t('Actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {campaigns.data.items.map((campaign) => (
                      <TableRow key={campaign.id}>
                        <TableCell className='max-w-64 whitespace-normal'>
                          <div className='font-medium break-words'>
                            {campaign.subject}
                          </div>
                          <div className='text-muted-foreground mt-1 text-xs'>
                            {campaign.group || t('All groups')}
                          </div>
                        </TableCell>
                        <TableCell>
                          <EmailCategoryLabel category={campaign.category} />
                        </TableCell>
                        <TableCell>
                          <EmailStatus status={campaign.status} />
                        </TableCell>
                        <TableCell className='text-xs whitespace-normal'>
                          <p>
                            {t('Accepted: {{sent}} / {{total}}', {
                              sent: formatNumber(
                                campaign.delivery_counts.sent,
                                locale
                              ),
                              total: formatNumber(
                                campaign.delivery_counts.total,
                                locale
                              ),
                            })}
                          </p>
                          <p className='text-muted-foreground'>
                            {t(
                              'Failed: {{failed}} · Uncertain: {{uncertain}}',
                              {
                                failed: formatNumber(
                                  campaign.delivery_counts.failed,
                                  locale
                                ),
                                uncertain: formatNumber(
                                  campaign.delivery_counts.uncertain,
                                  locale
                                ),
                              }
                            )}
                          </p>
                        </TableCell>
                        <TableCell className='text-xs'>
                          {formatTimestampToDate(campaign.created_at)}
                        </TableCell>
                        <TableCell>
                          <div className='flex flex-wrap gap-2'>
                            {campaign.status === 'draft' && (
                              <>
                                <Button
                                  variant='outline'
                                  size='sm'
                                  onClick={() => setEditor(campaign)}
                                >
                                  {t('Edit')}
                                </Button>
                                <Button
                                  size='sm'
                                  onClick={() => setQueueCampaign(campaign)}
                                >
                                  {t('Preview recipients')}
                                </Button>
                              </>
                            )}
                            {(campaign.status === 'queued' ||
                              campaign.status === 'sending') && (
                              <Button
                                variant='outline'
                                size='sm'
                                onClick={() => setCancelCampaign(campaign)}
                              >
                                {t('Cancel pending deliveries')}
                              </Button>
                            )}
                            {campaign.status !== 'draft' && (
                              <Button
                                variant='outline'
                                size='sm'
                                onClick={() => setDeliveries(campaign)}
                              >
                                {t('Delivery records')}
                              </Button>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </StaticDataTable>
              ) : (
                <EmptyState
                  title={t('No email campaigns yet')}
                  description={t(
                    'Create a draft to start preparing your first email campaign.'
                  )}
                  className='min-h-40'
                />
              )}
              <DataTablePagination table={table} compact />
            </>
          )}
        </div>
      </TitledCard>
      {editor && (
        <EmailDraftDialog
          campaign={editor === 'new' ? undefined : editor}
          onClose={() => setEditor(null)}
          onSaved={refresh}
        />
      )}
      {queueCampaign && (
        <EmailQueueDialog
          campaign={queueCampaign}
          onClose={() => setQueueCampaign(null)}
          onQueued={refresh}
          sendingReady={
            !!config.data?.smtp_configured &&
            !!config.data?.server_address_ready
          }
        />
      )}
      {deliveries && (
        <EmailDeliveryDialog
          campaign={
            campaigns.data?.items.find(
              (campaign) => campaign.id === deliveries.id
            ) ?? deliveries
          }
          onClose={() => setDeliveries(null)}
        />
      )}
      <ConfirmDialog
        open={!!cancelCampaign}
        onOpenChange={(open) => {
          if (!open && !cancel.isPending) setCancelCampaign(null)
        }}
        title={t('Cancel pending deliveries')}
        desc={t(
          'Cancel emails that have not started sending? Emails already accepted by SMTP cannot be recalled.'
        )}
        destructive
        confirmText={t('Confirm cancellation')}
        isLoading={cancel.isPending}
        handleConfirm={() => {
          if (cancelCampaign) cancel.mutate(cancelCampaign.id)
        }}
      />
    </div>
  )
}
