/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ArrowRight, Check, X } from 'lucide-react'
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
} from 'react'
import { useTranslation } from 'react-i18next'

import { PccAgentLogo } from '@/features/desktop-authorization/pcc-agent-logo'
import { useStatus } from '@/hooks/use-status'
import { useAuthStore } from '@/stores/auth-store'

import type { AnnouncementNotification } from '../notifications/notification-feed'
import {
  getPccAgentTicketBrowserStorageKey,
  getPccAgentTicketStorageKey,
  markPccAgentTicketSeen,
  readPccAgentTicketSeen,
} from './pcc-agent-ticket-storage'

type TicketState = 'ready' | 'dragging' | 'claimed'

type DragState = {
  pointerId: number
  startX: number
}

type TicketStyle = CSSProperties & {
  '--pcc-ticket-drag-x': string
  '--pcc-ticket-drag-tilt': string
  '--pcc-ticket-drag-progress': string
}

type TicketLoadState = {
  key: string | null
  shouldLoad: boolean
}

type PccAgentTicketDialogProps = {
  onComplete: () => void
  onPresented: () => void
}

const DESKTOP_TEAR_THRESHOLD = 112
const MOBILE_TEAR_THRESHOLD = 76
const MAX_DRAG_OFFSET = 220

function getBrowserStorage(): Storage | undefined {
  if (typeof window === 'undefined') return undefined

  try {
    return window.localStorage
  } catch {
    return undefined
  }
}

function getTearThreshold(): number {
  return typeof window !== 'undefined' && window.innerWidth < 560
    ? MOBILE_TEAR_THRESHOLD
    : DESKTOP_TEAR_THRESHOLD
}

export function PccAgentTicket() {
  const userId = useAuthStore((state) => state.auth.user?.id)
  const bootstrapState = useAuthStore((state) => state.auth.bootstrapState)
  const storageKey = useMemo(
    () => getPccAgentTicketStorageKey(userId),
    [userId]
  )
  const browserStorageKey = getPccAgentTicketBrowserStorageKey()
  const visitorStorageKey = getPccAgentTicketStorageKey(undefined)
  const [loadState, setLoadState] = useState<TicketLoadState>({
    key: null,
    shouldLoad: false,
  })
  const evaluatedStorageKeyRef = useRef<string | null>(null)

  useEffect(() => {
    if (bootstrapState === 'checking') return
    if (evaluatedStorageKeyRef.current === storageKey) return

    evaluatedStorageKeyRef.current = storageKey

    const storage = getBrowserStorage()
    const shouldLoad =
      !readPccAgentTicketSeen(storage, browserStorageKey) &&
      !readPccAgentTicketSeen(storage, visitorStorageKey) &&
      !readPccAgentTicketSeen(storage, storageKey)
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoadState({ key: storageKey, shouldLoad })
  }, [bootstrapState, browserStorageKey, storageKey, visitorStorageKey])

  if (loadState.key !== storageKey || !loadState.shouldLoad) return null

  return (
    <PccAgentTicketDialog
      onComplete={() =>
        setLoadState((current) =>
          current.key === storageKey
            ? { ...current, shouldLoad: false }
            : current
        )
      }
      onPresented={() => {
        const storage = getBrowserStorage()
        markPccAgentTicketSeen(storage, browserStorageKey)
        markPccAgentTicketSeen(storage, visitorStorageKey)
        markPccAgentTicketSeen(storage, storageKey)
      }}
    />
  )
}

