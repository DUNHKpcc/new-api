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
import { TagInput } from '@/components/tag-input'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
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
import { MacChipIcon } from '../mac-chip-icon'

const FORM_ID = 'guardian-service'

function toFormValues(service?: GuardianService): GuardianServiceForm {
  if (!service) {
    return {
      chip: 'm4',
      region: '',
      hardware_config: '16_256',
      base_sales: 0,
      title: '',
      subtitle: '',
      description: '',
      features: '',
      tags: [],
      price: '',
      claude_max_price: '',
      claude_pro_price: '',
      claude_ultra_price: '2500',
      quarter_price: '',
      annual_price: '',
      delivery_note: '',
      badge: '',
      route: 'any',
      out_of_stock: false,
      stock_note: '',
      claude_pro_out_of_stock: false,
      claude_max_out_of_stock: false,
      claude_ultra_out_of_stock: false,
      invoice_enabled: false,
      enabled: true,
      sort_order: 0,
    }
  }
  return {
    chip: service.chip ?? '',
    region: service.region ?? '',
    hardware_config: service.hardware_config ?? '',
    base_sales: service.base_sales ?? 0,
    title: service.title,
    subtitle: service.subtitle,
    description: service.description,
    features: service.features.join('\n'),
    tags: service.tags ?? [],
    price: centsToYuanInput(service.price_cents),
    claude_max_price: service.claude_max_price_cents
      ? centsToYuanInput(service.claude_max_price_cents)
      : '',
    claude_pro_price: service.claude_pro_price_cents
      ? centsToYuanInput(service.claude_pro_price_cents)
      : '',
    claude_ultra_price: service.claude_ultra_price_cents
      ? centsToYuanInput(service.claude_ultra_price_cents)
      : '',
    quarter_price: service.quarter_price_cents
      ? centsToYuanInput(service.quarter_price_cents)
      : '',
    annual_price: service.annual_price_cents
      ? centsToYuanInput(service.annual_price_cents)
      : '',
    delivery_note: service.delivery_note,
    badge: service.badge,
    route: service.route,
    out_of_stock: service.out_of_stock ?? false,
    stock_note: service.stock_note ?? '',
    claude_pro_out_of_stock: service.claude_pro_out_of_stock ?? false,
    claude_max_out_of_stock: service.claude_max_out_of_stock ?? false,
    claude_ultra_out_of_stock: service.claude_ultra_out_of_stock ?? false,
    invoice_enabled: service.invoice_enabled ?? false,
    enabled: service.enabled,
    sort_order: service.sort_order,
  }
}

