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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { toIntlLocale } from '@/i18n/languages'

import { listAdminGuardianServices } from '../../api'
import { centsToYuanInput, formatCny, yuanToCents } from '../../lib/price'
import { resellerRead, resellerWrite } from '../../reseller-api'
import type { GuardianService } from '../../types'

export interface TierPrice {
  tier: number
  service_id: number
  granted: boolean
  supply_cents: number
  quarter_supply_cents: number
  annual_supply_cents: number
  max_supply_cents: number
  pro_supply_cents: number
  ultra_supply_cents: number
}
export interface TierChange {
  tier: number
  prices?: TierPrice[]
  reseller_ids?: number[]
  service_ids?: number[]
  restore?: boolean
  preview_token?: string
}
export interface TierPreview {
  token: string
  reseller_ids?: number[]
  offer_changes?: {
    reseller_id: number
    service_id: number
    before: Omit<TierPrice, 'tier'>
    after: Omit<TierPrice, 'tier'>
  }[]
  resellers: number
  independent: number
  unavailable: number
  before: TierPrice[] | null
  after: TierPrice[] | null
}

export function TierSelect(props: {
  value: number
  onChange: (tier: number) => void
  all?: boolean
}) {
  const { t } = useTranslation()
  return (
    <NativeSelect
      aria-label={t('Reseller tier')}
      value={props.value}
      onChange={(e) => props.onChange(Number(e.target.value))}
    >
      {props.all && (
        <NativeSelectOption value={-1}>{t('All tiers')}</NativeSelectOption>
      )}
      {props.all && (
        <NativeSelectOption value={0}>{t('Unassigned')}</NativeSelectOption>
      )}
      {[1, 2, 3].map((tier) => (
        <NativeSelectOption key={tier} value={tier}>
          {t('Tier {{tier}}', { tier })}
        </NativeSelectOption>
      ))}
    </NativeSelect>
  )
}

export function TierConfirmation(props: {
  change: TierChange
  preview: TierPreview
  onClose: () => void
  onSaved: () => void
  services?: GuardianService[]
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const save = useMutation({
    mutationFn: () =>
      resellerWrite(
        '/admin/tier-supplies',
        { ...props.change, preview_token: props.preview.token },
        'post'
      ),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['guardian'] })
      toast.success(t('Saved'))
      props.onSaved()
    },
  })
  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.onClose()
      }}
      title={t('Confirm supply changes')}
      isLoading={save.isPending}
      handleConfirm={() => save.mutate()}
      desc={
        <div className='max-h-[60vh] space-y-3 overflow-y-auto'>
          <p>
            {t('Tier {{tier}}', { tier: props.change.tier })} ·{' '}
            {t('Affected resellers: {{count}}', {
              count: props.preview.resellers,
            })}
          </p>
          {props.preview.reseller_ids?.length ? (
            <p>
              {t('Reseller')}:{' '}
              {props.preview.reseller_ids.map((id) => `#${id}`).join(', ')}
            </p>
          ) : null}
          <p>
            {t('Independent product configurations: {{count}}', {
              count: props.preview.independent,
            })}
          </p>
          <p>
            {t('Listed products requiring new retail prices: {{count}}', {
              count: props.preview.unavailable,
            })}
          </p>
          <p>
            {props.change.restore
              ? t(
                  'Selected independent prices will be replaced by tier prices.'
                )
              : t('Independent prices and existing orders stay unchanged.')}
          </p>
          {(props.preview.offer_changes ?? []).map((change) => (
            <div
              key={`${change.reseller_id}-${change.service_id}`}
              className='rounded border p-2'
            >
              <strong>
                {t('Reseller')} #{change.reseller_id} · {t('Service')} #
                {change.service_id}
              </strong>
              <p>
                {change.before.granted ? t('Enabled') : t('Disabled')} →{' '}
                {change.after.granted ? t('Enabled') : t('Disabled')}
              </p>
              <SupplyPriceChanges before={change.before} after={change.after} />
            </div>
          ))}
          {(props.preview.after ?? []).map((p) => {
            const before = props.preview.before?.find(
              (b) => b.service_id === p.service_id
            )
            return (
              <div key={p.service_id} className='rounded border p-2'>
                <strong>
                  {props.services?.find((s) => s.id === p.service_id)?.title ??
                    `#${p.service_id}`}
                </strong>
                <p>
                  {before?.granted ? t('Enabled') : t('Disabled')} →{' '}
                  {p.granted ? t('Enabled') : t('Disabled')}
                </p>
                <SupplyPriceChanges before={before} after={p} />
              </div>
            )
          })}
        </div>
      }
    />
  )
}

export function GuardianTierPrices() {
  const { t } = useTranslation()
  const [tier, setTier] = useState(1)
  const products = useQuery({
    queryKey: ['guardian', 'admin-services'],
    queryFn: listAdminGuardianServices,
  })
  const prices = useQuery({
    queryKey: ['guardian', 'tier-supplies'],
    queryFn: () => resellerRead<TierPrice[]>('/admin/tier-supplies'),
  })
  return (
    <div className='space-y-4'>
      <TierSelect value={tier} onChange={setTier} />
      {(products.isPending || prices.isPending) && <LoadingState />}
      {(products.isError || prices.isError) && (
        <ErrorState
          onRetry={() => {
            void products.refetch()
            void prices.refetch()
          }}
        />
      )}
      {products.data && prices.data && (
        <TierPriceForm
          key={`${tier}-${prices.dataUpdatedAt}`}
          tier={tier}
          products={products.data}
          prices={prices.data.filter((p) => p.tier === tier)}
        />
      )}
      <p className='text-muted-foreground text-sm'>
        {t(
          'Tier prices apply to future orders. Zero closes an optional specification.'
        )}
      </p>
    </div>
  )
}

