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
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

import { formatCny } from '../lib/price'

export function ClaudeSubscriptionAddon(props: {
  variant: 'max' | 'ultra'
  priceCents: number
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const id = useId()
  return (
    <div className='relative min-w-0 [&>input]:absolute!'>
      <Checkbox
        id={id}
        className='peer sr-only absolute top-0 left-0'
        checked={props.checked}
        disabled={props.priceCents === 0}
        onCheckedChange={props.onCheckedChange}
        aria-labelledby={`${id}-title`}
        aria-describedby={`${id}-description`}
      />
      <Label
        htmlFor={id}
        className={cn(
          'g-rule block h-full cursor-pointer space-y-2 rounded-lg border p-3 peer-focus-visible:ring-2 peer-focus-visible:ring-[var(--g-accent)] peer-focus-visible:ring-offset-2',
          props.checked && 'border-[var(--g-accent)] bg-[var(--g-accent-wash)]',
          props.priceCents === 0 && 'cursor-not-allowed opacity-50'
        )}
      >
        <span
          id={`${id}-title`}
          className='flex items-start gap-1.5 text-sm leading-5'
        >
          <img
            src='/images/guardian/claude-color.png'
            alt=''
            className='size-5 shrink-0'
          />
          <span className='min-w-0'>
            {props.variant === 'ultra'
              ? t('Add Claude Ultra for the first month')
              : t('Add Claude MAX for the first month')}
          </span>
        </span>
        <span className='block text-sm font-medium text-[var(--g-accent)]'>
          {props.priceCents > 0 && (
            <span>
              {t('Charged: +{{price}}', {
                price: formatCny(props.priceCents, locale),
              })}
            </span>
          )}
        </span>
        <span
          id={`${id}-description`}
          className='block text-xs leading-5 font-normal text-[var(--g-ink-soft)]'
        >
          {props.priceCents > 0
            ? t('First month only. Later subscription renewals are separate.')
            : t('This add-on is not available yet.')}
        </span>
      </Label>
    </div>
  )
}
