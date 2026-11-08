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
import { useId } from 'react'
import { SiApple } from 'react-icons/si'

import { cn } from '@/lib/utils'

import type { GuardianMacChip } from '../types'

export function MacChipIcon(props: {
  chip: GuardianMacChip
  className?: string
}) {
  const id = useId()
  const color = props.chip === 'm4' ? '#818cf8' : '#6ee7b7'
  return (
    <svg
      role='img'
      aria-label={`Apple ${props.chip.toUpperCase()}`}
      viewBox='0 0 160 160'
      className={cn('size-24 shrink-0', props.className)}
    >
      <defs>
        <radialGradient id={`${id}-glow`} cx='0' cy='1' r='1.2'>
          <stop offset='0' stopColor={color} />
          <stop
            offset='.3'
            stopColor={props.chip === 'm4' ? '#243d9b' : '#12556a'}
          />
          <stop offset='.72' stopColor='#030506' />
        </radialGradient>
        <linearGradient id={`${id}-edge`} x2='1' y2='1'>
          <stop stopColor='#d2e7e8' />
          <stop offset='.5' stopColor={color} />
          <stop offset='1' stopColor='#284858' />
        </linearGradient>
      </defs>
      <rect
        x='2'
        y='2'
        width='156'
        height='156'
        rx='2'
        fill={`url(#${id}-glow)`}
        stroke={`url(#${id}-edge)`}
        strokeWidth='3'
      />
      <rect
        x='5'
        y='5'
        width='150'
        height='150'
        rx='1'
        fill='none'
        stroke={color}
        strokeOpacity='.25'
      />
      <SiApple x='29' y='59' size={40} fill='#d5e8eb' aria-hidden='true' />
      <text
        x='69'
        y='97'
        fill='#d5e8eb'
        fontFamily='Arial, sans-serif'
        fontSize='43'
        fontWeight='600'
      >
        {props.chip.toUpperCase()}
      </text>
    </svg>
  )
}
