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
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
} from 'react'

import {
  clampNotificationBottomOffset,
  floatingNotificationPosition,
  resolveNotificationBottomOffset,
  type FloatingVerticalSpan,
} from './floating-position'

const INTERACTIVE_SELECTOR =
  'button, a[href], input, select, textarea, iframe, [role="button"], [data-floating-action]'

type DragState = {
  pointerId: number
  startY: number
  startOffset: number
  moved: boolean
}

type NotificationPositionStyle = CSSProperties & {
  '--notification-bottom-offset': string
}

function hasFixedPosition(element: Element): boolean {
  let current: Element | null = element
  while (current && current !== document.documentElement) {
    const position = window.getComputedStyle(current).position
    if (position === 'fixed' || position === 'sticky') return true
    current = current.parentElement
  }
  return false
}

function collectFloatingBlockers(
  root: HTMLElement,
  rect: DOMRect,
  bottomOffset: number,
  viewportHeight: number
): FloatingVerticalSpan[] {
  const candidateBottom = viewportHeight - bottomOffset
  const candidateTop = candidateBottom - rect.height
  const xPositions = [0.08, 0.3, 0.5, 0.7, 0.92].map(
    (ratio) => rect.left + rect.width * ratio
  )
  const yPositions = [0.08, 0.3, 0.5, 0.7, 0.92].map(
    (ratio) => candidateTop + rect.height * ratio
  )
  const blockers = new Map<Element, FloatingVerticalSpan>()

  for (const x of xPositions) {
    for (const y of yPositions) {
      for (const element of document.elementsFromPoint(x, y)) {
        if (element === root || root.contains(element)) continue

        const interactive = element.closest(INTERACTIVE_SELECTOR)
        if (!interactive || root.contains(interactive)) continue
        if (!hasFixedPosition(interactive)) continue

        const style = window.getComputedStyle(interactive)
        if (
          style.pointerEvents === 'none' ||
          style.visibility === 'hidden' ||
          style.display === 'none'
        ) {
          continue
        }

        const blockerRect = interactive.getBoundingClientRect()
        if (
          blockerRect.width <= 0 ||
          blockerRect.height <= 0 ||
          blockerRect.right <= rect.left ||
          blockerRect.left >= rect.right
        ) {
          continue
        }

        blockers.set(interactive, {
          top: blockerRect.top,
          bottom: blockerRect.bottom,
        })
      }
    }
  }

  return [...blockers.values()]
}

function readStoredOffset(): number | null {
  try {
    const storedValue = window.localStorage.getItem(
      floatingNotificationPosition.storageKey
    )
    if (!storedValue) return null

    const offset = Number(storedValue)
    return Number.isFinite(offset) ? offset : null
  } catch {
    return null
  }
}

function storeOffset(offset: number) {
  try {
    window.localStorage.setItem(
      floatingNotificationPosition.storageKey,
      String(Math.round(offset))
    )
  } catch {
    // Storage may be unavailable in private or restricted browser contexts.
  }
}

