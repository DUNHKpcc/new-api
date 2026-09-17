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

type LoadedImage = {
  source: CanvasImageSource
  width: number
  height: number
  dispose: () => void
}

type CropArea = {
  unit?: 'px'
  x: number
  y: number
  width: number
  height: number
}

function loadHtmlImage(file: File): Promise<LoadedImage> {
  const url = URL.createObjectURL(file)
  const image = new Image()

  return new Promise((resolve, reject) => {
    const handleLoad = () => {
      resolve({
        source: image,
        width: image.naturalWidth,
        height: image.naturalHeight,
        dispose: () => URL.revokeObjectURL(url),
      })
    }
    const handleError = () => {
      URL.revokeObjectURL(url)
      reject(new Error('Failed to load thumbnail'))
    }
    image.addEventListener('load', handleLoad, { once: true })
    image.addEventListener('error', handleError, { once: true })
    image.src = url
  })
}

async function loadImage(file: File): Promise<LoadedImage> {
  if (typeof createImageBitmap === 'function') {
    try {
      const bitmap = await createImageBitmap(file)
      return {
        source: bitmap,
        width: bitmap.width,
        height: bitmap.height,
        dispose: () => bitmap.close(),
      }
    } catch {
      // Fall back to an HTML image for browsers with partial bitmap support.
    }
  }

  return loadHtmlImage(file)
}

export async function compressImageToWebP(
  file: File,
  aspectRatio?: number,
  crop?: CropArea
): Promise<string> {
  if (
    aspectRatio !== undefined &&
    (!Number.isFinite(aspectRatio) || aspectRatio <= 0)
  ) {
    throw new Error('Invalid crop')
  }
  if (
    !['image/jpeg', 'image/png', 'image/webp'].includes(file.type) ||
    file.size === 0 ||
    file.size > MAX_SOURCE_BYTES
  ) {
    throw new Error('Invalid thumbnail file')
  }

  const source = await loadImage(file)
  try {
    if (
      crop &&
      (![crop.x, crop.y, crop.width, crop.height].every(Number.isFinite) ||
        (crop.unit !== undefined && crop.unit !== 'px') ||
        crop.x < 0 ||
        crop.y < 0 ||
        crop.width <= 0 ||
        crop.height <= 0 ||
        crop.x + crop.width > source.width + 1 ||
        crop.y + crop.height > source.height + 1)
    ) {
      throw new Error('Invalid crop')
    }

    let cropWidth = source.width
    let cropHeight = source.height
    let cropX = 0
    let cropY = 0
    if (crop) {
      cropWidth = crop.width
      cropHeight = crop.height
      cropX = crop.x
      cropY = crop.y
    } else if (aspectRatio) {
      const sourceRatio = source.width / source.height
      if (sourceRatio > aspectRatio) {
        cropWidth = source.height * aspectRatio
        cropX = (source.width - cropWidth) / 2
      } else if (sourceRatio < aspectRatio) {
        cropHeight = source.width / aspectRatio
        cropY = (source.height - cropHeight) / 2
      }
    }

    for (const attempt of compressionAttempts) {
      const scale = Math.min(
        1,
        attempt.maxWidth / cropWidth,
        attempt.maxHeight / cropHeight
      )
      const width = Math.max(1, Math.round(cropWidth * scale))
      const height = Math.max(1, Math.round(cropHeight * scale))
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height

      const context = canvas.getContext('2d')
      if (!context) throw new Error('Failed to prepare thumbnail')
      context.drawImage(
        source.source,
        cropX,
        cropY,
        cropWidth,
        cropHeight,
        0,
        0,
        width,
        height
      )

      const blob = await canvasToWebP(canvas, attempt.quality)
      if (blob.size <= MAX_THUMBNAIL_BYTES) {
        return blobToDataUrl(blob)
      }
    }
  } finally {
    source.dispose()
  }

  throw new Error('Thumbnail is too large')
}

export const compressResourceThumbnail = compressImageToWebP
