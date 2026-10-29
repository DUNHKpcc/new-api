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
import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  Field,
  FieldLabel,
  FieldError,
  FieldDescription,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'

import { useGuardianLabels } from '../hooks/use-guardian-labels'
import type { GuardianContactUpdate } from '../types'

export function GuardianContactFields() {
  const { t } = useTranslation()
  const labels = useGuardianLabels()
  const form = useFormContext<GuardianContactUpdate>()
  const errors = form.formState.errors
  return (
    <>
      {' '}
      <div className='grid gap-4 sm:grid-cols-[9rem_1fr]'>
        <Field>
          <FieldLabel htmlFor='guardian-contact-type'>
            {t('Contact via')}
          </FieldLabel>
          <NativeSelect
            id='guardian-contact-type'
            {...form.register('contact_type')}
          >
            {Object.entries(labels.contactType).map(([value, label]) => (
              <NativeSelectOption key={value} value={value}>
                {label}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </Field>
        <Field data-invalid={!!errors.contact}>
          <FieldLabel htmlFor='guardian-contact'>{t('Contact')}</FieldLabel>
          <Input
            id='guardian-contact'
            maxLength={128}
            autoComplete='off'
            aria-invalid={!!errors.contact}
            aria-describedby={
              errors.contact ? 'guardian-contact-error' : undefined
            }
            {...form.register('contact')}
          />
          {errors.contact && (
            <FieldError id='guardian-contact-error'>
              {t(errors.contact.message ?? '')}
            </FieldError>
          )}
        </Field>
      </div>
      <Field data-invalid={!!errors.note}>
        <FieldLabel htmlFor='guardian-note'>{t('Note (optional)')}</FieldLabel>
        <Textarea
          id='guardian-note'
          rows={3}
          maxLength={1000}
          aria-invalid={!!errors.note}
          aria-describedby='guardian-note-help'
          {...form.register('note')}
        />
        <FieldDescription id='guardian-note-help'>
          {t(
            'Your current setup, budget, or anything we should know before scheduling.'
          )}
        </FieldDescription>
        {errors.note && <FieldError>{t(errors.note.message ?? '')}</FieldError>}
      </Field>
    </>
  )
}
