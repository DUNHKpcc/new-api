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
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { toIntlLocale } from '@/i18n/languages'

import { GuardianShell } from './components/guardian-shell'
import { ResellerOffersPanel } from './components/reseller/offers'
import { ResellerRecords } from './components/reseller/records'
import { ResellerWithdrawalForm } from './components/reseller/withdrawal-form'
import { formatCny } from './lib/price'
import {
  getGuardianReseller,
  resellerRead,
  type ResellerBalance,
} from './reseller-api'

export function GuardianResellerCenter() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const profile = useQuery({
    queryKey: ['guardian', 'reseller', 'self'],
    queryFn: () => getGuardianReseller(),
  })
  const balance = useQuery({
    queryKey: ['guardian', 'reseller-balance'],
    queryFn: () => resellerRead<ResellerBalance>('/reseller/balance'),
    enabled: !!profile.data,
  })
  const storefrontUrl = profile.data
    ? `${window.location.origin}/guardian/services?reseller=${encodeURIComponent(profile.data.code)}`
    : ''
  return (
    <GuardianShell>
      <section className='guardian-reseller-center mx-auto max-w-6xl space-y-6 px-4 py-12'>
        <h1 className='g-display text-3xl'>{t('Guardian reseller center')}</h1>
        {profile.isPending && <LoadingState />}
        {profile.isError && (
          <ErrorState onRetry={() => void profile.refetch()} />
        )}
        {profile.isSuccess && !profile.data && (
          <p>{t('Reseller access is opened by the platform administrator.')}</p>
        )}
        {profile.data && (
          <>
            <p>
              {profile.data.status === 'active'
                ? t('Reseller sales active')
                : t(
                    'Sales suspended. Existing earnings and order history remain available.'
                  )}
            </p>
            <div className='flex flex-col gap-4 rounded-xl border-2 bg-[var(--g-paper-raised)] p-4 sm:flex-row sm:items-center sm:justify-between'>
              <div className='min-w-0 space-y-2'>
                <h2 className='text-base'>{t('Storefront link')}</h2>
                <a
                  href={storefrontUrl}
                  className='block text-sm leading-6 break-all underline underline-offset-4'
                >
                  {storefrontUrl}
                </a>
              </div>
              <CopyButton
                value={storefrontUrl}
                variant='outline'
                size='default'
                aria-label={t('Copy storefront link')}
              >
                {t('Copy storefront link')}
              </CopyButton>
            </div>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Each order settles once, seven days after delivery and after actual payment fees are reconciled. All delivery and support are handled by the platform.'
              )}
            </p>
            {balance.isPending && <LoadingState />}
            {balance.isError && (
              <ErrorState onRetry={() => void balance.refetch()} />
            )}
            {balance.data && (
              <dl className='grid grid-cols-2 gap-4 sm:grid-cols-3'>
                {(
                  [
                    ['pending_cents', 'Pending earnings'],
                    ['available_cents', 'Available for withdrawal'],
                    ['reserved_cents', 'Reserved for withdrawal'],
                    ['paid_cents', 'Withdrawn'],
                    ['debt_cents', 'Amount to recover'],
                    ['held_cents', 'Refund hold'],
                  ] as const
                ).map(([key, label]) => (
                  <div
                    key={key}
                    className='rounded-xl border-2 bg-[var(--g-paper-raised)] p-4'
                  >
                    <dt className='text-sm'>{t(label)}</dt>
                    <dd className='mt-2 font-semibold'>
                      {formatCny(balance.data[key] ?? 0, locale)}
                    </dd>
                  </div>
                ))}
              </dl>
            )}
            <Tabs
              defaultValue='offers'
              className='rounded-xl border-2 bg-[var(--g-paper-raised)] p-4 sm:p-6'
            >
              <TabsList className='flex-wrap'>
                <TabsTrigger value='offers'>{t('My prices')}</TabsTrigger>
                <TabsTrigger value='orders'>{t('Order history')}</TabsTrigger>
                <TabsTrigger value='withdrawals'>
                  {t('Withdrawals')}
                </TabsTrigger>
              </TabsList>
              <TabsContent value='offers'>
                <ResellerOffersPanel
                  suspended={profile.data.status !== 'active'}
                />
              </TabsContent>
              <TabsContent value='orders'>
                <ResellerRecords />
              </TabsContent>
              <TabsContent value='withdrawals' className='space-y-6'>
                <ResellerWithdrawalForm />
                <ResellerRecords withdrawals />
              </TabsContent>
            </Tabs>
          </>
        )}
      </section>
    </GuardianShell>
  )
}
