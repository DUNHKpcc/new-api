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
import { Link } from '@tanstack/react-router'
import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { CopyButton } from '@/components/copy-button'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'

import { GuardianShell } from './components/guardian-shell'
import { useGuardianLabels } from './hooks/use-guardian-labels'
import { buildReceiptDocument } from './lib/receipt-document'
import { resellerRead, resellerWrite } from './reseller-api'
import type { GuardianReceipt } from './types'

export function GuardianReceiptPage(props: {
  orderId: number
  admin?: boolean
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const path = `${props.admin ? '/admin' : ''}/orders/${props.orderId}/receipt`
  const receipt = useQuery({
    queryKey: ['guardian', 'receipt', path],
    queryFn: () => resellerRead<GuardianReceipt>(path),
    refetchInterval: (q) =>
      ['pending', 'sending'].includes(q.state.data?.email_status ?? '')
        ? 15000
        : false,
  })
  const send = useMutation({
    mutationFn: () => resellerWrite(`${path}/email`, {}, 'post'),
    onSuccess: () => {
      toast.success(t('Receipt email queued'))
      void receipt.refetch()
    },
  })
  const status: Record<string, string> = {
    not_requested: t('Not sent'),
    no_email: t('No account email'),
    pending: t('Pending'),
    sending: t('Sending'),
    sent: t('Submitted to mail server'),
    failed: t('Failed'),
    uncertain: t('Delivery outcome unknown'),
  }
  const frame = useRef<HTMLIFrameElement>(null)
  const data = receipt.data
  return (
    <GuardianShell>
      <div className='mx-auto max-w-4xl px-4 py-6'>
        <div className='mb-4 flex flex-wrap items-center gap-3'>
          <Button
            variant='outline'
            render={
              <Link to={props.admin ? '/guardian-admin' : '/guardian/orders'} />
            }
          >
            {props.admin ? t('Admin area') : t('My orders')}
          </Button>
          {data && (
            <Button onClick={() => frame.current?.contentWindow?.print()}>
              {t('Print / Save PDF')}
            </Button>
          )}
        </div>
        {receipt.isPending && <LoadingState />}
        {receipt.isError && (
          <ErrorState onRetry={() => void receipt.refetch()} />
        )}
        {data && (
          <>
            <iframe
              ref={frame}
              sandbox='allow-same-origin allow-modals'
              title={t('Payment receipt')}
              className='block aspect-[210/148] w-full rounded border bg-white shadow-sm'
              srcDoc={buildReceiptDocument({
                data,
                locale,
                t,
                billingPeriod: data.billing_period
                  ? (labels.billingPeriod[data.billing_period] ?? '')
                  : '',
              })}
            />
            <section className='mt-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-[var(--g-paper-raised)] p-4 text-sm'>
              <div className='min-w-0 space-y-1'>
                <div className='flex items-center gap-2 break-all'>
                  {data.number}
                  <CopyButton value={data.number} />
                </div>
                <p>
                  {t('Email')}: {data.recipient || t('No account email')} ·{' '}
                  {status[data.email_status]}
                </p>
              </div>
              <Button
                variant='outline'
                disabled={
                  !data.recipient ||
                  send.isPending ||
                  ['pending', 'sending'].includes(data.email_status)
                }
                onClick={() => send.mutate()}
              >
                {t('Send to account email')}
              </Button>
            </section>
          </>
        )}
      </div>
    </GuardianShell>
  )
}
