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
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Skeleton } from '@/components/ui/skeleton'
import {
  MAX_RANKING_ADDED_TOKENS,
  parseRankingDisplayConfig,
  serializeRankingDisplayConfig,
} from '@/features/rankings/display-config'
import { useLiveRankings } from '@/features/rankings/hooks/use-rankings'

import {
  SettingsSwitchField,
  SettingsControlGroup,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { RankingAdjustmentRow } from './ranking-adjustment-row'

type RankingsDisplaySectionProps = {
  value: string
}

function getTodayDate() {
  return new Date().toISOString().slice(0, 10)
}

export function RankingsDisplaySection(props: RankingsDisplaySectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialConfig = useMemo(
    () => parseRankingDisplayConfig(props.value),
    [props.value]
  )
  const initialSerialized = useMemo(
    () => serializeRankingDisplayConfig(initialConfig),
    [initialConfig]
  )
  const [config, setConfig] = useState(initialConfig)
  const [date, setDate] = useState(getTodayDate)
  const rankingsQuery = useLiveRankings('today', date)
  const serialized = serializeRankingDisplayConfig(config)

  useEffect(() => {
    setConfig(initialConfig)
  }, [initialConfig])

  const updateAdjustment = (
    modelName: string,
    liveTokens: number,
    rawValue: string
  ) => {
    const addedTokens = rawValue === '' ? 0 : Number(rawValue)
    if (
      !Number.isSafeInteger(addedTokens) ||
      addedTokens < 0 ||
      addedTokens > MAX_RANKING_ADDED_TOKENS - liveTokens
    ) {
      toast.error(t('Enter a non-negative whole number.'))
      return
    }

    setConfig((current) => {
      const adjustments = {
        ...current.daily_records[date]?.adjustments,
        [modelName]: addedTokens,
      }
      return {
        ...current,
        daily_records: {
          ...current.daily_records,
          [date]: { adjustments },
        },
      }
    })
  }

  const save = async () => {
    await updateOption.mutateAsync({
      key: 'RankingDisplayConfig',
      value: serialized,
    })
  }

  const liveRows = rankingsQuery.data?.data.models ?? []
  const savedModels = Object.keys(
    config.daily_records[date]?.adjustments ?? {}
  ).map((modelName) => ({
    model_name: modelName,
    total_tokens: 0,
    vendor: 'Unknown',
  }))
  const liveModelNames = new Set(liveRows.map((row) => row.model_name))
  const rows = [
    ...liveRows,
    ...savedModels.filter((row) => !liveModelNames.has(row.model_name)),
  ]
  let rowsContent: ReactNode

  if (rankingsQuery.isLoading) {
    rowsContent = (
      <div className='space-y-2'>
        <Skeleton className='h-10 w-full' />
        <Skeleton className='h-10 w-full' />
        <Skeleton className='h-10 w-full' />
      </div>
    )
  } else if (rankingsQuery.isError) {
    rowsContent = (
      <p className='text-destructive text-sm'>
        {t('Unable to load rankings data')}
      </p>
    )
  } else if (rows.length === 0) {
    rowsContent = (
      <p className='text-muted-foreground text-sm'>
        {t('No ranking models are available for this period.')}
      </p>
    )
  } else {
    rowsContent = (
      <div className='rounded-lg border'>
        <div className='bg-muted/40 text-muted-foreground grid grid-cols-[minmax(0,1fr)_7rem] gap-3 px-3 py-2 text-xs font-medium md:grid-cols-[minmax(0,1fr)_9rem_minmax(9rem,12rem)_9rem]'>
          <span>{t('Model')}</span>
          <span className='text-right'>{t('Live total')}</span>
          <span className='hidden md:block'>{t('Added tokens')}</span>
          <span className='hidden text-right md:block'>
            {t('Final display')}
          </span>
        </div>
        <div className='divide-y'>
          {rows.map((row) => {
            const addedTokens =
              config.daily_records[date]?.adjustments[row.model_name] ?? 0

            return (
              <RankingAdjustmentRow
                key={row.model_name}
                modelName={row.model_name}
                liveTokens={row.total_tokens}
                addedTokens={addedTokens}
                enabled={config.enabled}
                onAdjustmentChange={updateAdjustment}
              />
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <SettingsSection title={t('Rankings display')}>
      <SettingsPageFormActions
        onSave={() => void save()}
        onReset={() => setConfig(initialConfig)}
        isSaving={updateOption.isPending}
        isSaveDisabled={serialized === initialSerialized}
        isResetDisabled={serialized === initialSerialized}
        saveLabel='Save Changes'
        resetLabel='Reset'
      />

      <SettingsSwitchField
        checked={config.enabled}
        onCheckedChange={(enabled) =>
          setConfig((current) => ({ ...current, enabled }))
        }
        label={t('Enable calibrated rankings')}
        description={t(
          'Added tokens are applied on top of live usage totals, so displayed totals can never fall below live data.'
        )}
      />

      <SettingsControlGroup className='space-y-4'>
        <div className='space-y-1'>
          <p className='text-sm font-medium'>{t('Displayed data policy')}</p>
          <p className='text-muted-foreground text-xs leading-5'>
            {t(
              'Daily calibration records are stored by date and automatically included in today, week, month, and year totals, trends, and charts.'
            )}
          </p>
        </div>

        <label className='flex max-w-sm flex-col gap-2 text-sm font-medium'>
          {t('Calibration date')}
          <input
            type='date'
            value={date}
            onChange={(event) => setDate(event.currentTarget.value)}
            className='border-input bg-background h-9 rounded-md border px-3 font-mono text-sm'
          />
        </label>

        {rowsContent}
      </SettingsControlGroup>
    </SettingsSection>
  )
}