export function useFloatingNotificationPosition(expanded: boolean) {
  const rootRef = useRef<HTMLElement>(null)
  const dragStateRef = useRef<DragState | null>(null)
  const suppressClickRef = useRef(false)
  const animationFrameRef = useRef<number | null>(null)
  const [preferredOffset, setPreferredOffset] = useState<number>(
    floatingNotificationPosition.viewportMargin
  )
  const [resolvedOffset, setResolvedOffset] = useState<number>(
    floatingNotificationPosition.viewportMargin
  )
  const [dragging, setDragging] = useState(false)

  const resolvePosition = useCallback(() => {
    const root = rootRef.current
    if (!root || dragging) return

    const rect = root.getBoundingClientRect()
    const safeAreaBottom = Math.max(
      0,
      window.innerHeight - rect.bottom - resolvedOffset
    )
    const usableViewportHeight = window.innerHeight - safeAreaBottom
    const desiredOffset = expanded ? resolvedOffset : preferredOffset
    let nextOffset = clampNotificationBottomOffset(
      desiredOffset,
      rect.height,
      usableViewportHeight
    )

    if (!expanded) {
      for (
        let pass = 0;
        pass < floatingNotificationPosition.maxCollisionPasses;
        pass += 1
      ) {
        const blockers = collectFloatingBlockers(
          root,
          rect,
          nextOffset,
          usableViewportHeight
        )
        const availableOffset = resolveNotificationBottomOffset({
          preferredOffset: nextOffset,
          widgetHeight: rect.height,
          viewportHeight: usableViewportHeight,
          blockers,
        })
        if (availableOffset === nextOffset) break
        nextOffset = availableOffset
      }
    }

    setResolvedOffset((currentOffset) =>
      currentOffset === nextOffset ? currentOffset : nextOffset
    )
  }, [dragging, expanded, preferredOffset, resolvedOffset])

  const schedulePositionResolution = useCallback(() => {
    if (animationFrameRef.current !== null) {
      window.cancelAnimationFrame(animationFrameRef.current)
    }
    animationFrameRef.current = window.requestAnimationFrame(() => {
      animationFrameRef.current = null
      resolvePosition()
    })
  }, [resolvePosition])

  useEffect(() => {
    const storedOffset = readStoredOffset()
    if (storedOffset === null) return
    setPreferredOffset(storedOffset)
  }, [])

  useLayoutEffect(() => {
    resolvePosition()
  }, [resolvePosition])

  useEffect(() => {
    const root = rootRef.current
    if (!root) return

    const resizeObserver = new ResizeObserver(schedulePositionResolution)
    resizeObserver.observe(root)
    const mutationObserver = new MutationObserver(schedulePositionResolution)
    mutationObserver.observe(document.body, {
      childList: true,
      subtree: true,
    })
    window.addEventListener('resize', schedulePositionResolution)

    return () => {
      resizeObserver.disconnect()
      mutationObserver.disconnect()
      window.removeEventListener('resize', schedulePositionResolution)
      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current)
        animationFrameRef.current = null
      }
    }
  }, [schedulePositionResolution])

  const getClampedOffset = useCallback((offset: number) => {
    const widgetHeight = rootRef.current?.getBoundingClientRect().height ?? 64
    return clampNotificationBottomOffset(
      offset,
      widgetHeight,
      window.innerHeight
    )
  }, [])

  const handlePointerDown = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      if (expanded || event.button !== 0) return

      dragStateRef.current = {
        pointerId: event.pointerId,
        startY: event.clientY,
        startOffset: resolvedOffset,
        moved: false,
      }
      event.currentTarget.setPointerCapture(event.pointerId)
      setDragging(true)
    },
    [expanded, resolvedOffset]
  )

  const handlePointerMove = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      const dragState = dragStateRef.current
      if (!dragState || dragState.pointerId !== event.pointerId) return

      const delta = dragState.startY - event.clientY
      if (Math.abs(delta) >= 4) dragState.moved = true
      const nextOffset = getClampedOffset(dragState.startOffset + delta)
      setPreferredOffset(nextOffset)
      setResolvedOffset(nextOffset)
    },
    [getClampedOffset]
  )

  const finishDragging = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      const dragState = dragStateRef.current
      if (!dragState || dragState.pointerId !== event.pointerId) return

      const finalOffset = getClampedOffset(
        dragState.startOffset + dragState.startY - event.clientY
      )
      suppressClickRef.current = dragState.moved
      dragStateRef.current = null
      setDragging(false)
      setPreferredOffset(finalOffset)
      setResolvedOffset(finalOffset)
      storeOffset(finalOffset)
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId)
      }
      schedulePositionResolution()
    },
    [getClampedOffset, schedulePositionResolution]
  )

  const cancelDragging = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      suppressClickRef.current = false
      finishDragging(event)
      suppressClickRef.current = false
    },
    [finishDragging]
  )

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLButtonElement>) => {
      if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return

      event.preventDefault()
      const direction = event.key === 'ArrowUp' ? 1 : -1
      const nextOffset = getClampedOffset(
        resolvedOffset + direction * floatingNotificationPosition.keyboardStep
      )
      setPreferredOffset(nextOffset)
      setResolvedOffset(nextOffset)
      storeOffset(nextOffset)
    },
    [getClampedOffset, resolvedOffset]
  )

  const shouldOpenFromTrigger = useCallback(() => {
    if (!suppressClickRef.current) return true
    suppressClickRef.current = false
    return false
  }, [])

  const style: NotificationPositionStyle = {
    '--notification-bottom-offset': `${resolvedOffset}px`,
  }

  return {
    rootRef,
    style,
    dragging,
    handlePointerDown,
    handlePointerMove,
    finishDragging,
    cancelDragging,
    handleKeyDown,
    shouldOpenFromTrigger,
  }
}
