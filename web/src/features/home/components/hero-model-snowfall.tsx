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
import type { CSSProperties } from 'react'

import { getLobeIcon } from '@/lib/lobe-icon'

const heroModelLogos = [
  'OpenAI',
  'Claude.Color',
  'DeepSeek.Color',
  'Kimi.Avatar',
  'Grok',
  'Gemini.Color',
] as const

const desktopFlakeCount = 48
const modelLogoSize = 48
const snowfallDuration = 52
const desktopLaneCount = 14
const desktopSlotCount = 4
const tabletLaneCount = 9
const tabletSlotCount = 6
const mobileLaneCount = 5
const mobileSlotCount = 8
const snowfallLayout = Array.from({ length: desktopFlakeCount }, (_, index) => {
  const desktopLane = index % desktopLaneCount
  const desktopSlot = Math.floor(index / desktopLaneCount)
  const desktopLeft = desktopLane / (desktopLaneCount - 1)
  const desktopTop = desktopSlot / (desktopSlotCount - 1)
  const tabletLane = index % tabletLaneCount
  const tabletSlot = Math.floor(index / tabletLaneCount)
  const tabletLeft = tabletLane / (tabletLaneCount - 1)
  const tabletTop = tabletSlot / (tabletSlotCount - 1)
  const mobileLane = index % mobileLaneCount
  const mobileSlot = Math.floor(index / mobileLaneCount)
  const mobileLeft = mobileLane / (mobileLaneCount - 1)
  const mobileTop = mobileSlot / (mobileSlotCount - 1)

  return {
    desktopDelay: -(
      2 +
      (desktopSlot + (desktopLane % 4) / 4) *
        (snowfallDuration / desktopSlotCount)
    ),
    desktopLeft: `calc(${desktopLeft * 100}% - ${desktopLeft * modelLogoSize}px)`,
    desktopTop: `calc(${desktopTop * 100}% - ${desktopTop * modelLogoSize}px)`,
    mobileDelay: -(
      2 +
      (mobileSlot + (mobileLane % 5) / 5) * (snowfallDuration / mobileSlotCount)
    ),
    mobileLeft: `calc(${mobileLeft * 100}% - ${mobileLeft * modelLogoSize}px)`,
    mobileTop: `calc(${mobileTop * 100}% - ${mobileTop * modelLogoSize}px)`,
    tabletDelay: -(
      2 +
      (tabletSlot + (tabletLane % 3) / 3) * (snowfallDuration / tabletSlotCount)
    ),
    tabletLeft: `calc(${tabletLeft * 100}% - ${tabletLeft * modelLogoSize}px)`,
    tabletTop: `calc(${tabletTop * 100}% - ${tabletTop * modelLogoSize}px)`,
  }
})

type SnowfallStyle = CSSProperties & {
  '--hero-model-delay': string
  '--hero-model-delay-mobile': string
  '--hero-model-delay-tablet': string
  '--hero-model-drift': string
  '--hero-model-drift-mid': string
  '--hero-model-duration': string
  '--hero-model-left': string
  '--hero-model-left-mobile': string
  '--hero-model-left-tablet': string
  '--hero-model-opacity': string
  '--hero-model-rotate': string
  '--hero-model-rotate-mid': string
  '--hero-model-size': string
  '--hero-model-static-top': string
  '--hero-model-static-top-mobile': string
  '--hero-model-static-top-tablet': string
}

export function HeroModelSnowfall() {
  return (
    <div
      data-hero-model-snowfall='true'
      aria-hidden='true'
      className='hero-model-snowfall pointer-events-none absolute inset-0 z-0 overflow-hidden'
    >
      {snowfallLayout.map((flake, index) => {
        const icon = heroModelLogos[index % heroModelLogos.length]
        const style: SnowfallStyle = {
          '--hero-model-delay': `${flake.desktopDelay}s`,
          '--hero-model-delay-mobile': `${flake.mobileDelay}s`,
          '--hero-model-delay-tablet': `${flake.tabletDelay}s`,
          '--hero-model-drift': '0px',
          '--hero-model-drift-mid': '0px',
          '--hero-model-duration': `${snowfallDuration}s`,
          '--hero-model-left': flake.desktopLeft,
          '--hero-model-left-mobile': flake.mobileLeft,
          '--hero-model-left-tablet': flake.tabletLeft,
          '--hero-model-opacity': `${0.28 + (index % 5) * 0.03}`,
          '--hero-model-rotate': '0deg',
          '--hero-model-rotate-mid': '0deg',
          '--hero-model-size': `${modelLogoSize}px`,
          '--hero-model-static-top': flake.desktopTop,
          '--hero-model-static-top-mobile': flake.mobileTop,
          '--hero-model-static-top-tablet': flake.tabletTop,
        }

        return (
          <span
            key={`${icon}-${flake.desktopLeft}-${flake.desktopDelay}`}
            data-model-icon={icon.split('.')[0]}
            className='hero-model-flake absolute grid place-items-center'
            style={style}
          >
            {getLobeIcon(icon, modelLogoSize)}
          </span>
        )
      })}
    </div>
  )
}
