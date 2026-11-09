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
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Combobox } from '@/components/ui/combobox'
import { Field, FieldLabel } from '@/components/ui/field'

import { resellerRead, type GuardianResellerSummary } from '../../reseller-api'
import { ResellerOffersPanel } from '../reseller/offers'

export function GuardianResellerProductsDialog(props: { onClose: () => void }) {
  const { t } = useTranslation()
  const [userId, setUserId] = useState<string | null>(null)
  const resellers = useQuery({
    queryKey: ['guardian', 'admin', 'resellers'],
    queryFn: () => resellerRead<GuardianResellerSummary[]>('/admin/resellers'),
  })
  return (
    <Dialog
      open
      title={t('Configure reseller products')}
      description={t(
        'Choose the products this reseller may sell and set their supply prices. The reseller sets customer-facing prices.'
      )}
      contentClassName='sm:max-w-3xl'
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
    >
      {resellers.isPending && <LoadingState />}
      {resellers.isError && (
        <ErrorState onRetry={() => void resellers.refetch()} />
      )}
      {resellers.isSuccess && resellers.data.length === 0 && (
        <EmptyState
          title={t(
            'No resellers yet. Enable Guardian reseller access in User management first.'
          )}
          className='min-h-40'
        />
      )}
      {resellers.isSuccess && resellers.data.length > 0 && (
        <div className='space-y-5'>
          <Field>
            <FieldLabel htmlFor='guardian-reseller-products-user'>
              {t('Select reseller')}
            </FieldLabel>
            <Combobox
              id='guardian-reseller-products-user'
              value={userId}
              onValueChange={setUserId}
              placeholder={t('Search reseller by username or name')}
              options={resellers.data.map((reseller) => ({
                value: String(reseller.user_id),
                label: `${reseller.username}${reseller.display_name ? ` (${reseller.display_name})` : ''} · #${reseller.user_id} · ${reseller.status === 'active' ? t('Active') : t('Suspended')}`,
              }))}
            />
          </Field>
          {userId && (
            <ResellerOffersPanel key={userId} userId={Number(userId)} />
          )}
        </div>
      )}
    </Dialog>
  )
}
