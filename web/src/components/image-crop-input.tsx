import { Check, RotateCcw, X } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import 'react-image-crop/dist/ReactCrop.css'
import ReactCrop, {
  centerCrop,
  makeAspectCrop,
  type PercentCrop,
  type PixelCrop,
} from 'react-image-crop'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { compressImageToWebP } from '@/features/resource-downloads/lib/compress-thumbnail'
import { cn } from '@/lib/utils'

type ImageCropInputProps = {
  id?: string
  value?: string
  aspectRatio?: number
  disabled?: boolean
  label: string
  onChange: (value: string) => void
}

type ImageSize = {
  width: number
  height: number
}

const DEFAULT_CROP: PercentCrop = {
  unit: '%',
  x: 5,
  y: 5,
  width: 90,
  height: 90,
}

function percentCropToPixels(crop: PercentCrop, size: ImageSize): PixelCrop {
  return {
    unit: 'px',
    x: Math.round((crop.x / 100) * size.width),
    y: Math.round((crop.y / 100) * size.height),
    width: Math.round((crop.width / 100) * size.width),
    height: Math.round((crop.height / 100) * size.height),
  }
}

function getInitialCrop(size: ImageSize, aspectRatio?: number): PercentCrop {
  if (!aspectRatio) return DEFAULT_CROP

  return centerCrop(
    makeAspectCrop(
      { unit: '%', width: 90 },
      aspectRatio,
      size.width,
      size.height
    ),
    size.width,
    size.height
  )
}

export function ImageCropInput(props: ImageCropInputProps) {
  const { t } = useTranslation()
  const inputId = useId()
  const generation = useRef(0)
  const [source, setSource] = useState<{ file: File; url: string } | null>(null)
  const [crop, setCrop] = useState<PercentCrop>(DEFAULT_CROP)
  const [area, setArea] = useState<PercentCrop | null>(null)
  const [sourceSize, setSourceSize] = useState<ImageSize | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(false)

  useEffect(
    () => () => {
      generation.current += 1
    },
    []
  )
  useEffect(
    () => () => {
      if (source) URL.revokeObjectURL(source.url)
    },
    [source]
  )

  return (
    <div className='min-w-0 space-y-2'>
      {source ? (
        <div className='relative max-h-64 w-full overflow-auto rounded-md bg-black p-2 sm:max-h-80'>
          <ReactCrop
            crop={crop}
            aspect={props.aspectRatio}
            disabled={busy || props.disabled}
            keepSelection
            minWidth={1}
            minHeight={1}
            onChange={(_, percentageCrop) => {
              setCrop(percentageCrop)
              setArea(percentageCrop)
            }}
            onComplete={(_, percentageCrop) => setArea(percentageCrop)}
          >
            <img
              src={source.url}
              alt={props.label}
              className='block max-h-60 max-w-full object-contain sm:max-h-72'
              onLoad={(event) => {
                const image = event.currentTarget
                setSourceSize({
                  width: image.naturalWidth,
                  height: image.naturalHeight,
                })
                const initialCrop = getInitialCrop(
                  {
                    width: image.naturalWidth,
                    height: image.naturalHeight,
                  },
                  props.aspectRatio
                )
                setCrop(initialCrop)
                setArea(initialCrop)
                setError(false)
              }}
              onError={() => {
                setError(true)
                setArea(null)
                setSourceSize(null)
              }}
            />
          </ReactCrop>
        </div>
      ) : (
        <div
          className={cn(
            'bg-muted overflow-hidden rounded-md border',
            props.aspectRatio ? undefined : 'min-h-32'
          )}
          style={
            props.aspectRatio ? { aspectRatio: props.aspectRatio } : undefined
          }
        >
          {props.value ? (
            <img
              src={props.value}
              alt={props.label}
              className={
                props.aspectRatio
                  ? 'size-full object-cover'
                  : 'block h-auto max-h-64 w-full object-contain'
              }
            />
          ) : null}
        </div>
      )}
      {source ? (
        <div className='flex flex-wrap gap-2'>
          <Button
            type='button'
            size='icon-sm'
            variant='outline'
            title={t('Reset')}
            aria-label={t('Reset')}
            disabled={busy || props.disabled}
            onClick={() => {
              const initialCrop = sourceSize
                ? getInitialCrop(sourceSize, props.aspectRatio)
                : DEFAULT_CROP
              setCrop(initialCrop)
              setArea(initialCrop)
            }}
          >
            <RotateCcw />
          </Button>
          <Button
            type='button'
            variant='outline'
            disabled={busy || props.disabled}
            onClick={() => {
              generation.current += 1
              setSource(null)
              setArea(null)
              setSourceSize(null)
              setError(false)
            }}
          >
            <X />
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            disabled={busy || props.disabled || !area || !sourceSize || error}
            onClick={async () => {
              if (!area || !sourceSize || busy) return
              const request = ++generation.current
              const pixelCrop = percentCropToPixels(area, sourceSize)
              setBusy(true)
              setError(false)
              try {
                const image = await compressImageToWebP(
                  source.file,
                  props.aspectRatio,
                  pixelCrop
                )
                if (request !== generation.current) return
                props.onChange(image)
                setSource(null)
                setSourceSize(null)
                setArea(null)
              } catch {
                if (request === generation.current) setError(true)
              } finally {
                if (request === generation.current) setBusy(false)
              }
            }}
          >
            <Check />
            {t('Apply')}
          </Button>
        </div>
      ) : null}
      <Input
        id={props.id ?? inputId}
        type='file'
        accept='image/jpeg,image/png,image/webp'
        disabled={busy || props.disabled}
        aria-label={props.label}
        onChange={(event) => {
          const file = event.currentTarget.files?.[0]
          event.currentTarget.value = ''
          if (!file) return
          setError(false)
          if (
            !['image/jpeg', 'image/png', 'image/webp'].includes(file.type) ||
            !file.size ||
            file.size > 8 * 1024 * 1024
          ) {
            setError(true)
            return
          }
          generation.current += 1
          setCrop(DEFAULT_CROP)
          setArea(null)
          setSourceSize(null)
          setSource({ file, url: URL.createObjectURL(file) })
        }}
      />
      {error ? (
        <p role='alert' className='text-destructive text-sm'>
          {t(
            'Image processing failed. Use a valid JPEG, PNG or WebP image up to 8 MB.'
          )}
        </p>
      ) : null}
    </div>
  )
}