function PccAgentTicketDialog(props: PccAgentTicketDialogProps) {
  const { t } = useTranslation()
  const { onComplete, onPresented } = props
  const [ticketState, setTicketState] = useState<TicketState | null>(null)
  const [dragOffset, setDragOffset] = useState(0)
  const dragStateRef = useRef<DragState | null>(null)
  const { status, loading: statusLoading } = useStatus()
  const ticketNotification = useMemo(() => {
    if (statusLoading || status?.global_notifications_enabled === false) {
      return null
    }

    const globalNotifications = status?.global_notifications
    if (!Array.isArray(globalNotifications)) return null

    return (
      (globalNotifications as AnnouncementNotification[]).find((notification) =>
        Boolean(
          notification &&
          typeof notification === 'object' &&
          notification.kind === 'pcc-agent-ticket'
        )
      ) ?? null
    )
  }, [status, statusLoading])
  const ticketTitle =
    ticketNotification?.title?.trim() ||
    t('Claim {{amount}} in AI API credits', { amount: '$150' })
  const ticketSubline =
    ticketNotification?.content?.trim() ||
    t('Use PccAgent to unlock your credit')

  useEffect(() => {
    if (statusLoading) return
    if (!ticketNotification) {
      onComplete()
      return
    }
    // Mark before painting so StrictMode and rapid route transitions cannot duplicate it.
    onPresented()
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setTicketState('ready')
  }, [onComplete, onPresented, statusLoading, ticketNotification])

  if (!ticketState || !ticketNotification) return null

  const claimed = ticketState === 'claimed'
  const dragging = ticketState === 'dragging'
  const ticketStyle: TicketStyle = {
    '--pcc-ticket-drag-x': `${dragOffset}px`,
    '--pcc-ticket-drag-tilt': `${Math.min(dragOffset / 18, 8)}deg`,
    '--pcc-ticket-drag-progress': String(
      Math.min(dragOffset / getTearThreshold(), 1)
    ),
  }

  const completeTear = (releasedOffset = 0) => {
    dragStateRef.current = null
    setDragOffset(releasedOffset)
    setTicketState('claimed')
  }

  const resetDrag = (event?: PointerEvent<HTMLButtonElement>) => {
    if (event) {
      const dragState = dragStateRef.current
      if (!dragState || dragState.pointerId !== event.pointerId) return
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId)
      }
    }
    dragStateRef.current = null
    setDragOffset(0)
    setTicketState('ready')
  }

  const handlePointerDown = (event: PointerEvent<HTMLButtonElement>) => {
    if (claimed || (ticketState !== 'ready' && !dragging)) return
    if (event.button !== 0) return

    dragStateRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
    }
    event.currentTarget.setPointerCapture(event.pointerId)
    setTicketState('dragging')
  }

  const handlePointerMove = (event: PointerEvent<HTMLButtonElement>) => {
    const dragState = dragStateRef.current
    if (!dragState || dragState.pointerId !== event.pointerId) return

    const nextOffset = Math.min(
      MAX_DRAG_OFFSET,
      Math.max(0, event.clientX - dragState.startX)
    )
    setDragOffset(nextOffset)
  }

  const finishPointerDrag = (event: PointerEvent<HTMLButtonElement>) => {
    const dragState = dragStateRef.current
    if (!dragState || dragState.pointerId !== event.pointerId) return

    const finalOffset = Math.min(
      MAX_DRAG_OFFSET,
      Math.max(0, event.clientX - dragState.startX)
    )
    if (finalOffset >= getTearThreshold()) {
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId)
      }
      completeTear(finalOffset)
      return
    }

    resetDrag(event)
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (claimed) return
    if (event.key !== 'Enter' && event.key !== ' ') return

    event.preventDefault()
    completeTear()
  }

  const handleDismiss = () => {
    dragStateRef.current = null
    setDragOffset(0)
    onComplete()
  }

  return (
    <div
      className='pcc-agent-ticket-root'
      data-floating-action='pcc-agent-promo'
    >
      <article
        className={`pcc-agent-ticket${claimed ? ' is-claimed' : ''}${dragging ? ' is-dragging' : ''}`}
        style={ticketStyle}
        role='dialog'
        aria-modal='true'
        aria-label={
          claimed
            ? t('PccAgent credit claimed')
            : t('PccAgent $150 credit offer')
        }
        data-state={ticketState}
      >
        <div className='pcc-agent-ticket__body'>
          <div className='pcc-agent-ticket__topline'>
            <span className='pcc-agent-ticket__eyebrow'>
              {t('Limited offer')}
            </span>
            <span className='pcc-agent-ticket__serial'>PCC / 150</span>
          </div>

          <div className='pcc-agent-ticket__brand'>
            <span className='pcc-agent-ticket__brand-mark'>
              <PccAgentLogo className='pcc-agent-ticket__logo' />
            </span>
            <span className='pcc-agent-ticket__brand-name'>PccAgent</span>
          </div>

          <p className='pcc-agent-ticket__headline'>{ticketTitle}</p>

          <div className='pcc-agent-ticket__amount' aria-hidden='true'>
            <span className='pcc-agent-ticket__currency'>$</span>
            <strong>150</strong>
            <span className='pcc-agent-ticket__unit'>
              {t('AI API credits')}
            </span>
          </div>

          <p className='pcc-agent-ticket__subline'>{ticketSubline}</p>

          <p className='pcc-agent-ticket__notice-link'>
            {t('View in Global notifications')}
          </p>

          <div className='pcc-agent-ticket__footer'>
            <span>{t('One-time claim')}</span>
            <span>{t('Official PccAgent offer')}</span>
          </div>
        </div>

        <div className='pcc-agent-ticket__stub-shell'>
          <div
            className='pcc-agent-ticket__claim-panel'
            aria-hidden={!claimed}
            aria-live='polite'
          >
            <Check
              className='pcc-agent-ticket__claim-icon'
              aria-hidden='true'
            />
            <span className='pcc-agent-ticket__claim-label'>
              {t('Claimed successfully')}
            </span>
            <strong>$150</strong>
            <small>{t('PccAgent credit claim complete')}</small>
          </div>

          <button
            type='button'
            className='pcc-agent-ticket__stub'
            aria-label={t('Drag right to tear and claim')}
            aria-describedby='pcc-agent-ticket-guide'
            aria-keyshortcuts='Enter Space'
            aria-hidden={claimed}
            disabled={claimed}
            tabIndex={claimed ? -1 : 0}
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={finishPointerDrag}
            onPointerCancel={resetDrag}
            onKeyDown={handleKeyDown}
          >
            <span className='pcc-agent-ticket__stub-topline'>
              <span>PccAgent</span>
              <span>01 / 01</span>
            </span>
            <span className='pcc-agent-ticket__stub-title'>
              {t('Tear to claim')}
            </span>
            <span
              className='pcc-agent-ticket__stub-guide'
              id='pcc-agent-ticket-guide'
            >
              <ArrowRight aria-hidden='true' />
              <span>{t('Drag right to tear and claim')}</span>
            </span>
            <span className='pcc-agent-ticket__stub-amount'>$150</span>
          </button>
        </div>

        <span className='pcc-agent-ticket__seam' aria-hidden='true' />
        <span
          className='pcc-agent-ticket__notch pcc-agent-ticket__notch--top'
          aria-hidden='true'
        />
        <span
          className='pcc-agent-ticket__notch pcc-agent-ticket__notch--bottom'
          aria-hidden='true'
        />

        {claimed ? (
          <span className='pcc-agent-ticket__confetti' aria-hidden='true'>
            <i />
            <i />
            <i />
          </span>
        ) : null}

        <button
          type='button'
          className='pcc-agent-ticket__close'
          aria-label={t('Close PccAgent offer')}
          title={t('Close PccAgent offer')}
          onClick={handleDismiss}
        >
          <X aria-hidden='true' />
        </button>
      </article>
    </div>
  )
}
