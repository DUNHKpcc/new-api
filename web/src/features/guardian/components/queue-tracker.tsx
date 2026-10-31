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
import { useEffect, useState, type CSSProperties } from 'react'
import { useTranslation } from 'react-i18next'

import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import type { GuardianOrder, GuardianQueueState } from '../types'

/** Squares drawn ahead of the buyer; longer queues collapse into "+N". */
const MAX_CELLS = 12
const ADVANCE_MS = 600

function formatQueueDay(value: string, locale: string | undefined): string {
  const [year, month, day] = value.split('-').map(Number)
  return new Intl.DateTimeFormat(locale, {
    month: 'long',
    day: 'numeric',
  }).format(new Date(year, month - 1, day))
}

export function QueueTracker(props: {
  order: GuardianOrder
  queue?: GuardianQueueState
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const inProgress = props.order.status === 'in_progress'
  const ahead = props.order.queue_ahead ?? 0
  const paused = !inProgress && (props.queue?.paused ?? false)
  // When orders ahead finish, keep their squares for one dissolve animation.
  const [advance, setAdvance] = useState({ ahead, leaving: 0 })
  if (advance.ahead !== ahead) {
    setAdvance({
      ahead,
      leaving: Math.min(Math.max(advance.ahead - ahead, 0), MAX_CELLS),
    })
  }
  useEffect(() => {
    if (advance.leaving === 0) return
    const timer = window.setTimeout(
      () => setAdvance((value) => ({ ...value, leaving: 0 })),
      ADVANCE_MS
    )
    return () => window.clearTimeout(timer)
  }, [advance])

  if (!inProgress && props.order.queue_ahead == null) return null

  const visible = inProgress ? 0 : Math.min(ahead, MAX_CELLS)
  const overflow = inProgress ? 0 : ahead - visible
  let headline = t('ahead of you')
  if (inProgress) headline = t('Your setup is in progress')
  else if (ahead === 0) headline = t("You're next")
  let detail = ''
  if (paused) {
    detail = props.queue?.resume_at
      ? t('Scheduling is paused. Expected to resume {{date}}', {
          date: new Intl.DateTimeFormat(locale, {
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          }).format(new Date(props.queue.resume_at * 1000)),
        })
      : t('Scheduling is paused for now')
  } else if (!inProgress && props.order.queue_start_date) {
    detail = t('Estimated start {{date}}', {
      date: formatQueueDay(props.order.queue_start_date, locale),
    })
  }

  return (
    <section
      aria-label={t('Delivery queue')}
      className='mt-5 rounded-xl bg-[var(--g-paper-sunk)] p-4'
    >
      <p role='status' className='flex items-baseline gap-2'>
        {!inProgress && ahead > 0 && (
          <span key={ahead} className='g-heading g-queue-count text-3xl'>
            {formatNumber(ahead, locale)}
          </span>
        )}
        <span className='text-sm font-medium'>{headline}</span>
      </p>
      <div
        aria-hidden='true'
        className='g-queue-row mt-3 overflow-hidden'
        data-paused={paused || undefined}
      >
        {overflow > 0 && (
          <span className='g-mono text-xs text-[var(--g-ink-soft)]'>
            +{formatNumber(overflow, locale)}
          </span>
        )}
        {Array.from({ length: advance.leaving }, (_, index) => (
          <span
            key={`leaving-${index}`}
            className='g-queue-cell'
            data-leaving
          />
        ))}
        {Array.from({ length: visible }, (_, index) => (
          <span
            key={`ahead-${visible - index}`}
            className='g-queue-cell'
            style={{ '--g-queue-index': visible - index } as CSSProperties}
          />
        ))}
        <span
          className='g-queue-cell'
          data-own
          data-active={inProgress || undefined}
        />
      </div>
      {detail && (
        <p className='mt-3 text-sm text-[var(--g-ink-soft)]'>{detail}</p>
      )}
    </section>
  )
}
