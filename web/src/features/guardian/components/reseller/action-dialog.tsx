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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel, FieldError } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { resellerMoneySchema } from '../../lib/reseller-schema'

export interface ResellerAction {
  title: string
  description?: string
  fields: {
    name: string
    label: string
    value?: string
    kind?: 'money' | 'id' | 'text'
    options?: { value: string; label: string }[]
  }[]
  preview?: (values: Record<string, string>) => ReactNode
  submit: (values: Record<string, string>) => Promise<unknown>
}

// Shared by pricing, reconciliation, refund and payout forms; every submission remains explicit.
export function ResellerActionDialog({
  action,
  onClose,
  contentClassName,
}: {
  action: ResellerAction
  onClose: () => void
  contentClassName?: string
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const schema = z.object(
    Object.fromEntries(
      action.fields.map((field): [string, z.ZodType<string, string>] => {
        if (field.options) {
          return [
            field.name,
            z.enum(field.options.map((o) => o.value) as [string, ...string[]]),
          ]
        }
        if (field.kind === 'money') return [field.name, resellerMoneySchema]
        if (field.kind === 'id') {
          return [field.name, z.string().regex(/^\d{1,9}$/, 'Enter a valid ID')]
        }
        return [
          field.name,
          z
            .string()
            .trim()
            .min(1, 'Required')
            .max(field.name === 'reason' ? 500 : 128),
        ]
      })
    )
  )
  const form = useForm<Record<string, string>>({
    resolver: zodResolver(schema),
    defaultValues: Object.fromEntries(
      action.fields.map((f) => [f.name, f.value ?? ''])
    ),
  })
  const values = useWatch({ control: form.control })
  const mutation = useMutation({
    mutationFn: action.submit,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['guardian'] })
      toast.success(t('Saved successfully'))
      onClose()
    },
  })
  return (
    <Dialog
      contentClassName={contentClassName}
      open
      onOpenChange={(v) => {
        if (!v && !mutation.isPending) onClose()
      }}
      title={t(action.title)}
      description={action.description ? t(action.description) : undefined}
      footer={
        <>
          <Button
            variant='outline'
            disabled={mutation.isPending}
            onClick={onClose}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='submit'
            form='reseller-action'
            disabled={mutation.isPending}
          >
            {t('Confirm')}
          </Button>
        </>
      }
    >
      <form
        id='reseller-action'
        className='space-y-4'
        onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
      >
        {action.fields.map((field) => (
          <Field
            key={field.name}
            data-invalid={!!form.formState.errors[field.name]}
          >
            <FieldLabel htmlFor={`reseller-${field.name}`}>
              {t(field.label)}
            </FieldLabel>
            {field.options ? (
              <NativeSelect
                id={`reseller-${field.name}`}
                {...form.register(field.name)}
              >
                {field.options.map((o) => (
                  <NativeSelectOption key={o.value} value={o.value}>
                    {t(o.label)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            ) : (
              <Input
                id={`reseller-${field.name}`}
                inputMode={
                  ({ money: 'decimal', id: 'numeric', text: 'text' } as const)[
                    field.kind ?? 'text'
                  ]
                }
                {...form.register(field.name)}
              />
            )}
            {form.formState.errors[field.name] && (
              <FieldError>
                {t(String(form.formState.errors[field.name]?.message))}
              </FieldError>
            )}
          </Field>
        ))}
        {action.preview && (
          <div className='rounded-lg border p-3 text-sm' aria-live='polite'>
            {action.preview(values as Record<string, string>)}
          </div>
        )}
      </form>
    </Dialog>
  )
}
