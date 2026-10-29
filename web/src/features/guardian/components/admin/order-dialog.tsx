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
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { toIntlLocale } from '@/i18n/languages'
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { updateAdminGuardianOrder } from '../../api'
import { GUARDIAN_ADMIN_TRANSITIONS } from '../../constants'
import { useGuardianLabels } from '../../hooks/use-guardian-labels'
import { formatCny } from '../../lib/price'
import type { AdminGuardianOrder, GuardianOrderStatus } from '../../types'
import { OrderStatusBadge } from '../order-status-badge'
import { GuardianOrderHistory } from './order-history'

const MAX_ADMIN_NOTE = 2000

export function GuardianOrderDialog(props: {
  order: AdminGuardianOrder
  onClose: () => void
  onSaved: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const order = props.order
  const [status, setStatus] = useState<GuardianOrderStatus>(order.status)
  const [adminNote, setAdminNote] = useState(order.admin_note)
  const statusOptions = [
    order.status,
    ...GUARDIAN_ADMIN_TRANSITIONS[order.status],
  ]
  const noteTooLong = adminNote.length > MAX_ADMIN_NOTE
  const save = useMutation({
    mutationFn: () =>
      updateAdminGuardianOrder(order.id, {
        status,
        admin_note: adminNote.trim(),
      }),
    onSuccess: () => {
      toast.success(t('Order updated'))
      props.onSaved()
      props.onClose()
    },
    onError: (error) => handleServerError(error, t('Failed to update order')),
  })
  const details = [
    { label: t('Order number'), value: order.trade_no },
    { label: t('User'), value: `${order.username} (#${order.user_id})` },
    { label: t('Amount'), value: formatCny(order.amount_cents, locale) },
    { label: t('Payment method'), value: order.payment_method || '-' },
    { label: t('Route'), value: labels.orderRoute[order.route] ?? order.route },
    { label: t('Operating system'), value: labels.os[order.os] ?? order.os },
    {
      label: t('Contact'),
      value: `${labels.contactType[order.contact_type] ?? order.contact_type} · ${order.contact}`,
    },
    { label: t('Ordered at'), value: formatTimestampToDate(order.created_at) },
    {
      label: t('Marked refunded at'),
      value: formatTimestampToDate(order.refunded_at || 0),
    },
    { label: t('Paid at'), value: formatTimestampToDate(order.paid_at) },
    {
      label: t('Delivered at'),
      value: formatTimestampToDate(order.delivered_at),
    },
  ]

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.onClose()
      }}
      title={order.service_title}
      description={<OrderStatusBadge status={order.status} />}
      contentClassName='sm:max-w-xl'
      footer={
        <>
          <Button
            variant='outline'
            onClick={props.onClose}
            disabled={save.isPending}
          >
            {t('Cancel')}
          </Button>
          <Button
            onClick={() => save.mutate()}
            disabled={save.isPending || noteTooLong}
          >
            {save.isPending ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <div className='space-y-5'>
        <dl className='grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2'>
          {details.map((detail) => (
            <div key={detail.label} className='flex min-w-0 gap-3'>
              <dt className='text-muted-foreground w-24 shrink-0'>
                {detail.label}
              </dt>
              <dd className='min-w-0 break-words'>{detail.value}</dd>
            </div>
          ))}
        </dl>
        {order.note && (
          <div className='bg-muted/50 rounded-lg border p-3 text-sm whitespace-pre-line'>
            {order.note}
          </div>
        )}

        <Field>
          <FieldLabel htmlFor='guardian-order-status'>{t('Status')}</FieldLabel>
          <NativeSelect
            id='guardian-order-status'
            value={status}
            disabled={statusOptions.length === 1}
            onChange={(event) =>
              setStatus(event.target.value as GuardianOrderStatus)
            }
          >
            {statusOptions.map((option) => (
              <NativeSelectOption key={option} value={option}>
                {labels.status[option]}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          {statusOptions.length === 1 && (
            <FieldDescription>
              {t('This order is closed and its status can no longer change.')}
            </FieldDescription>
          )}
        </Field>

        {status === 'refunded' && order.status !== 'refunded' && (
          <Alert variant='destructive'>
            <AlertDescription>
              {t(
                'Marking as refunded does not send money back. Refund the payment in the payment gateway merchant dashboard first.'
              )}
            </AlertDescription>
          </Alert>
        )}

        <Field data-invalid={noteTooLong}>
          <FieldLabel htmlFor='guardian-order-note'>
            {t('Message to the customer')}
          </FieldLabel>
          <Textarea
            id='guardian-order-note'
            rows={4}
            value={adminNote}
            aria-invalid={noteTooLong}
            onChange={(event) => setAdminNote(event.target.value)}
          />
          <FieldDescription>
            {t(
              'Shown on the customer order page. Do not include credentials here either.'
            )}
          </FieldDescription>
          {noteTooLong && (
            <FieldError>
              {t('Message must be at most 2000 characters')}
            </FieldError>
          )}
        </Field>
        <GuardianOrderHistory orderId={order.id} />
      </div>
    </Dialog>
  )
}
