/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { compressImageToWebP } from '../compress-thumbnail'

describe('compressImageToWebP', () => {
  let canvas: HTMLCanvasElement
  let drawImage: ReturnType<typeof vi.fn>

  beforeEach(() => {
    const originalCreateElement = document.createElement.bind(document)
    drawImage = vi.fn()
    canvas = {
      width: 0,
      height: 0,
      getContext: () => ({ drawImage }),
      toBlob: (callback: BlobCallback) =>
        callback(new Blob(['webp'], { type: 'image/webp' })),
    } as unknown as HTMLCanvasElement

    class MockImage {
      naturalWidth = 1000
      naturalHeight = 800
      private listeners = new Map<string, () => void>()

      addEventListener(type: string, listener: () => void) {
        this.listeners.set(type, listener)
      }

      set src(_value: string) {
        this.listeners.get('load')?.()
      }
    }

    vi.stubGlobal('createImageBitmap', undefined)
    vi.stubGlobal('Image', MockImage)
    vi.stubGlobal(
      'FileReader',
      class MockFileReader {
        result = 'data:image/webp;base64,test'
        private listeners = new Map<string, () => void>()

        addEventListener(type: string, listener: () => void) {
          this.listeners.set(type, listener)
        }

        readAsDataURL() {
          this.listeners.get('load')?.()
        }
      }
    )
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:test')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    vi.spyOn(document, 'createElement').mockImplementation((tagName) => {
      if (tagName === 'canvas') return canvas
      return originalCreateElement(tagName)
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  test('uses an HTML image fallback and preserves an arbitrary crop ratio', async () => {
    const file = new File(['image'], 'source.png', { type: 'image/png' })
    const crop = {
      unit: 'px' as const,
      x: 100,
      y: 50,
      width: 300,
      height: 200,
    }

    const result = await compressImageToWebP(file, undefined, crop)

    expect(result).toBe('data:image/webp;base64,test')
    expect(canvas.width).toBe(300)
    expect(canvas.height).toBe(200)
    expect(drawImage).toHaveBeenCalledWith(
      expect.anything(),
      100,
      50,
      300,
      200,
      0,
      0,
      300,
      200
    )
    expect(URL.createObjectURL).toHaveBeenCalledWith(file)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:test')
  })
})
