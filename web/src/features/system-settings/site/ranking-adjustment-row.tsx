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
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import {
  getRankingDisplayTotal,
  MAX_RANKING_ADDED_TOKENS,
} from '@/features/rankings/display-config'
import { formatTokens } from '@/features/rankings/lib/format'

type RankingAdjustmentRowProps = {
  modelName: string
  liveTokens: number
  addedTokens: number
  enabled: boolean
  onAdjustmentChange: (
    modelName: string,
    liveTokens: number,
    rawValue: string
  ) => void
}

export function RankingAdjustmentRow(props: RankingAdjustmentRowProps) {
  const { t } = useTranslation()
  const displayTokens = getRankingDisplayTotal(
    props.liveTokens,
    props.addedTokens
  )

  return (
    <div className='grid grid-cols-[minmax(0,1fr)_7rem] items-center gap-3 px-3 py-2 md:grid-cols-[minmax(0,1fr)_9rem_minmax(9rem,12rem)_9rem]'>
      <span className='truncate font-mono text-sm'>{props.modelName}</span>
      <span className='text-muted-foreground text-right font-mono text-xs tabular-nums'>
        <span className='text-foreground block text-sm font-medium'>
          {formatTokens(props.liveTokens)}
        </span>
        {String(props.liveTokens)}
      </span>
      <label className='space-y-1'>
        <span className='text-muted-foreground text-xs md:hidden'>
          {t('Added tokens')}
        </span>
        <Input
          type='number'
          min={0}
          max={MAX_RANKING_ADDED_TOKENS - props.liveTokens}
          step={1}
          aria-label={t('Added tokens for {{model}}', {
            model: props.modelName,
          })}
          value={props.addedTokens || ''}
          onChange={(event) =>
            props.onAdjustmentChange(
              props.modelName,
              props.liveTokens,
              event.currentTarget.value
            )
          }
          placeholder='0'
          disabled={!props.enabled}
          className='font-mono tabular-nums'
        />
      </label>
      <span className='text-muted-foreground text-right font-mono text-xs tabular-nums'>
        <span className='mb-1 block text-left md:hidden'>
          {t('Final display')}
        </span>
        <span className='text-foreground block text-sm font-medium'>
          {formatTokens(displayTokens)}
        </span>
        {String(displayTokens)}
      </span>
    </div>
  )
}
