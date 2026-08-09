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
const MAX_SOURCE_BYTES = 8 * 1024 * 1024
const MAX_THUMBNAIL_BYTES = 300 * 1024

const compressionAttempts = [
  { maxWidth: 960, maxHeight: 540, quality: 0.8 },
  { maxWidth: 720, maxHeight: 405, quality: 0.7 },
  { maxWidth: 560, maxHeight: 315, quality: 0.62 },
] as const

function canvasToWebP(canvas: HTMLCanvasElement, quality: number) {
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob || blob.type !== 'image/webp') {
          reject(new Error('WebP encoding is not supported'))
          return
        }
        resolve(blob)
      },
      'image/webp',
      quality
    )
  })
}

function blobToDataUrl(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('error', () =>
      reject(new Error('Failed to read thumbnail'))
    )
    reader.addEventListener('load', () => resolve(String(reader.result)))
    reader.readAsDataURL(blob)
  })
}

export async function compressImageToWebP(file: File): Promise<string> {
  if (
    !['image/jpeg', 'image/png', 'image/webp'].includes(file.type) ||
    file.size === 0 ||
    file.size > MAX_SOURCE_BYTES
  ) {
    throw new Error('Invalid thumbnail file')
  }

  const source = await createImageBitmap(file)
  try {
    for (const attempt of compressionAttempts) {
      const scale = Math.min(
        1,
        attempt.maxWidth / source.width,
        attempt.maxHeight / source.height
      )
      const width = Math.max(1, Math.round(source.width * scale))
      const height = Math.max(1, Math.round(source.height * scale))
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height

      const context = canvas.getContext('2d')
      if (!context) throw new Error('Failed to prepare thumbnail')
      context.drawImage(source, 0, 0, width, height)

      const blob = await canvasToWebP(canvas, attempt.quality)
      if (blob.size <= MAX_THUMBNAIL_BYTES) {
        return blobToDataUrl(blob)
      }
    }
  } finally {
    source.close()
  }

  throw new Error('Thumbnail is too large')
}

export const compressResourceThumbnail = compressImageToWebP
