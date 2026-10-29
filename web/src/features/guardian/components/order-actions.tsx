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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { submitPaymentForm } from '@/features/wallet/lib/payment'
import { handleServerError } from '@/lib/handle-server-error'

import { cancelGuardianOrder, payGuardianOrder } from '../api'
import { GUARDIAN_QUERY_KEYS } from '../constants'
import type { GuardianOrder } from '../types'

export function OrderActions(props: { order: GuardianOrder }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmingCancel, setConfirmingCancel] = useState(false)
  const refreshOrders = () =>
    queryClient.invalidateQueries({ queryKey: GUARDIAN_QUERY_KEYS.myOrders })
  const pay = useMutation({
    mutationFn: () => payGuardianOrder(props.order.id),
    onSuccess: (result) => {
      submitPaymentForm(result.pay_url, result.pay_params)
      toast.success(t('Payment page opened'))
    },
    onError: (error) => {
      handleServerError(error, t('Payment request failed'))
      // The order may have expired or been paid meanwhile.
      void refreshOrders()
    },
  })
  const cancel = useMutation({
    mutationFn: () => cancelGuardianOrder(props.order.id),
    onSuccess: () => {
      setConfirmingCancel(false)
      toast.success(t('Order cancelled'))
      void refreshOrders()
    },
    onError: (error) => {
      handleServerError(error, t('Failed to cancel the order'))
      void refreshOrders()
    },
  })
  const busy = pay.isPending || cancel.isPending

  return (
    <div className='mt-4 flex flex-wrap gap-2'>
      <Button size='sm' disabled={busy} onClick={() => pay.mutate()}>
        {t('Continue payment')}
      </Button>
      <Button
        size='sm'
        variant='outline'
        disabled={busy}
        onClick={() => setConfirmingCancel(true)}
      >
        {t('Cancel order')}
      </Button>
      <ConfirmDialog
        open={confirmingCancel}
        onOpenChange={setConfirmingCancel}
        title={t('Cancel this order?')}
        desc={t(
          'Only cancel if you have not paid. If a payment still goes through, the order will show as paid.'
        )}
        cancelBtnText={t('Keep order')}
        confirmText={t('Cancel order')}
        destructive
        isLoading={cancel.isPending}
        handleConfirm={() => cancel.mutate()}
        className='guardian-root'
      />
    </div>
  )
}
