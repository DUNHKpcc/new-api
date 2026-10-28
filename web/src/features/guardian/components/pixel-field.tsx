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
import { useEffect, useRef } from 'react'

import {
  cellAlpha,
  isRippleAlive,
  parseHexColor,
  rippleRing,
  spotlight,
  valueNoise3D,
  type PixelRipple,
} from '../lib/pixel-field'

const FRAME_MS = 1000 / 30
const MAX_RIPPLES = 4
const NOISE_SCALE = 0.09
const NOISE_SPEED = 0.0006
const FALLBACK_COLOR = '#b4552f'

/**
 * Background matrix of square pixels, after the promo video's PixelField.
 * The bright patch trails the cursor, clicks send out a ripple, and the
 * pattern scrolls with the page. Reduced motion draws one static frame.
 */
export function PixelField() {
  const ref = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = ref.current
    const ctx = canvas?.getContext('2d')
    if (!canvas || !ctx) return

    const reduceMotion = window.matchMedia(
      '(prefers-reduced-motion: reduce)'
    ).matches
    const readColor = () =>
      parseHexColor(getComputedStyle(canvas).getPropertyValue('--g-accent')) ??
      FALLBACK_COLOR
    let color = readColor()
    let width = 0
    let height = 0
    let cell = 28
    const pointer = { x: 0, y: 0, tx: 0, ty: 0, strength: 0, target: 0 }
    const ripples: PixelRipple[] = []

    const resize = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2)
      width = window.innerWidth
      height = window.innerHeight
      cell = width < 640 ? 22 : 28
      canvas.width = Math.round(width * dpr)
      canvas.height = Math.round(height * dpr)
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    }

    const draw = (now: number) => {
      pointer.x += (pointer.tx - pointer.x) * 0.14
      pointer.y += (pointer.ty - pointer.y) * 0.14
      pointer.strength += (pointer.target - pointer.strength) * 0.08
      while (ripples.length && !isRippleAlive(ripples[0], now)) ripples.shift()

      const scrollY = window.scrollY
      const rowShift = Math.floor(scrollY / cell)
      const yOffset = scrollY - rowShift * cell
      // A slight drift toward the cursor so the whole matrix leans with it.
      const driftX = (pointer.x - width / 2) * 0.0008 * pointer.strength
      const driftY = (pointer.y - height / 2) * 0.0008 * pointer.strength
      const cols = Math.ceil(width / cell) + 1
      const rows = Math.ceil(height / cell) + 1
      const time = now * NOISE_SPEED

      ctx.clearRect(0, 0, width, height)
      ctx.fillStyle = color
      for (let row = 0; row < rows; row++) {
        const py = row * cell - yOffset + cell / 2
        for (let col = 0; col < cols; col++) {
          const px = col * cell + cell / 2
          const noise = valueNoise3D(
            col * NOISE_SCALE + driftX,
            (row + rowShift) * NOISE_SCALE + driftY,
            time
          )
          const spot =
            pointer.strength > 0.01
              ? spotlight(Math.hypot(px - pointer.x, py - pointer.y)) *
                pointer.strength
              : 0
          let ring = 0
          for (const ripple of ripples) {
            ring += rippleRing(
              Math.hypot(px - ripple.x, py - ripple.y),
              now - ripple.start
            )
          }
          const alpha = cellAlpha(noise, spot, ring)
          if (alpha < 0.02) continue
          const size = cell * (0.56 + 0.2 * spot)
          ctx.globalAlpha = alpha
          ctx.fillRect(px - size / 2, py - size / 2, size, size)
        }
      }
      ctx.globalAlpha = 1
    }

    resize()
    const themeObserver = new MutationObserver(() => {
      color = readColor()
      if (reduceMotion) draw(0)
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme-preset'],
    })

    if (reduceMotion) {
      const redraw = () => {
        resize()
        draw(0)
      }
      redraw()
      window.addEventListener('resize', redraw)
      window.addEventListener('scroll', redraw, { passive: true })
      return () => {
        themeObserver.disconnect()
        window.removeEventListener('resize', redraw)
        window.removeEventListener('scroll', redraw)
      }
    }

    const onMove = (event: PointerEvent) => {
      pointer.tx = event.clientX
      pointer.ty = event.clientY
      if (pointer.target === 0) {
        pointer.x = event.clientX
        pointer.y = event.clientY
      }
      // Touch has no hover, so only a fine pointer lights the trailing patch.
      pointer.target = event.pointerType === 'mouse' ? 1 : 0
    }
    const onDown = (event: PointerEvent) => {
      ripples.push({
        x: event.clientX,
        y: event.clientY,
        start: performance.now(),
      })
      if (ripples.length > MAX_RIPPLES) ripples.shift()
    }
    const onLeave = () => {
      pointer.target = 0
    }

    let frame = 0
    let last = 0
    const tick = (now: number) => {
      frame = requestAnimationFrame(tick)
      if (now - last < FRAME_MS) return
      last = now
      draw(now)
    }
    frame = requestAnimationFrame(tick)

    window.addEventListener('resize', resize)
    window.addEventListener('pointermove', onMove, { passive: true })
    window.addEventListener('pointerdown', onDown, { passive: true })
    document.documentElement.addEventListener('pointerleave', onLeave)
    return () => {
      cancelAnimationFrame(frame)
      themeObserver.disconnect()
      window.removeEventListener('resize', resize)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerdown', onDown)
      document.documentElement.removeEventListener('pointerleave', onLeave)
    }
  }, [])

  return (
    <canvas
      ref={ref}
      aria-hidden='true'
      className='pointer-events-none fixed inset-0 z-0 size-full opacity-60 dark:opacity-75'
    />
  )
}
