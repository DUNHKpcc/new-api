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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'

import { formatCny, centsToYuanInput } from '../../lib/price'
import {
  resellerRead,
  resellerWrite,
  type ResellerOffers,
} from '../../reseller-api'
import {
  TierConfirmation,
  type TierChange,
  type TierPreview,
} from '../admin/tier-prices'
import { ResellerActionDialog, type ResellerAction } from './action-dialog'

export function ResellerOffersPanel({
  userId,
  suspended = false,
  tier = 0,
}: {
  userId?: number
  suspended?: boolean
  tier?: number
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const path = userId ? `/admin/resellers/${userId}/offers` : '/reseller/offers'
  const query = useQuery({
    queryKey: ['guardian', 'offers', userId ?? 'self'],
    queryFn: () => resellerRead<ResellerOffers>(path),
  })
  const [confirmation, setConfirmation] = useState<{
    change: TierChange
    preview: TierPreview
  } | null>(null)
  const preview = useMutation({
    mutationFn: (change: TierChange) =>
      resellerWrite<TierPreview>('/admin/tier-supplies', change, 'post'),
    onSuccess: (result, change) => setConfirmation({ change, preview: result }),
  })
  const [action, setAction] = useState<ResellerAction | null>(null)
  if (query.isPending) return <LoadingState />
  if (query.isError) return <ErrorState onRetry={() => void query.refetch()} />
  return (
    <div className='space-y-3'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Retail prices must cover supply prices and estimated payment fees. Zero disables an optional price.'
        )}
      </p>
      <p className='text-sm'>
        {query.data.setting
          ? t('Configured payment fee: {{rate}}%', {
              rate: query.data.setting.fee_bps / 100,
            })
          : t('The administrator must configure the payment fee first.')}
      </p>
      <StaticDataTable
        data={query.data.services}
        getRowKey={(s) => s.id}
        emptyContent={t('No authorized products')}
        columns={[
          { id: 'title', header: t('Service'), cell: (s) => s.title },
          {
            id: 'source',
            header: t('Supply source'),
            cell: (s) => {
              const offer = query.data.offers.find((o) => o.service_id === s.id)
              return (
                <div>
                  {offer?.follow_tier
                    ? t('Following tier')
                    : t('Independent price')}
                  {userId && tier > 0 && !offer?.follow_tier && (
                    <Button
                      variant='ghost'
                      disabled={preview.isPending}
                      onClick={() =>
                        preview.mutate({
                          tier,
                          reseller_ids: [userId],
                          service_ids: [s.id],
                          restore: true,
                        })
                      }
                    >
                      {t('Restore tier prices')}
                    </Button>
                  )}
                </div>
              )
            },
          },
          {
            id: 'supply',
            header: t('Supply price'),
            cell: (s) =>
              formatCny(
                query.data.offers.find((o) => o.service_id === s.id)
                  ?.supply_cents ?? 0,
                locale
              ),
          },
          {
            id: 'retail',
            header: t('Retail price'),
            cell: (s) =>
              formatCny(
                query.data.offers.find((o) => o.service_id === s.id)
                  ?.retail_cents ?? 0,
                locale
              ),
          },
          {
            id: 'action',
            header: t('Actions'),
            cell: (s) => {
              const offer = query.data.offers.find((o) => o.service_id === s.id)
              return (
                <Button
                  variant='outline'
                  disabled={suspended || (!userId && !offer?.granted)}
                  onClick={() => {
                    const suffix = userId ? 'supply_cents' : 'retail_cents'
                    const specs = [
                      {
                        prefix: '',
                        label:
                          s.route === 'mac_mini'
                            ? 'Monthly price (CNY)'
                            : 'Price (CNY)',
                      },
                      ...(s.route === 'mac_mini'
                        ? [
                            {
                              prefix: 'quarter_',
                              label: 'Quarterly price (CNY)',
                            },
                            { prefix: 'annual_', label: 'Annual price (CNY)' },
                            { prefix: 'max_', label: 'MAX add-on price (CNY)' },
                            {
                              prefix: 'pro_',
                              label: 'Pro add-on price (CNY)',
                            },
                            {
                              prefix: 'ultra_',
                              label: 'Ultra add-on price (CNY)',
                            },
                          ]
                        : []),
                    ]
                    const flag = userId ? 'granted' : 'listed'
                    setAction({
                      title: userId
                        ? 'Configure supply prices'
                        : 'Set retail prices',
                      description: s.title,
                      fields: [
                        {
                          name: flag,
                          label: userId
                            ? 'Authorized for sale'
                            : 'Listed for sale',
                          value: String(offer?.[flag] ?? false),
                          options: [
                            { value: 'false', label: 'No' },
                            { value: 'true', label: 'Yes' },
                          ],
                        },
                        ...specs.map((spec) => ({
                          name: spec.prefix + suffix,
                          label: spec.label,
                          kind: 'money' as const,
                          value: centsToYuanInput(
                            (offer?.[
                              (spec.prefix + suffix) as keyof typeof offer
                            ] as number) ?? 0
                          ),
                        })),
                      ],
                      submit: (values) =>
                        resellerWrite(
                          `${path}/${s.id}`,
                          Object.fromEntries(
                            Object.entries(values).map(([key, value]) => [
                              key,
                              key === flag
                                ? value === 'true'
                                : Math.round(Number(value) * 100),
                            ])
                          )
                        ),
                    })
                  }}
                >
                  {t('Edit')}
                </Button>
              )
            },
          },
        ]}
      />
      {confirmation && (
        <TierConfirmation
          {...confirmation}
          onClose={() => setConfirmation(null)}
          onSaved={() => {
            setConfirmation(null)
            void query.refetch()
          }}
        />
      )}
      {action && (
        <ResellerActionDialog action={action} onClose={() => setAction(null)} />
      )}
    </div>
  )
}
