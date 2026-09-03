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
import { Calculator, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { SubscriptionPlan } from '@/features/subscriptions/types'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { DEFAULT_CURRENCY_CONFIG } from '@/stores/system-config-store'

import {
  estimateSubscriptionTokens,
  formatEstimatedTokenCount,
  getSubscriptionResetCount,
  getSubscriptionTotalBudgetUsd,
  resolveSubscriptionDisplayModels,
} from '../lib/subscription-estimate'
import type { PricingModel } from '../types'

export interface SubscriptionPlanEstimateProps {
  plan: SubscriptionPlan
  models?: PricingModel[]
  quotaPerUnit?: number
  subscriptionDisplayModels?: string
}

export function SubscriptionPlanEstimate(props: SubscriptionPlanEstimateProps) {
  const { t } = useTranslation()
  const totalAmount = Number(props.plan.total_amount || 0)
  const quotaPerUnit =
    props.quotaPerUnit ?? DEFAULT_CURRENCY_CONFIG.quotaPerUnit
  const resetCount = getSubscriptionResetCount(props.plan)
  const totalBudgetUsd = getSubscriptionTotalBudgetUsd(props.plan, quotaPerUnit)
  const displayModels = resolveSubscriptionDisplayModels(
    props.subscriptionDisplayModels,
    props.models || []
  )
  const estimateRows = displayModels.map((entry) => ({
    ...entry,
    estimate: entry.model
      ? estimateSubscriptionTokens(
          props.plan,
          entry.model,
          quotaPerUnit,
          props.plan.upgrade_group
        )
      : null,
  }))

  return (
    <>
      <div
        className='border-border/60 flex min-w-0 items-center justify-between gap-3 border-b py-3'
        data-subscription-total-budget={
          totalBudgetUsd === null ? 'unlimited' : totalBudgetUsd
        }
        data-subscription-reset-count={resetCount}
      >
        <span className='text-muted-foreground flex min-w-0 items-center gap-1 text-[11px] leading-4'>
          <Calculator className='size-3 shrink-0' aria-hidden='true' />
          <span className='truncate'>{t('Total quota over validity')}</span>
        </span>
        <span className='shrink-0 text-right'>
          <strong className='block text-xs font-semibold tabular-nums'>
            {totalBudgetUsd === null
              ? t('Unlimited')
              : formatBillingCurrencyFromUSD(totalBudgetUsd, {
                  digitsLarge: 2,
                  digitsSmall: 2,
                  abbreviate: false,
                })}
          </strong>
          {totalAmount > 0 && (
            <span className='text-muted-foreground block text-[10px] tabular-nums'>
              {t('{{count}} reset periods', { count: resetCount })}
            </span>
          )}
        </span>
      </div>

      {estimateRows.length > 0 && (
        <div
          className='mt-3 min-w-0 space-y-2'
          data-subscription-token-estimates='true'
        >
          <div className='flex min-w-0 items-center justify-between gap-3'>
            <span className='text-muted-foreground flex min-w-0 items-center gap-1 text-[11px] leading-4'>
              <Sparkles className='size-3 shrink-0' aria-hidden='true' />
              <span className='truncate'>{t('Estimated tokens')}</span>
            </span>
            <span className='text-muted-foreground shrink-0 text-[10px]'>
              {t('Current pricing')}
            </span>
          </div>
          <div className='border-border/60 divide-y rounded-md border'>
            {estimateRows.map((row) => {
              const estimate = row.estimate
              let estimateLabel = t('Not available')
              if (estimate?.reason === 'ok' && estimate.tokens !== null) {
                estimateLabel = `${formatEstimatedTokenCount(estimate.tokens)} ${t('tokens')}`
              } else if (
                estimate?.reason === 'unlimited' ||
                estimate?.reason === 'free'
              ) {
                estimateLabel = t('Unlimited')
              }

              return (
                <div
                  key={row.name}
                  className='flex min-w-0 items-center justify-between gap-3 px-2.5 py-2'
                  data-subscription-estimate-model={row.name}
                >
                  <span
                    className='min-w-0 truncate text-xs font-medium'
                    title={row.name}
                  >
                    {row.name}
                  </span>
                  <span className='shrink-0 text-right text-xs font-semibold tabular-nums'>
                    {estimateLabel}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </>
  )
}
