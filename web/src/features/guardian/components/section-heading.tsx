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
import type { ReactNode } from 'react'

import { AnimateInView } from '@/components/animate-in-view'
import { cn } from '@/lib/utils'

export function SectionHeading(props: {
  id: string
  eyebrow: string
  title: string
  lede?: ReactNode
  className?: string
}) {
  return (
    <AnimateInView className={cn('max-w-2xl', props.className)}>
      <p className='g-eyebrow'>{props.eyebrow}</p>
      <h2
        id={`${props.id}-title`}
        className='g-heading mt-4 text-[2rem] sm:text-[2.5rem]'
      >
        {props.title}
      </h2>
      {props.lede && <p className='g-lede mt-5'>{props.lede}</p>}
    </AnimateInView>
  )
}

export function GuardianSection(props: {
  id: string
  children: ReactNode
  tone?: 'paper' | 'sunk'
  className?: string
}) {
  return (
    <section
      id={props.id}
      aria-labelledby={`${props.id}-title`}
      className={cn('scroll-mt-20', props.tone === 'sunk' && 'g-band')}
    >
      <div
        className={cn(
          'mx-auto max-w-6xl 2xl:max-w-7xl px-4 py-20 sm:px-6 lg:py-28',
          props.className
        )}
      >
        {props.children}
      </div>
    </section>
  )
}
