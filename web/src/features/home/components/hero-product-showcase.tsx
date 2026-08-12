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
import { Download, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import {
  heroProductShowcaseClasses,
  heroProductShowcaseMedia,
  type ShowcaseDevice,
} from './hero-product-showcase-config'

export function HeroProductShowcase() {
  const { t } = useTranslation()
  const labels: Record<ShowcaseDevice, string> = {
    desktop: t('Desktop Client'),
    web: t('Web Console'),
    'wechat-mini-program': t('WeChat Mini Program'),
  }

  return (
    <div className={heroProductShowcaseClasses.stage}>
      {heroProductShowcaseMedia.map((item) => {
        const Icon = item.icon
        const href = 'href' in item ? item.href : undefined
        const labelIconSrc =
          'labelIconSrc' in item ? item.labelIconSrc : undefined
        const storeIconSrc =
          'storeIconSrc' in item ? item.storeIconSrc : undefined
        const linkLabel =
          item.id === 'desktop'
            ? t('Download PccAgent from Microsoft Store')
            : t('View Pricing')

        return (
          <figure
            key={item.id}
            data-showcase-device={item.id}
            className={cn(
              'hero-device-rise group absolute will-change-transform',
              item.positionClassName
            )}
            style={{ animationDelay: item.animationDelay }}
          >
            <div
              className={cn(
                'relative transform-gpu transition-transform duration-500 ease-out',
                item.surfaceClassName
              )}
            >
              {href ? (
                <a
                  href={href}
                  target='_blank'
                  rel='noopener noreferrer'
                  aria-label={linkLabel}
                  title={linkLabel}
                  className='focus-visible:ring-primary absolute inset-0 z-20 rounded-[8px] focus-visible:ring-2 focus-visible:ring-offset-4 focus-visible:outline-none'
                />
              ) : null}
              <figcaption
                className={cn(
                  'border-border/60 bg-background/92 text-foreground pointer-events-none absolute z-30 flex h-8 items-center gap-2 whitespace-nowrap rounded-md border px-2.5 text-xs font-medium shadow-sm backdrop-blur-md',
                  item.labelClassName
                )}
              >
                {labelIconSrc ? (
                  <img
                    src={labelIconSrc}
                    alt=''
                    aria-hidden='true'
                    width={28}
                    height={28}
                    className='size-3.5 shrink-0'
                  />
                ) : (
                  <Icon
                    aria-hidden='true'
                    className='text-primary size-3.5 shrink-0'
                  />
                )}
                <span>{labels[item.id]}</span>
                {storeIconSrc ? (
                  <>
                    <span aria-hidden='true' className='text-border'>
                      ·
                    </span>
                    <img
                      src={storeIconSrc}
                      alt=''
                      aria-hidden='true'
                      width={16}
                      height={16}
                      className='size-3.5 shrink-0'
                    />
                    <span>PccAgent</span>
                    <Download
                      aria-hidden='true'
                      className='size-3.5 shrink-0'
                    />
                  </>
                ) : null}
              </figcaption>
              {item.id === 'desktop' ? (
                <div className={heroProductShowcaseClasses.promotion}>
                  <span className='bg-primary text-primary-foreground inline-flex shrink-0 items-center gap-1 rounded-sm px-1.5 py-0.5 font-semibold'>
                    <Sparkles aria-hidden='true' className='size-3' />
                    {t('Discount')}
                  </span>
                  <span>
                    {t('Use the official Agent to get one month of Lite free')}
                  </span>
                </div>
              ) : null}
              <img
                src={item.src}
                alt={labels[item.id]}
                width={item.width}
                height={item.height}
                loading='eager'
                decoding='async'
                draggable={false}
                className={cn(
                  heroProductShowcaseClasses.image,
                  item.id === 'wechat-mini-program'
                    ? 'rounded-[1.4rem] shadow-[0_28px_70px_-24px_rgba(15,23,42,0.42)]'
                    : heroProductShowcaseClasses.imageFrame
                )}
              />
            </div>
          </figure>
        )
      })}
    </div>
  )
}
