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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { getPricing } from '@/features/pricing/api'
import { isTokenBasedModel } from '@/features/pricing/lib/model-helpers'
import {
  MAX_SUBSCRIPTION_DISPLAY_MODELS,
  SUBSCRIPTION_ESTIMATE_CACHE_HIT_RATE,
  SUBSCRIPTION_ESTIMATE_INPUT_TOKENS,
  SUBSCRIPTION_ESTIMATE_OUTPUT_TOKENS,
  parseSubscriptionDisplayModels,
  serializeSubscriptionDisplayModels,
} from '@/features/pricing/lib/subscription-estimate'

import { SettingsControlGroup } from '../components/settings-form-layout'

const EMPTY_MODEL_VALUE = '__subscription_estimate_none__'
const MODEL_SLOT_IDS = ['first', 'second', 'third'] as const

type SubscriptionDisplayModelsFieldProps = {
  value: string
  onChange: (value: string) => void
}

function toModelSlots(value: string): string[] {
  const selected = parseSubscriptionDisplayModels(value)
  return Array.from(
    { length: MAX_SUBSCRIPTION_DISPLAY_MODELS },
    (_, index) => selected[index] || ''
  )
}

export function SubscriptionDisplayModelsField(
  props: SubscriptionDisplayModelsFieldProps
) {
  const { t } = useTranslation()
  const initialSlots = useMemo(() => toModelSlots(props.value), [props.value])
  const [slots, setSlots] = useState(initialSlots)
  const pricingQuery = useQuery({
    queryKey: ['pricing'],
    queryFn: getPricing,
    staleTime: 5 * 60 * 1000,
  })

  useEffect(() => {
    setSlots(initialSlots)
  }, [initialSlots])

  const tokenModels = useMemo(
    () =>
      (pricingQuery.data?.data ?? []).filter(
        (model) => isTokenBasedModel(model) && model.model_name.trim()
      ),
    [pricingQuery.data]
  )

  const modelOptions = useMemo(() => {
    const availableNames = new Set(tokenModels.map((model) => model.model_name))
    const unavailableNames = slots.filter(
      (name) => name && !availableNames.has(name)
    )
    const unavailableOptions = unavailableNames.map((name) => ({
      name,
      unavailable: true,
    }))
    const availableOptions = tokenModels.map((model) => ({
      name: model.model_name,
      unavailable: false,
    }))

    return [...unavailableOptions, ...availableOptions]
  }, [slots, tokenModels])

  let modelSelectorContent
  if (pricingQuery.isLoading) {
    modelSelectorContent = (
      <div className='grid gap-3 sm:grid-cols-3'>
        {MODEL_SLOT_IDS.map((id) => (
          <Skeleton key={id} className='h-8 w-full' />
        ))}
      </div>
    )
  } else if (pricingQuery.isError) {
    modelSelectorContent = (
      <p className='text-destructive text-sm'>{t('Failed to fetch models')}</p>
    )
  } else if (tokenModels.length === 0) {
    modelSelectorContent = (
      <p className='text-muted-foreground text-sm'>
        {t('No models available')}
      </p>
    )
  } else {
    modelSelectorContent = (
      <div className='grid gap-3 sm:grid-cols-3'>
        {slots.map((slot, index) => (
          <label key={MODEL_SLOT_IDS[index]} className='grid min-w-0 gap-1.5'>
            <span className='text-muted-foreground text-xs font-medium'>
              {t('Model slot {{number}}', { number: index + 1 })}
            </span>
            <Select
              value={slot || EMPTY_MODEL_VALUE}
              onValueChange={(value) => handleSlotChange(index, value)}
            >
              <SelectTrigger className='w-full min-w-0'>
                <SelectValue placeholder={t('Select a model')} />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value={EMPTY_MODEL_VALUE}>
                    {t('No model')}
                  </SelectItem>
                  {modelOptions.map((option) => (
                    <SelectItem
                      key={option.name}
                      value={option.name}
                      disabled={
                        !option.unavailable &&
                        slots.some(
                          (selected, selectedIndex) =>
                            selectedIndex !== index && selected === option.name
                        )
                      }
                    >
                      {option.unavailable
                        ? `${option.name} (${t('Not available')})`
                        : option.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </label>
        ))}
      </div>
    )
  }

  const handleSlotChange = (index: number, value: string | null) => {
    if (value === null) return

    const nextSlots = [...slots]
    nextSlots[index] = value === EMPTY_MODEL_VALUE ? '' : value
    setSlots(nextSlots)
    props.onChange(
      serializeSubscriptionDisplayModels(
        nextSlots.filter((name): name is string => Boolean(name))
      )
    )
  }

  return (
    <SettingsControlGroup
      className='space-y-4 lg:col-span-2'
      data-subscription-display-models-setting='true'
    >
      <div className='space-y-1'>
        <p className='text-sm font-medium'>
          {t('Subscription token estimate')}
        </p>
        <p className='text-muted-foreground text-xs leading-5'>
          {t(
            'Choose up to three models to show token estimates on subscription cards.'
          )}
        </p>
      </div>

      {modelSelectorContent}

      <p className='text-muted-foreground text-xs leading-5'>
        {t(
          'Based on {{input}} input tokens, {{output}} output tokens, and {{cache}}% cache hits per sample.',
          {
            input: SUBSCRIPTION_ESTIMATE_INPUT_TOKENS.toLocaleString(),
            output: SUBSCRIPTION_ESTIMATE_OUTPUT_TOKENS.toLocaleString(),
            cache: SUBSCRIPTION_ESTIMATE_CACHE_HIT_RATE * 100,
          }
        )}
      </p>
    </SettingsControlGroup>
  )
}
