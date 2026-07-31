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
import { Download } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import type { ResourceDownloadItem } from '../types'

type ResourceDownloadsGridProps = {
  items: ResourceDownloadItem[]
}

export const resourceDownloadsGridClassName =
  'grid grid-cols-[repeat(auto-fill,minmax(min(100%,17rem),1fr))] gap-4'

export function ResourceDownloadsGrid(props: ResourceDownloadsGridProps) {
  const { t } = useTranslation()

  return (
    <div
      data-layout='responsive-resource-grid'
      className={resourceDownloadsGridClassName}
    >
      {props.items.map((item) => (
        <article
          key={item.id}
          className='bg-card text-card-foreground grid h-full grid-rows-[auto_1fr] overflow-hidden rounded-lg border'
        >
          <div className='bg-muted aspect-video overflow-hidden'>
            <img
              src={item.thumbnail}
              alt={item.name}
              className='size-full object-cover'
              loading='lazy'
            />
          </div>
          <div className='flex min-h-36 flex-col gap-3 p-4'>
            <div className='min-w-0 flex-1'>
              <h3 className='line-clamp-2 text-base font-semibold'>
                {item.name}
              </h3>
              {item.description ? (
                <p className='text-muted-foreground mt-1 line-clamp-2 text-sm'>
                  {item.description}
                </p>
              ) : null}
            </div>
            <Button
              variant='outline'
              className='w-full'
              render={
                <a href={item.url} target='_blank' rel='noopener noreferrer' />
              }
            >
              <Download data-icon='inline-start' />
              <span>{t('Download')}</span>
            </Button>
          </div>
        </article>
      ))}
    </div>
  )
}
