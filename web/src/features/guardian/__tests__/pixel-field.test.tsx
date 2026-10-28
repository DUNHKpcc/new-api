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
import { cleanup, render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { PixelField } from '../components/pixel-field'
import {
  cellAlpha,
  rippleRing,
  spotlight,
  SPOTLIGHT_RADIUS,
  valueNoise3D,
} from '../lib/pixel-field'

function stubCanvasContext() {
  const ctx = {
    clearRect: vi.fn(),
    fillRect: vi.fn(),
    setTransform: vi.fn(),
    fillStyle: '',
    globalAlpha: 1,
  }
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    ctx as unknown as CanvasRenderingContext2D
  )
  return ctx
}

function preferReducedMotion(reduce: boolean) {
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (query: string) =>
      ({
        matches: reduce && query.includes('prefers-reduced-motion'),
        media: query,
        onchange: null,
        addListener: () => undefined,
        removeListener: () => undefined,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        dispatchEvent: () => false,
      }) as MediaQueryList
  )
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('pixel field math', () => {
  it('returns the same noise for the same coordinates and stays within [-1, 1]', () => {
    for (let i = 0; i < 200; i++) {
      const x = i * 0.37
      const y = i * 0.11
      const z = i * 0.05
      const value = valueNoise3D(x, y, z)
      expect(value).toBe(valueNoise3D(x, y, z))
      expect(value).toBeGreaterThanOrEqual(-1)
      expect(value).toBeLessThanOrEqual(1)
    }
  })

  it('lights cells under the cursor and leaves cells beyond the radius untouched', () => {
    expect(spotlight(0)).toBe(1)
    expect(spotlight(SPOTLIGHT_RADIUS / 2)).toBeCloseTo(0.25)
    expect(spotlight(SPOTLIGHT_RADIUS)).toBe(0)
    expect(cellAlpha(-1, spotlight(0), 0)).toBeGreaterThan(cellAlpha(-1, 0, 0))
  })

  it('moves the ripple ring outward over time and fades it out', () => {
    expect(rippleRing(0, 0)).toBeCloseTo(1)
    expect(rippleRing(220, 200)).toBeCloseTo(rippleRing(0, 0) * 0.84, 1)
    expect(rippleRing(0, 400)).toBeLessThan(0.01)
    expect(rippleRing(2000, 5000)).toBe(0)
    expect(rippleRing(0, -10)).toBe(0)
  })

  it('never exceeds full opacity however strong the inputs', () => {
    expect(cellAlpha(1, 1, 3)).toBe(1)
  })
})

describe('PixelField', () => {
  it('draws one static frame without an animation loop when reduced motion is preferred', () => {
    preferReducedMotion(true)
    const ctx = stubCanvasContext()
    const raf = vi.spyOn(window, 'requestAnimationFrame')

    const { container } = render(<PixelField />)

    expect(container.querySelector('canvas')).toHaveAttribute(
      'aria-hidden',
      'true'
    )
    expect(ctx.clearRect).toHaveBeenCalled()
    expect(raf).not.toHaveBeenCalled()
  })

  it('animates when motion is allowed and stops the loop and listeners on unmount', () => {
    preferReducedMotion(false)
    stubCanvasContext()
    const raf = vi
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation(() => 42)
    const cancel = vi.spyOn(window, 'cancelAnimationFrame')
    const removeListener = vi.spyOn(window, 'removeEventListener')

    const { unmount } = render(<PixelField />)
    expect(raf).toHaveBeenCalledTimes(1)

    unmount()
    expect(cancel).toHaveBeenCalledWith(42)
    expect(removeListener).toHaveBeenCalledWith(
      'pointermove',
      expect.any(Function)
    )
    expect(removeListener).toHaveBeenCalledWith(
      'pointerdown',
      expect.any(Function)
    )
  })
})
