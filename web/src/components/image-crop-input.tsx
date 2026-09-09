import { Check, RotateCcw, X } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import Cropper, { type Area } from 'react-easy-crop'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { compressImageToWebP } from '@/features/resource-downloads/lib/compress-thumbnail'

type ImageCropInputProps = {
  id?: string
  value?: string
  aspectRatio: number
  disabled?: boolean
  label: string
  onChange: (value: string) => void
}

export function ImageCropInput(props: ImageCropInputProps) {
  const { t } = useTranslation()
  const zoomId = useId()
  const generation = useRef(0)
  const [source, setSource] = useState<{ file: File; url: string } | null>(null)
  const [crop, setCrop] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [area, setArea] = useState<Area | null>(null)
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
        <div className='relative h-64 w-full overflow-hidden rounded-md bg-black sm:h-80'>
          <Cropper
            image={source.url}
            crop={crop}
            zoom={zoom}
            aspect={props.aspectRatio}
            onCropChange={setCrop}
            onZoomChange={setZoom}
            onCropComplete={(_, pixels) => setArea(pixels)}
            zoomWithScroll={false}
            mediaProps={{
              onError: () => {
                setError(true)
                setArea(null)
              },
            }}
          />
        </div>
      ) : (
        <div
          className='bg-muted overflow-hidden rounded-md border'
          style={{ aspectRatio: props.aspectRatio }}
        >
          {props.value ? (
            <img
              src={props.value}
              alt={props.label}
              className='size-full object-cover'
            />
          ) : null}
        </div>
      )}
      {source ? (
        <>
          <label htmlFor={zoomId} className='text-sm'>
            {t('Zoom')}
          </label>
          <input
            id={zoomId}
            type='range'
            min={1}
            max={3}
            step={0.01}
            value={zoom}
            disabled={busy || props.disabled}
            className='w-full'
            onChange={(event) => setZoom(Number(event.currentTarget.value))}
          />
          <div className='flex flex-wrap gap-2'>
            <Button
              type='button'
              size='icon-sm'
              variant='outline'
              title={t('Reset')}
              aria-label={t('Reset')}
              disabled={busy || props.disabled}
              onClick={() => {
                setCrop({ x: 0, y: 0 })
                setZoom(1)
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
                setError(false)
              }}
            >
              <X />
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              disabled={busy || props.disabled || !area || error}
              onClick={async () => {
                if (!area || busy) return
                const request = ++generation.current
                setBusy(true)
                setError(false)
                try {
                  const image = await compressImageToWebP(
                    source.file,
                    props.aspectRatio,
                    area
                  )
                  if (request !== generation.current) return
                  props.onChange(image)
                  setSource(null)
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
        </>
      ) : null}
      <Input
        id={props.id}
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
          setCrop({ x: 0, y: 0 })
          setZoom(1)
          setArea(null)
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
