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
import { useTranslation } from 'react-i18next'

import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import { useGuardianLabels } from '../hooks/use-guardian-labels'
import { formatCny, guardianPeriodPrice } from '../lib/price'
import type { GuardianBillingPeriod, GuardianService } from '../types'

export function GuardianPeriodSelector(props: {
  prices?: Pick<
    GuardianService,
    'price_cents' | 'quarter_price_cents' | 'annual_price_cents'
  >
  value: GuardianBillingPeriod
  onChange: (value: GuardianBillingPeriod) => void
  label: string
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const prices = props.prices ?? { price_cents: 0 }
  const pricing = guardianPeriodPrice(prices, props.value)
  return (
    <div>
      <RadioGroup
        aria-label={props.label}
        value={props.value}
        onValueChange={(value) =>
          props.onChange(value as GuardianBillingPeriod)
        }
        className='mt-4 grid grid-cols-3 gap-2'
      >
        {(['monthly', 'quarterly', 'yearly'] as const).map((option) => {
          const plan = guardianPeriodPrice(prices, option)
          return (
            <label
              key={option}
              className={cn(
                'g-rule relative flex cursor-pointer flex-col items-center gap-1 rounded-lg border px-1 py-2 text-xs has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-[var(--g-accent)] has-[:focus-visible]:ring-offset-2',
                props.value === option &&
                  'border-[var(--g-accent)] bg-[var(--g-accent-wash)]',
                !plan.total && 'cursor-not-allowed opacity-40'
              )}
            >
              <RadioGroupItem
                className='sr-only absolute'
                value={option}
                disabled={!plan.total}
              />
              <span>{labels.billingPeriod[option]}</span>
              <span className='font-medium'>
                {plan.total ? formatCny(plan.total, locale) : '—'}
              </span>
              {plan.total > 0 && plan.savings > 0 && (
                <span className='text-[var(--g-accent)]'>
                  {t('Save {{percent}}%', {
                    percent: formatNumber(plan.discount, locale),
                  })}
                </span>
              )}
            </label>
          )
        })}
      </RadioGroup>
      {props.prices && (
        <div className='mt-2 space-y-1 text-sm'>
          <p className='text-[var(--g-ink-soft)]'>
            {labels.billingPeriod[props.value]} ·{' '}
            {t('Equivalent to {{price}}/month', {
              price: formatCny(pricing.monthly, locale),
            })}
          </p>
          {props.value !== 'monthly' && pricing.savings > 0 && (
            <p className='font-medium text-[var(--g-accent)]'>
              {t('Save {{amount}} ({{percent}}%) vs monthly payments', {
                amount: formatCny(pricing.savings, locale),
                percent: formatNumber(pricing.discount, locale),
              })}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