export function GuardianServiceDialog(props: {
  service?: GuardianService
  onClose: () => void
  onSaved: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const form = useForm<GuardianServiceForm>({
    resolver: zodResolver(guardianServiceSchema),
    defaultValues: toFormValues(props.service),
  })
  const save = useMutation({
    mutationFn: (values: GuardianServiceForm) =>
      saveGuardianService(
        {
          chip: values.route === 'mac_mini' ? values.chip : '',
          region: values.route === 'mac_mini' ? values.region : '',
          hardware_config:
            values.route === 'mac_mini' ? values.hardware_config : '',
          base_sales: values.base_sales,
          title: values.title,
          subtitle: values.subtitle,
          description: values.description,
          features: splitFeatureLines(values.features),
          tags: values.tags,
          claude_max_price_cents:
            values.route === 'mac_mini'
              ? (yuanToCents(values.claude_max_price) ?? 0)
              : 0,
          claude_pro_price_cents:
            values.route === 'mac_mini'
              ? (yuanToCents(values.claude_pro_price) ?? 0)
              : 0,
          claude_ultra_price_cents:
            values.route === 'mac_mini'
              ? (yuanToCents(values.claude_ultra_price) ?? 0)
              : 0,
          quarter_price_cents:
            values.route === 'mac_mini'
              ? (yuanToCents(values.quarter_price) ?? 0)
              : 0,
          annual_price_cents:
            values.route === 'mac_mini'
              ? (yuanToCents(values.annual_price) ?? 0)
              : 0,
          price_cents: yuanToCents(values.price) ?? 0,
          delivery_note: values.delivery_note,
          badge: values.badge,
          route: values.route,
          out_of_stock: values.out_of_stock,
          stock_note: values.stock_note,
          claude_pro_out_of_stock: values.claude_pro_out_of_stock,
          claude_max_out_of_stock: values.claude_max_out_of_stock,
          claude_ultra_out_of_stock: values.claude_ultra_out_of_stock,
          invoice_enabled: values.invoice_enabled ?? false,
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
  const route = form.watch('route')
  const baseSales = form.watch('base_sales')
  const actualSales = props.service ? props.service.actual_sales : 0
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
              {route === 'mac_mini'
                ? t('Monthly price (CNY)')
                : t('Price (CNY)')}
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

        {route === 'mac_mini' && (
          <div className='grid gap-4 sm:grid-cols-2'>
            {(['quarter_price', 'annual_price'] as const).map((field) => (
              <Field key={field} data-invalid={!!errors[field]}>
                <FieldLabel htmlFor={`guardian-service-${field}`}>
                  {field === 'quarter_price'
                    ? t('Quarterly total (CNY)')
                    : t('Yearly total (CNY)')}
                </FieldLabel>
                <Input
                  id={`guardian-service-${field}`}
                  inputMode='decimal'
                  {...form.register(field)}
                  aria-invalid={!!errors[field]}
                />
                <FieldDescription>
                  {t('Leave blank to disable this period.')}
                </FieldDescription>
                {fieldError(errors[field]?.message)}
              </Field>
            ))}
          </div>
        )}

        {route === 'mac_mini' &&
          (
            [
              {
                field: 'claude_pro_price',
                label: 'Claude Pro add-on price (CNY)',
              },
              {
                field: 'claude_max_price',
                label: 'Claude MAX add-on price (CNY)',
              },
              {
                field: 'claude_ultra_price',
                label: 'Claude Ultra add-on price (CNY)',
              },
            ] as const
          ).map(({ field, label }) => (
            <Field key={field} data-invalid={!!errors[field]}>
              <FieldLabel htmlFor={`guardian-service-${field}`}>
                {t(label)}
              </FieldLabel>
              <Input
                id={`guardian-service-${field}`}
                inputMode='decimal'
                {...form.register(field)}
                aria-invalid={!!errors[field]}
              />
              <FieldDescription>
                {t(
                  'Charged once with the hosting order. Leave blank to disable. Later subscription renewals are separate.'
                )}
              </FieldDescription>
              {fieldError(errors[field]?.message)}
            </Field>
          ))}
        <fieldset className='space-y-3 rounded-lg border p-4'>
          <legend className='px-1 text-sm font-medium'>
            {t('Availability')}
          </legend>
          {(
            [
              { name: 'out_of_stock', label: t('Service out of stock') },
              ...(route === 'mac_mini'
                ? ([
                    {
                      name: 'claude_pro_out_of_stock',
                      label: t('Claude Pro out of stock'),
                    },
                    {
                      name: 'claude_max_out_of_stock',
                      label: t('Claude MAX out of stock'),
                    },
                    {
                      name: 'claude_ultra_out_of_stock',
                      label: t('Claude Ultra out of stock'),
                    },
                  ] as const)
                : []),
            ] as const
          ).map(({ name, label }) => (
            <Field key={name} orientation='horizontal'>
              <Controller
                control={form.control}
                name={name}
                render={({ field }) => (
                  <Switch
                    id={`guardian-${name}`}
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                )}
              />
              <FieldLabel htmlFor={`guardian-${name}`}>{label}</FieldLabel>
            </Field>
          ))}
          <Field data-invalid={!!errors.stock_note}>
            <FieldLabel htmlFor='guardian-stock-note'>
              {t('Stock notice')}
            </FieldLabel>
            <Textarea
              id='guardian-stock-note'
              maxLength={255}
              {...form.register('stock_note')}
              aria-invalid={!!errors.stock_note}
            />
            <FieldDescription>
              {t(
                'Optional. Shown when the service or an add-on is out of stock.'
              )}
            </FieldDescription>
            {fieldError(errors.stock_note?.message)}
          </Field>
        </fieldset>
        <Field data-invalid={!!errors.base_sales}>
          <FieldLabel htmlFor='guardian-service-base-sales'>
            {t('Base sales')}
          </FieldLabel>
          <Input
            id='guardian-service-base-sales'
            type='number'
            min={0}
            max={1000000000}
            step={1}
            aria-invalid={!!errors.base_sales}
            {...form.register('base_sales', { valueAsNumber: true })}
          />
          <FieldDescription>
            {t('Added to valid paid orders for display only.')}
          </FieldDescription>
          {errors.base_sales && (
            <FieldError>
              {t('Enter a whole number from 0 to 1,000,000,000')}
            </FieldError>
          )}
          <p className='text-muted-foreground text-sm'>
            {t('Actual sales: {{total}}', {
              total: formatNumber(actualSales, locale),
            })}
            {' · '}
            {t('Displayed sales: {{total}}', {
              total: formatNumber(
                actualSales == null ||
                  !Number.isInteger(baseSales) ||
                  baseSales < 0 ||
                  baseSales > 1000000000
                  ? undefined
                  : actualSales + baseSales,
                locale
              ),
            })}
          </p>
        </Field>

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

        <Field data-invalid={!!errors.tags}>
          <FieldLabel htmlFor='guardian-service-tags'>
            {t('Display tags')}
          </FieldLabel>
          <Controller
            control={form.control}
            name='tags'
            render={({ field }) => (
              <>
                <TagInput
                  id='guardian-service-tags'
                  value={field.value}
                  onChange={field.onChange}
                  disabled={save.isPending}
                  ariaInvalid={!!errors.tags}
                  ariaDescribedBy='guardian-tags-help'
                />
                <div className='flex flex-wrap gap-1.5'>
                  {[
                    t('Static IP'),
                    t('Claude Pro subscription'),
                    t('Claude MAX subscription'),
                    t('Claude Ultra subscription'),
                  ].map((tag) => (
                    <Button
                      key={tag}
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={
                        save.isPending ||
                        field.value.includes(tag) ||
                        field.value.length >= 8
                      }
                      onClick={() => field.onChange([...field.value, tag])}
                    >
                      {tag}
                    </Button>
                  ))}
                </div>
              </>
            )}
          />
          <FieldDescription id='guardian-tags-help'>
            {t(
              'Optional display labels only. Add, edit or remove them without changing the price. Up to 8 tags, 24 characters each.'
            )}
          </FieldDescription>
          {fieldError(
            errors.tags?.message ??
              errors.tags?.find?.((error) => error?.message)?.message
          )}
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

        {route === 'mac_mini' && (
          <Field>
            <FieldLabel id='guardian-chip-label'>{t('Mac chip')}</FieldLabel>
            <Controller
              control={form.control}
              name='chip'
              render={({ field }) => (
                <RadioGroup
                  aria-labelledby='guardian-chip-label'
                  value={field.value}
                  onValueChange={field.onChange}
                  className='grid-cols-2'
                >
                  {(['m4', 'm6'] as const).map((chip) => (
                    <label
                      key={chip}
                      className='border-input has-data-checked:border-primary has-data-checked:bg-muted flex cursor-pointer items-center justify-center gap-3 rounded-lg border p-3'
                    >
                      <RadioGroupItem
                        value={chip}
                        aria-label={`Apple ${chip.toUpperCase()}`}
                      />
                      <span aria-hidden='true'>
                        <MacChipIcon chip={chip} />
                      </span>
                    </label>
                  ))}
                </RadioGroup>
              )}
            />
          </Field>
        )}

        {route === 'mac_mini' && (
          <div className='grid gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='guardian-service-region'>
                {t('Hosting region')}
              </FieldLabel>
              <NativeSelect
                id='guardian-service-region'
                {...form.register('region')}
              >
                <NativeSelectOption value=''>{t('Not set')}</NativeSelectOption>
                {Object.entries(labels.macRegion).map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>
                    {label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field>
              <FieldLabel htmlFor='guardian-service-config'>
                {t('Hardware configuration')}
              </FieldLabel>
              <NativeSelect
                id='guardian-service-config'
                {...form.register('hardware_config')}
              >
                <NativeSelectOption value=''>{t('Not set')}</NativeSelectOption>
                {Object.entries(labels.macConfig).map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>
                    {label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
          </div>
        )}

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
          <Field orientation='horizontal'>
            <Controller
              control={form.control}
              name='invoice_enabled'
              render={({ field }) => (
                <Switch
                  id='guardian-invoice-enabled'
                  checked={field.value ?? false}
                  onCheckedChange={field.onChange}
                />
              )}
            />
            <FieldLabel htmlFor='guardian-invoice-enabled'>
              {t('Supports invoicing')}
            </FieldLabel>
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
