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
import { LOTTERY_IMAGE_ASPECT_CLASS } from '@/features/lottery/lib/image-layout'
import { toDateTimeLocalValue } from '@/features/lottery/lib/lottery-items'
import type { LotteryItem } from '@/features/lottery/types'
import { cn } from '@/lib/utils'

type LotteryItemEditorProps = {
  item: LotteryItem
  index: number
  total: number
  uploading: boolean
  onChange: (item: LotteryItem) => void
  onImageChange: (file: File) => void
  onMove: (direction: -1 | 1) => void
  onRemove: () => void
}

export function LotteryItemEditor(props: LotteryItemEditorProps) {
  const { t } = useTranslation()
  const inputPrefix = `lottery-${props.item.id}`

  return (
    <div className='bg-muted/20 rounded-lg border p-3 sm:p-4'>
      <div className='mb-3 flex items-center justify-between gap-2'>
        <p className='text-sm font-medium'>
          {t('Lottery {{number}}', { number: props.index + 1 })}
        </p>
        <div className='flex items-center gap-1'>
          <Button
            type='button'
            size='icon-sm'
            variant='ghost'
            title={t('Move up')}
            aria-label={t('Move lottery up')}
            disabled={props.index === 0 || props.uploading}
            onClick={() => props.onMove(-1)}
          >
            <ChevronUp />
          </Button>
          <Button
            type='button'
            size='icon-sm'
            variant='ghost'
            title={t('Move down')}
            aria-label={t('Move lottery down')}
            disabled={props.index === props.total - 1 || props.uploading}
            onClick={() => props.onMove(1)}
          >
            <ChevronDown />
          </Button>
          <Button
            type='button'
            size='icon-sm'
            variant='destructive'
            title={t('Delete')}
            aria-label={t('Delete lottery')}
            disabled={props.uploading}
            onClick={props.onRemove}
          >
            <Trash2 />
          </Button>
        </div>
      </div>

      <div className='grid min-w-0 gap-4 md:grid-cols-[minmax(180px,0.7fr)_minmax(0,1.3fr)]'>
        <div className='min-w-0 space-y-2'>
          <Label htmlFor={`${inputPrefix}-image`}>{t('Lottery image')}</Label>
          <div
            className={cn(
              'bg-muted flex items-center justify-center overflow-hidden rounded-lg border',
              LOTTERY_IMAGE_ASPECT_CLASS
            )}
          >
            {props.item.image ? (
              <img
                src={props.item.image}
                alt={props.item.title || t('Lottery image')}
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
            id={`${inputPrefix}-image`}
            type='file'
            accept='image/jpeg,image/png,image/webp'
            disabled={props.uploading}
            aria-label={t('Upload lottery image')}
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (file) props.onImageChange(file)
              event.target.value = ''
            }}
          />
        </div>

        <div className='grid min-w-0 content-start gap-4'>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-title`}>{t('Title')}</Label>
            <Input
              id={`${inputPrefix}-title`}
              value={props.item.title}
              maxLength={80}
              required
              onChange={(event) =>
                props.onChange({ ...props.item, title: event.target.value })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-content`}>
              {t('Lottery content')}
            </Label>
            <Textarea
              id={`${inputPrefix}-content`}
              value={props.item.content}
              maxLength={500}
              rows={4}
              required
              onChange={(event) =>
                props.onChange({ ...props.item, content: event.target.value })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-winner`}>
              {t('Winning information')}
            </Label>
            <Textarea
              id={`${inputPrefix}-winner`}
              value={props.item.winnerInfo}
              maxLength={500}
              rows={3}
              onChange={(event) =>
                props.onChange({
                  ...props.item,
                  winnerInfo: event.target.value,
                })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${inputPrefix}-publish-date`}>
              {t('Publish Date')}
            </Label>
            <Input
              id={`${inputPrefix}-publish-date`}
              type='datetime-local'
              value={toDateTimeLocalValue(props.item.publishDate)}
              required
              onChange={(event) => {
                const date = new Date(event.target.value)
                props.onChange({
                  ...props.item,
                  publishDate: Number.isNaN(date.getTime())
                    ? ''
                    : date.toISOString(),
                })
              }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
