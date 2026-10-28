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
import { useMutation } from '@tanstack/react-query'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import { saveGuardianService } from '../../api'
import { useGuardianLabels } from '../../hooks/use-guardian-labels'
import { centsToYuanInput, yuanToCents } from '../../lib/price'
import {
  guardianServiceSchema,
  splitFeatureLines,
  type GuardianServiceForm,
} from '../../lib/schema'
import type { GuardianService } from '../../types'

const FORM_ID = 'guardian-service'

function toFormValues(service?: GuardianService): GuardianServiceForm {
  if (!service) {
    return {
      title: '',
      subtitle: '',
      description: '',
      features: '',
      price: '',
      delivery_note: '',
      badge: '',
      route: 'any',
      enabled: true,
      sort_order: 0,
    }
  }
  return {
    title: service.title,
    subtitle: service.subtitle,
    description: service.description,
    features: service.features.join('\n'),
    price: centsToYuanInput(service.price_cents),
    delivery_note: service.delivery_note,
    badge: service.badge,
    route: service.route,
    enabled: service.enabled,
    sort_order: service.sort_order,
  }
}

export function GuardianServiceDialog(props: {
  service?: GuardianService
  onClose: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const labels = useGuardianLabels()
  const form = useForm<GuardianServiceForm>({
    resolver: zodResolver(guardianServiceSchema),
    defaultValues: toFormValues(props.service),
  })
  const save = useMutation({
    mutationFn: (values: GuardianServiceForm) =>
      saveGuardianService(
        {
          title: values.title,
          subtitle: values.subtitle,
          description: values.description,
          features: splitFeatureLines(values.features),
          price_cents: yuanToCents(values.price) ?? 0,
          delivery_note: values.delivery_note,
          badge: values.badge,
          route: values.route,
          enabled: values.enabled,
          sort_order: values.sort_order,
        },
        props.service?.id
      ),
    onSuccess: () => {
      toast.success(t('Service saved'))
      props.onSaved()
      props.onClose()
    },
    onError: (error) => handleServerError(error, t('Failed to save service')),
  })
  const errors = form.formState.errors
  const fieldError = (message?: string) =>
    message ? <FieldError>{t(message)}</FieldError> : null

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.onClose()
      }}
      title={props.service ? t('Edit service') : t('New service')}
      description={t(
        'Changes apply to new orders only. Existing orders keep the name and price they were placed with.'
      )}
      contentClassName='sm:max-w-2xl'
      footer={
        <>
          <Button
            variant='outline'
            onClick={props.onClose}
            disabled={save.isPending}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={FORM_ID} disabled={save.isPending}>
            {save.isPending ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <form
        id={FORM_ID}
        className='space-y-4'
        noValidate
        onSubmit={form.handleSubmit((values) => save.mutate(values))}
      >
        <div className='grid gap-4 sm:grid-cols-[1fr_10rem]'>
          <Field data-invalid={!!errors.title}>
            <FieldLabel htmlFor='guardian-service-title'>
              {t('Service name')}
            </FieldLabel>
            <Input
              id='guardian-service-title'
              maxLength={128}
              aria-invalid={!!errors.title}
              {...form.register('title')}
            />
            {fieldError(errors.title?.message)}
          </Field>
          <Field data-invalid={!!errors.price}>
            <FieldLabel htmlFor='guardian-service-price'>
              {t('Price (CNY)')}
            </FieldLabel>
            <Input
              id='guardian-service-price'
              inputMode='decimal'
              placeholder='599'
              aria-invalid={!!errors.price}
              {...form.register('price')}
            />
            {fieldError(errors.price?.message)}
          </Field>
        </div>

        <Field data-invalid={!!errors.subtitle}>
          <FieldLabel htmlFor='guardian-service-subtitle'>
            {t('Subtitle')}
          </FieldLabel>
          <Input
            id='guardian-service-subtitle'
            maxLength={255}
            aria-invalid={!!errors.subtitle}
            {...form.register('subtitle')}
          />
          {fieldError(errors.subtitle?.message)}
        </Field>

        <Field data-invalid={!!errors.features}>
          <FieldLabel htmlFor='guardian-service-features'>
            {t('Specifications')}
          </FieldLabel>
          <Textarea
            id='guardian-service-features'
            rows={5}
            aria-invalid={!!errors.features}
            aria-describedby='guardian-service-features-help'
            {...form.register('features')}
          />
          <FieldDescription id='guardian-service-features-help'>
            {t(
              'One item per line, up to 12 lines. Shown as a list on the card.'
            )}
          </FieldDescription>
          {fieldError(errors.features?.message)}
        </Field>

        <Field data-invalid={!!errors.description}>
          <FieldLabel htmlFor='guardian-service-description'>
            {t('Description')}
          </FieldLabel>
          <Textarea
            id='guardian-service-description'
            rows={5}
            maxLength={5000}
            aria-invalid={!!errors.description}
            aria-describedby='guardian-service-description-help'
            {...form.register('description')}
          />
          <FieldDescription id='guardian-service-description-help'>
            {t(
              'Plain text shown in the checkout dialog. Separate paragraphs with a blank line.'
            )}
          </FieldDescription>
          {fieldError(errors.description?.message)}
        </Field>

        <div className='grid gap-4 sm:grid-cols-3'>
          <Field data-invalid={!!errors.delivery_note}>
            <FieldLabel htmlFor='guardian-service-delivery'>
              {t('Delivery time')}
            </FieldLabel>
            <Input
              id='guardian-service-delivery'
              maxLength={128}
              aria-invalid={!!errors.delivery_note}
              {...form.register('delivery_note')}
            />
            {fieldError(errors.delivery_note?.message)}
          </Field>
          <Field data-invalid={!!errors.badge}>
            <FieldLabel htmlFor='guardian-service-badge'>
              {t('Badge')}
            </FieldLabel>
            <Input
              id='guardian-service-badge'
              maxLength={32}
              aria-invalid={!!errors.badge}
              {...form.register('badge')}
            />
            {fieldError(errors.badge?.message)}
          </Field>
          <Field>
            <FieldLabel htmlFor='guardian-service-route'>
              {t('Route')}
            </FieldLabel>
            <NativeSelect
              id='guardian-service-route'
              {...form.register('route')}
            >
              {Object.entries(labels.serviceRoute).map(([value, label]) => (
                <NativeSelectOption key={value} value={value}>
                  {label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
        </div>

        <div className='grid items-end gap-4 sm:grid-cols-2'>
          <Field data-invalid={!!errors.sort_order}>
            <FieldLabel htmlFor='guardian-service-sort'>
              {t('Sort order')}
            </FieldLabel>
            <Input
              id='guardian-service-sort'
              type='number'
              step={1}
              aria-invalid={!!errors.sort_order}
              {...form.register('sort_order', { valueAsNumber: true })}
            />
            {fieldError(errors.sort_order?.message)}
          </Field>
          <Field orientation='horizontal' className='h-8 items-center'>
            <Controller
              control={form.control}
              name='enabled'
              render={({ field }) => (
                <Switch
                  id='guardian-service-enabled'
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              )}
            />
            <FieldLabel htmlFor='guardian-service-enabled'>
              {t('Listed on the page')}
            </FieldLabel>
          </Field>
        </div>
      </form>
    </Dialog>
  )
}
