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
import Autoplay from 'embla-carousel-autoplay'
import { ArrowLeft, ArrowRight, Check, Download, Gift } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'
import {
  Carousel,
  type CarouselApi,
  CarouselContent,
  CarouselItem,
} from '@/components/ui/carousel'
import { PccAgentLogo } from '@/features/desktop-authorization/pcc-agent-logo'
import { cn } from '@/lib/utils'

import {
  PCC_AGENT_AUTOPLAY_OPTIONS,
  PCC_AGENT_SECTION_CLASSES,
  PCC_AGENT_SLIDE_DIMENSIONS,
  PCC_AGENT_SLIDE_SOURCES,
  PCC_AGENT_STORE_URL,
} from './pcc-agent-config'

export function PccAgent() {
  const { t } = useTranslation()
  const [autoplay] = useState(() => Autoplay(PCC_AGENT_AUTOPLAY_OPTIONS))
  const [api, setApi] = useState<CarouselApi>()
  const [selectedIndex, setSelectedIndex] = useState(0)

  const slides = [
    {
      src: PCC_AGENT_SLIDE_SOURCES[0],
      alt: t('PccAgent Agent marketplace preview'),
    },
    {
      src: PCC_AGENT_SLIDE_SOURCES[1],
      alt: t('PccAgent workspace preview'),
    },
    {
      src: PCC_AGENT_SLIDE_SOURCES[2],
      alt: t('PccAgent account analytics preview'),
    },
  ]

  const syncSelectedIndex = useCallback((carouselApi: CarouselApi) => {
    if (!carouselApi) return
    setSelectedIndex(carouselApi.selectedScrollSnap())
  }, [])

  useEffect(() => {
    if (!api) return

    syncSelectedIndex(api)
    api.on('select', syncSelectedIndex)
    api.on('reInit', syncSelectedIndex)

    return () => {
      api.off('select', syncSelectedIndex)
      api.off('reInit', syncSelectedIndex)
    }
  }, [api, syncSelectedIndex])

  useEffect(() => {
    if (!api) return

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    const syncReducedMotion = () => {
      if (reducedMotion.matches) {
        autoplay.stop()
      } else {
        autoplay.play()
      }
    }

    reducedMotion.addEventListener('change', syncReducedMotion)
    syncReducedMotion()

    return () => {
      reducedMotion.removeEventListener('change', syncReducedMotion)
    }
  }, [api, autoplay])

  return (
    <section className='border-border/40 bg-muted/10 relative z-10 border-b px-6 py-24 md:py-32'>
      <div className={PCC_AGENT_SECTION_CLASSES.container}>
        <div className={PCC_AGENT_SECTION_CLASSES.layout}>
          <AnimateInView className='max-w-xl'>
            <div className='mb-5 flex items-center gap-3'>
              <PccAgentLogo className='size-11 rounded-lg shadow-sm' />
              <div>
                <p className='text-muted-foreground text-xs font-medium tracking-widest uppercase'>
                  {t('Official Agent')}
                </p>
                <p className='text-lg font-bold'>PccAgent</p>
              </div>
            </div>

            <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl lg:text-4xl'>
              {t('One workspace from conversation to execution')}
            </h2>
            <p className='text-muted-foreground mt-5 text-sm leading-7 md:text-base'>
              {t(
                'Bring chats, terminals, Claude Code, Codex, and DPCC API account insights into one focused desktop workspace.'
              )}
            </p>

            <ul className='mt-7 space-y-3 text-sm'>
              {[
                t('Work with multiple Agents in one workspace'),
                t('Track balance, keys, and Token activity'),
                t('Connect DPCC API with secure browser authorization'),
              ].map((feature) => (
                <li key={feature} className='flex items-start gap-3'>
                  <span className='border-primary/25 bg-primary/8 text-primary mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border'>
                    <Check aria-hidden='true' className='size-3' />
                  </span>
                  <span>{feature}</span>
                </li>
              ))}
            </ul>

            <div className='mt-8 flex flex-col items-stretch gap-3 sm:flex-row sm:items-center'>
              <Button
                className='h-11 gap-2 px-5'
                render={
                  <a
                    href={PCC_AGENT_STORE_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                  />
                }
              >
                <img
                  src='/landing/microsoft-store.png'
                  alt=''
                  aria-hidden='true'
                  width={16}
                  height={16}
                  className='size-4'
                />
                {t('Download PccAgent from Microsoft Store')}
                <Download aria-hidden='true' className='size-4' />
              </Button>
              <span className='text-muted-foreground text-center text-xs sm:text-left'>
                Windows 10/11
              </span>
            </div>

            <div className='border-primary/20 mt-6 flex items-start gap-3 border-t pt-5'>
              <Gift aria-hidden='true' className='text-primary mt-0.5 size-5' />
              <div>
                <p className='text-sm font-semibold'>{t('Limited offer')}</p>
                <p className='text-muted-foreground mt-1 text-sm leading-6'>
                  {t('Use the official Agent to get one month of Lite free')}
                </p>
              </div>
            </div>
          </AnimateInView>

          <AnimateInView animation='scale-in' delay={120}>
            <Carousel
              setApi={setApi}
              opts={{ loop: true }}
              plugins={[autoplay]}
              aria-label={t('PccAgent product previews')}
              className='min-w-0'
            >
              <div className='border-border/50 overflow-hidden rounded-lg border shadow-[0_32px_90px_-40px_rgba(15,23,42,0.42)]'>
                <CarouselContent className='ml-0'>
                  {slides.map((slide, index) => (
                    <CarouselItem
                      key={slide.src}
                      className='pl-0'
                      aria-label={t('Slide {{current}} of {{total}}', {
                        current: index + 1,
                        total: slides.length,
                      })}
                    >
                      <div className={PCC_AGENT_SECTION_CLASSES.media}>
                        <img
                          src={slide.src}
                          alt={slide.alt}
                          width={PCC_AGENT_SLIDE_DIMENSIONS.width}
                          height={PCC_AGENT_SLIDE_DIMENSIONS.height}
                          loading='lazy'
                          decoding='async'
                          draggable={false}
                          className={PCC_AGENT_SECTION_CLASSES.image}
                        />
                      </div>
                    </CarouselItem>
                  ))}
                </CarouselContent>
              </div>

              <div className='mt-4 flex items-center justify-between gap-4'>
                <div className='flex items-center gap-2'>
                  {slides.map((slide, index) => (
                    <button
                      key={slide.src}
                      type='button'
                      onClick={() => {
                        api?.scrollTo(index)
                        autoplay.reset()
                      }}
                      aria-label={t('View slide {{number}}', {
                        number: index + 1,
                      })}
                      aria-current={
                        index === selectedIndex ? 'true' : undefined
                      }
                      className={cn(
                        'focus-visible:ring-ring size-2.5 rounded-full border transition-colors outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
                        index === selectedIndex
                          ? 'border-primary bg-primary'
                          : 'border-border bg-muted-foreground/20 hover:bg-muted-foreground/40'
                      )}
                    />
                  ))}
                  <span className='text-muted-foreground ml-1 text-xs tabular-nums'>
                    {selectedIndex + 1} / {slides.length}
                  </span>
                </div>

                <div className='flex items-center gap-2'>
                  <Button
                    type='button'
                    variant='outline'
                    size='icon'
                    onClick={() => {
                      api?.scrollPrev()
                      autoplay.reset()
                    }}
                    aria-label={t('Previous slide')}
                  >
                    <ArrowLeft aria-hidden='true' className='size-4' />
                  </Button>
                  <Button
                    type='button'
                    variant='outline'
                    size='icon'
                    onClick={() => {
                      api?.scrollNext()
                      autoplay.reset()
                    }}
                    aria-label={t('Next slide')}
                  >
                    <ArrowRight aria-hidden='true' className='size-4' />
                  </Button>
                </div>
              </div>
            </Carousel>
          </AnimateInView>
        </div>
      </div>
    </section>
  )
}
