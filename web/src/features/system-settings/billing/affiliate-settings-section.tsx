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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { getCurrencyLabel } from '@/lib/currency'
import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import { updateAffiliateSetting } from '../api'
import { SettingsSection } from '../components/settings-section'

export type AffiliateSettingForm = {
  registration_reward_enabled: boolean
  inviter_reward_quota: number
  invitee_reward_quota: number
  commission_enabled: boolean
  qualification_threshold_minor: string
  commission_rate_bps: number
  commission_wait_days: number
  version: number
}

const DEFAULT_AFFILIATE_SETTING: AffiliateSettingForm = {
  registration_reward_enabled: true,
  inviter_reward_quota: 0,
  invitee_reward_quota: 0,
  commission_enabled: false,
  qualification_threshold_minor: '10000',
  commission_rate_bps: 0,
  commission_wait_days: 7,
  version: 1,
}

function parseAffiliateSetting(value: string): AffiliateSettingForm {
  try {
    const parsed = JSON.parse(value) as Partial<AffiliateSettingForm>
    const threshold = value.match(
      /"qualification_threshold_minor"\s*:\s*(-?\d+)/
    )?.[1]
    return {
      ...DEFAULT_AFFILIATE_SETTING,
      ...parsed,
      qualification_threshold_minor:
        threshold ?? DEFAULT_AFFILIATE_SETTING.qualification_threshold_minor,
    }
  } catch {
    return DEFAULT_AFFILIATE_SETTING
  }
}

function hasValidSafeInteger(
  value: number,
  min: number,
  max?: number
): boolean {
  return (
    Number.isSafeInteger(value) &&
    value >= min &&
    (max === undefined || value <= max)
  )
}

export function AffiliateSettingsSection(props: { defaultValue: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [setting, setSetting] = useState(() =>
    parseAffiliateSetting(props.defaultValue)
  )

  useEffect(() => {
    setSetting(parseAffiliateSetting(props.defaultValue))
  }, [props.defaultValue])

  const saveMutation = useMutation({
    mutationFn: () => updateAffiliateSetting(setting),
    onSuccess: (result) => {
      if (result.success && result.data) {
        setSetting(result.data)
        void queryClient.invalidateQueries({ queryKey: ['system-options'] })
        toast.success(t('Setting updated successfully'))
        return
      }
      toast.error(result.message || t('Failed to update setting'))
    },
    onError: () => toast.error(t('Failed to update setting')),
  })

  const valid =
    hasValidSafeInteger(setting.inviter_reward_quota, 0) &&
    hasValidSafeInteger(setting.invitee_reward_quota, 0) &&
    /^\d+$/.test(setting.qualification_threshold_minor) &&
    hasValidSafeInteger(setting.commission_rate_bps, 0, 10000) &&
    hasValidSafeInteger(setting.commission_wait_days, 0, 3650)

  const setNumber = (key: keyof AffiliateSettingForm, value: string) => {
    const number = Number(value)
    setSetting((current) => ({
      ...current,
      [key]: Number.isNaN(number) ? 0 : number,
    }))
  }

  const setRewardAmount = (
    key: 'inviter_reward_quota' | 'invitee_reward_quota',
    value: string
  ) => {
    const amount = Number(value)
    setSetting((current) => ({
      ...current,
      [key]: Number.isFinite(amount) ? parseQuotaFromDollars(amount) : 0,
    }))
  }

  const setCommissionRatePercent = (value: string) => {
    const percentage = Number(value)
    setSetting((current) => ({
      ...current,
      commission_rate_bps: Number.isFinite(percentage)
        ? Math.round(percentage * 100)
        : 0,
    }))
  }

  return (
    <SettingsSection title={t('Affiliate Program')}>
      <div className='grid gap-5'>
        <div className='flex items-center justify-between gap-4 rounded-lg border p-4'>
          <div>
            <p className='font-medium'>{t('Registration rewards')}</p>
            <p className='text-muted-foreground text-sm'>
              {t('Reward users and inviters after registration.')}
            </p>
          </div>
          <Switch
            checked={setting.registration_reward_enabled}
            onCheckedChange={(value) =>
              setSetting((current) => ({
                ...current,
                registration_reward_enabled: value,
              }))
            }
          />
        </div>
        <div className='grid gap-4 sm:grid-cols-2'>
          <label className='grid gap-2 text-sm font-medium'>
            {t('Inviter Reward')} ({getCurrencyLabel()})
            <Input
              type='number'
              min='0'
              step='any'
              value={quotaUnitsToDollars(setting.inviter_reward_quota)}
              onChange={(event) =>
                setRewardAmount(
                  'inviter_reward_quota',
                  event.currentTarget.value
                )
              }
            />
          </label>
          <label className='grid gap-2 text-sm font-medium'>
            {t('Invitee Reward')} ({getCurrencyLabel()})
            <Input
              type='number'
              min='0'
              step='any'
              value={quotaUnitsToDollars(setting.invitee_reward_quota)}
              onChange={(event) =>
                setRewardAmount(
                  'invitee_reward_quota',
                  event.currentTarget.value
                )
              }
            />
          </label>
        </div>
        <div className='flex items-center justify-between gap-4 rounded-lg border p-4'>
          <div>
            <p className='font-medium'>{t('Top-up commission')}</p>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Allow qualified users to earn from verified online payments.'
              )}
            </p>
          </div>
          <Switch
            checked={setting.commission_enabled}
            onCheckedChange={(value) =>
              setSetting((current) => ({
                ...current,
                commission_enabled: value,
              }))
            }
          />
        </div>
        <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
          <label className='grid gap-2 text-sm font-medium'>
            {t('Qualification threshold (minor units)')}
            <Input
              inputMode='numeric'
              value={setting.qualification_threshold_minor}
              onChange={(event) => {
                const sanitizedValue = event.currentTarget.value.replaceAll(
                  /\D/g,
                  ''
                )
                setSetting((current) => ({
                  ...current,
                  qualification_threshold_minor: sanitizedValue,
                }))
              }}
            />
          </label>
          <label className='grid gap-2 text-sm font-medium'>
            {t('Commission Rate')} (%)
            <Input
              type='number'
              min='0'
              max='100'
              step='0.01'
              value={setting.commission_rate_bps / 100}
              onChange={(event) =>
                setCommissionRatePercent(event.currentTarget.value)
              }
            />
          </label>
          <label className='grid gap-2 text-sm font-medium'>
            {t('Commission wait days')}
            <Input
              type='number'
              min='0'
              max='3650'
              value={setting.commission_wait_days}
              onChange={(event) =>
                setNumber('commission_wait_days', event.currentTarget.value)
              }
            />
          </label>
        </div>
        <div className='flex justify-end'>
          <Button
            disabled={!valid || saveMutation.isPending}
            onClick={() => saveMutation.mutate()}
          >
            {t('Save Changes')}
          </Button>
        </div>
      </div>
    </SettingsSection>
  )
}