function TierPriceForm(props: {
  tier: number
  products: GuardianService[]
  prices: TierPrice[]
}) {
  const { t } = useTranslation()
  const [values, setValues] = useState(() =>
    props.products.map((product) => {
      const price = props.prices.find((p) => p.service_id === product.id)
      return {
        service_id: product.id,
        granted: price?.granted ?? false,
        amounts: [
          price?.supply_cents,
          price?.quarter_supply_cents,
          price?.annual_supply_cents,
          price?.pro_supply_cents,
          price?.max_supply_cents,
          price?.ultra_supply_cents,
        ].map((n) => (n ? centsToYuanInput(n) : '')),
      }
    })
  )
  const [confirmation, setConfirmation] = useState<{
    change: TierChange
    preview: TierPreview
  } | null>(null)
  const preview = useMutation({
    mutationFn: (change: TierChange) =>
      resellerWrite<TierPreview>('/admin/tier-supplies', change, 'post'),
    onSuccess: (result, change) => setConfirmation({ change, preview: result }),
  })
  const labels = [
    t('Single / monthly'),
    t('Quarterly'),
    t('Yearly'),
    t('Claude Pro subscription'),
    t('Claude MAX subscription'),
    t('Claude Ultra subscription'),
  ]
  const invalid = values.some(
    (row) =>
      row.amounts.some(
        (v) => v !== '' && v !== '0' && yuanToCents(v) === null
      ) ||
      (row.granted && !yuanToCents(row.amounts[0]))
  )
  return (
    <form
      className='space-y-4'
      onSubmit={(e) => {
        e.preventDefault()
        if (invalid) return
        preview.mutate({
          tier: props.tier,
          prices: values.map((row) => {
            const a = row.amounts.map((v) => yuanToCents(v) ?? 0)
            return {
              tier: props.tier,
              service_id: row.service_id,
              granted: row.granted,
              supply_cents: a[0],
              quarter_supply_cents: a[1],
              annual_supply_cents: a[2],
              pro_supply_cents: a[3],
              max_supply_cents: a[4],
              ultra_supply_cents: a[5],
            }
          }),
        })
      }}
    >
      {values.map((row, i) => (
        <fieldset
          key={row.service_id}
          className='space-y-3 rounded-lg border p-4'
        >
          <legend className='px-1 font-medium'>
            {props.products[i].title}
          </legend>
          <label className='flex items-center gap-2'>
            <Checkbox
              checked={row.granted}
              onCheckedChange={(v) =>
                setValues(
                  values.map((r, j) => (j === i ? { ...r, granted: !!v } : r))
                )
              }
            />
            {t('Authorized for sale')}
          </label>
          <div className='grid gap-3 sm:grid-cols-3 lg:grid-cols-3'>
            {labels.map((label, k) => {
              if (k > 0 && props.products[i].route !== 'mac_mini') return null
              return (
                <Field key={label}>
                  <FieldLabel htmlFor={`tier-${row.service_id}-${k}`}>
                    {label} · CNY
                  </FieldLabel>
                  <Input
                    id={`tier-${row.service_id}-${k}`}
                    inputMode='decimal'
                    value={row.amounts[k]}
                    onChange={(e) =>
                      setValues(
                        values.map((r, j) =>
                          j === i
                            ? {
                                ...r,
                                amounts: r.amounts.map((v, n) =>
                                  n === k ? e.target.value : v
                                ),
                              }
                            : r
                        )
                      )
                    }
                  />
                </Field>
              )
            })}
          </div>
        </fieldset>
      ))}
      {invalid && (
        <p role='alert'>{t('Enter a price between 0.01 and 100000 yuan')}</p>
      )}
      <Button
        type='submit'
        disabled={invalid || preview.isPending || values.length === 0}
      >
        {t('Preview changes')}
      </Button>
      {confirmation && (
        <TierConfirmation
          {...confirmation}
          services={props.products}
          onClose={() => setConfirmation(null)}
          onSaved={() => setConfirmation(null)}
        />
      )}
    </form>
  )
}

function SupplyPriceChanges(props: {
  before?: Omit<TierPrice, 'tier'>
  after: Omit<TierPrice, 'tier'>
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const fields = [
    ['supply_cents', t('Single / monthly')],
    ['quarter_supply_cents', t('Quarterly')],
    ['annual_supply_cents', t('Yearly')],
    ['max_supply_cents', t('Claude MAX subscription')],
    ['ultra_supply_cents', t('Claude Ultra subscription')],
    ['pro_supply_cents', t('Claude Pro subscription')],
  ] as const
  return fields.map(([key, label]) => (
    <p key={key}>
      {label}: {formatCny(props.before?.[key] ?? 0, locale)} →{' '}
      {formatCny(props.after[key], locale)}
    </p>
  ))
}
