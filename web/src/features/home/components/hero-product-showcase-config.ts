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
import { Globe2, Monitor, Smartphone, type LucideIcon } from 'lucide-react'

export type ShowcaseDevice = 'desktop' | 'web' | 'wechat-mini-program'

interface ShowcaseMedia {
  id: ShowcaseDevice
  src: string
  width: number
  height: number
  icon: LucideIcon
  positionClassName: string
  surfaceClassName: string
  labelClassName: string
  animationDelay: string
  href?: string
  labelIconSrc?: string
  storeIconSrc?: string
}

export const heroProductShowcaseClasses = {
  stage:
    'relative mx-auto mt-24 hidden h-[min(40vw,44svh,30rem)] min-h-[20rem] w-full max-w-7xl min-[700px]:block',
  image: 'block h-auto w-full select-none object-contain',
  imageFrame:
    'rounded-[8px] border border-black/10 shadow-[0_28px_80px_-30px_rgba(15,23,42,0.38)]',
  promotion:
    'border-primary/25 bg-background/94 text-foreground pointer-events-none absolute top-14 right-3 z-30 flex max-w-[calc(100%-1.5rem)] items-center gap-2 rounded-md border px-2 py-1.5 text-[11px] leading-4 font-medium shadow-sm backdrop-blur-md',
} as const

export const heroProductShowcaseMedia = [
  {
    id: 'web',
    src: '/landing/site.webp',
    width: 1800,
    height: 1178,
    icon: Globe2,
    positionClassName:
      'bottom-0 left-1/2 z-10 w-[66%] min-[1100px]:bottom-auto min-[1100px]:-top-[35px]',
    surfaceClassName: '-translate-x-1/2 group-hover:-translate-y-2',
    labelClassName: 'top-3 left-3',
    animationDelay: '280ms',
    href: 'https://api.dpccgaming.xyz/pricing',
    labelIconSrc: '/landing/dpcc-api-favicon.png',
  },
  {
    id: 'desktop',
    src: '/landing/desktop.webp',
    width: 1800,
    height: 1200,
    icon: Monitor,
    positionClassName:
      'bottom-[-18%] left-[1%] z-20 w-[50%] min-[1100px]:w-[48%]',
    surfaceClassName: 'group-hover:-translate-y-3',
    labelClassName: 'top-3 right-3',
    animationDelay: '420ms',
    href: 'https://apps.microsoft.com/detail/9pf5ff13cbhp?hl=zh-CN&gl=CN',
    storeIconSrc: '/landing/microsoft-store.png',
  },
  {
    id: 'wechat-mini-program',
    src: '/landing/app.webp',
    width: 640,
    height: 1387,
    icon: Smartphone,
    positionClassName:
      'right-[2%] bottom-[-11%] z-30 w-[14%] min-[1100px]:w-[13%]',
    surfaceClassName: 'group-hover:-translate-y-4',
    labelClassName: '-top-10 right-0',
    animationDelay: '540ms',
  },
] as const satisfies readonly ShowcaseMedia[]
