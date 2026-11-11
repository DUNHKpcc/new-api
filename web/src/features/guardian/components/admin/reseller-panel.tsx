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

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

import { resellerRead, resellerWrite } from '../../reseller-api'
import {
  ResellerActionDialog,
  type ResellerAction,
} from '../reseller/action-dialog'
import { ResellerRecords } from '../reseller/records'
import { GuardianResellerList } from './reseller-list'
import { GuardianTierPrices } from './tier-prices'

export function GuardianResellerAdminPanel() {
  const { t } = useTranslation()
  const [action, setAction] = useState<ResellerAction | null>(null)
  const setting = useQuery({
    queryKey: ['guardian', 'reseller-setting'],
    queryFn: () =>
      resellerRead<{ fee_bps: number } | null>('/admin/reseller-setting'),
  })
  return (
    <div className='space-y-5'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Open or suspend reseller access in User management. Reconcile channel fees before processing manual payouts.'
        )}
      </p>
      {setting.isPending && <LoadingState />}
      {setting.isError && <ErrorState onRetry={() => void setting.refetch()} />}
      <div className='flex flex-wrap gap-3'>
        <Button
          variant='outline'
          disabled={!setting.isSuccess}
          onClick={() =>
            setAction({
              title: 'Configure payment fee',
              description:
                'Use the actual Epay merchant fee, not an assumed Alipay rate. This is used for estimates; each paid order must still be reconciled.',
              fields: [
                {
                  name: 'rate',
                  label: 'Payment fee (%)',
                  kind: 'money',
                  value: String((setting.data?.fee_bps ?? 0) / 100),
                },
              ],
              submit: (v) =>
                resellerWrite('/admin/reseller-setting', {
                  fee_bps: Math.round(Number(v.rate) * 100),
                }),
            })
          }
        >
          {t('Configure payment fee')}
        </Button>
      </div>
      <Tabs defaultValue='resellers'>
        <TabsList>
          <TabsTrigger value='resellers'>{t('Reseller list')}</TabsTrigger>
          <TabsTrigger value='prices'>{t('Tier supply prices')}</TabsTrigger>
          <TabsTrigger value='withdrawals'>{t('Withdrawals')}</TabsTrigger>
        </TabsList>
        <TabsContent value='resellers'>
          <GuardianResellerList />
        </TabsContent>
        <TabsContent value='prices'>
          <GuardianTierPrices />
        </TabsContent>
        <TabsContent value='withdrawals'>
          <ResellerRecords admin withdrawals />
        </TabsContent>
      </Tabs>
      {action && (
        <ResellerActionDialog action={action} onClose={() => setAction(null)} />
      )}
    </div>
  )
}
