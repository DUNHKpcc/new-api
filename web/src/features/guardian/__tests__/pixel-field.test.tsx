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
