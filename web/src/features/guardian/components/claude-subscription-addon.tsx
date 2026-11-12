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
import { Check } from 'lucide-react'
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

import { formatCny } from '../lib/price'

export function ClaudeSubscriptionAddon(props: {
  variant: 'pro' | 'max' | 'ultra'
  outOfStock?: boolean
  disabled?: boolean
  priceCents: number
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const id = useId()
  const unavailable =
    props.priceCents === 0 || props.outOfStock || props.disabled
  const titles = {
    pro: t('Add Claude Pro subscription'),
    max: t('Add Claude MAX subscription'),
    ultra: t('Add Claude Ultra subscription'),
  }
  let description = t('One-time add-on')
  if (props.outOfStock) {
    description = t('Temporarily out of stock')
  } else if (props.priceCents === 0) {
    description = t('This add-on is not available yet.')
  }
  return (
    <div className='relative min-w-0 [&>input]:absolute!'>
      <Checkbox
        id={id}
        className='peer sr-only absolute top-0 left-0'
        checked={props.checked}
        disabled={unavailable}
        onCheckedChange={props.onCheckedChange}
        aria-labelledby={`${id}-title`}
        aria-describedby={`${id}-description`}
      />
      <Label
        htmlFor={id}
        className={cn(
          'relative flex h-full min-h-36 cursor-pointer flex-col items-stretch! gap-0! rounded-xl border border-[var(--g-rule)] bg-[var(--g-paper-raised)] p-3 text-left transition-[background-color,border-color,box-shadow] duration-150 peer-focus-visible:ring-2 peer-focus-visible:ring-[var(--g-accent)] peer-focus-visible:ring-offset-2 motion-reduce:transition-none sm:p-4',
          !unavailable &&
            'hover:border-[var(--g-accent)] hover:bg-[var(--g-accent-wash)]',
          props.checked &&
            'border-[var(--g-accent)] bg-[var(--g-accent-wash)] ring-1 ring-[var(--g-accent)]',
          unavailable &&
            'cursor-not-allowed bg-[var(--g-paper-sunk)] text-[var(--g-ink-soft)]'
        )}
      >
        <span id={`${id}-title`} className='sr-only'>
          {titles[props.variant]}
        </span>
        <span
          className='flex items-center justify-between gap-1'
          aria-hidden='true'
        >
          <span className='text-[10px] font-medium tracking-[0.12em] text-[var(--g-ink-soft)] uppercase'>
            Claude
          </span>
          <span
            className={cn(
              'flex size-4 shrink-0 items-center justify-center rounded-full border border-[var(--g-rule)]',
              props.checked &&
                'border-[var(--g-accent)] bg-[var(--g-accent)] text-white'
            )}
          >
            {props.checked && <Check className='size-3' strokeWidth={2.5} />}
          </span>
        </span>
        <span
          aria-hidden='true'
          className='mt-2 text-xl leading-6 font-semibold tracking-tight'
        >
          {{ pro: 'Pro', max: 'MAX', ultra: 'Ultra' }[props.variant]}
        </span>
        <span className='mt-1 text-[11px] leading-4 font-normal text-[var(--g-ink-soft)]'>
          {t('Subscription')}
        </span>
        <span
          className={cn(
            'mt-4 whitespace-nowrap text-sm leading-6 font-semibold tabular-nums sm:text-lg',
            props.checked && 'text-[var(--g-accent)]'
          )}
        >
          {props.priceCents > 0
            ? `+${formatCny(props.priceCents, locale)}`
            : '—'}
        </span>
        <span
          id={`${id}-description`}
          className='mt-1 text-[10px] leading-4 font-normal text-[var(--g-ink-soft)]'
        >
          {description}
        </span>
      </Label>
    </div>
  )
}
