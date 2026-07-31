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
  ChevronDown,
  ChevronUp,
  Image as ImageIcon,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { ResourceDownloadItem } from '@/features/resource-downloads/types'

type ResourceDownloadItemEditorProps = {
  item: ResourceDownloadItem
  index: number
  total: number
  uploading: boolean
  onChange: (item: ResourceDownloadItem) => void
  onThumbnailChange: (file: File) => void
  onMove: (direction: -1 | 1) => void
  onRemove: () => void
}

export function ResourceDownloadItemEditor(
  props: ResourceDownloadItemEditorProps
) {
  const { t } = useTranslation()
  const inputPrefix = `resource-download-${props.item.id}`

  return (
    <div className='bg-muted/20 rounded-lg border p-3 sm:p-4'>
      <div className='mb-3 flex items-center justify-between gap-2'>
        <p className='text-sm font-medium'>
          {t('Resource {{number}}', { number: props.index + 1 })}
        </p>
        <div className='flex items-center gap-1'>
          <Button
            type='button'
            size='icon-sm'
            variant='ghost'
            title={t('Move up')}
            aria-label={t('Move resource up')}
            disabled={props.index === 0}
            onClick={() => props.onMove(-1)}
          >
            <ChevronUp />
          </Button>
          <Button
            type='button'
            size='icon-sm'
            variant='ghost'
            title={t('Move down')}
            aria-label={t('Move resource down')}
            disabled={props.index === props.total - 1}
            onClick={() => props.onMove(1)}
          >
            <ChevronDown />
          </Button>
          <Button
            type='button'
            size='icon-sm'
            variant='destructive'
            title={t('Delete')}
            aria-label={t('Delete resource')}
            onClick={props.onRemove}
          >
            <Trash2 />
          </Button>
        </div>
      </div>

      <div className='grid min-w-0 gap-4 md:grid-cols-[minmax(180px,0.7fr)_minmax(0,1.3fr)]'>
        <div className='min-w-0 space-y-2'>
          <Label htmlFor={`${inputPrefix}-thumbnail`}>{t('Thumbnail')}</Label>
          <div className='bg-muted flex aspect-video items-center justify-center overflow-hidden rounded-lg border'>
            {props.item.thumbnail ? (
              <img
                src={props.item.thumbnail}
                alt={props.item.name || t('Resource thumbnail')}
                className='size-full object-cover'
              />
            ) : (
              <ImageIcon
                className='text-muted-foreground size-7'
                aria-hidden='true'
              />
            )}
          </div>
          <Input
            id={`${inputPrefix}-thumbnail`}
            type='file'
            accept='image/jpeg,image/png,image/webp'
            disabled={props.uploading}
            aria-label={t('Upload thumbnail')}
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (file) props.onThumbnailChange(file)
              event.target.value = ''
            }}
          />
        </div>

        <div className='grid min-w-0 content-start gap-4'>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-name`}>{t('Name')}</Label>
            <Input
              id={`${inputPrefix}-name`}
              value={props.item.name}
              maxLength={80}
              required
              onChange={(event) =>
                props.onChange({ ...props.item, name: event.target.value })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-url`}>{t('Download URL')}</Label>
            <Input
              id={`${inputPrefix}-url`}
              type='url'
              value={props.item.url}
              maxLength={2048}
              required
              placeholder='https://'
              onChange={(event) =>
                props.onChange({ ...props.item, url: event.target.value })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-description`}>
              {t('Description')}
            </Label>
            <Textarea
              id={`${inputPrefix}-description`}
              value={props.item.description}
              maxLength={200}
              rows={3}
              onChange={(event) =>
                props.onChange({
                  ...props.item,
                  description: event.target.value,
                })
              }
            />
          </div>
        </div>
      </div>
    </div>
  )
}
