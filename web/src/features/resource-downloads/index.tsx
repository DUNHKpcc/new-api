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
import { useQuery } from '@tanstack/react-query'
import { Download, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'

import { getResourceDownloads } from './api'
import {
  ResourceDownloadsGrid,
  resourceDownloadsGridClassName,
} from './components/resource-downloads-grid'

export function ResourceDownloads() {
  const { t } = useTranslation()
  const resources = useQuery({
    queryKey: ['resource-downloads'],
    queryFn: getResourceDownloads,
    staleTime: 5 * 60 * 1000,
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Resource Downloads')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='w-full'>
          {resources.isLoading ? (
            <div className={resourceDownloadsGridClassName}>
              {Array.from({ length: 4 }, (_, index) => (
                <div key={index} className='overflow-hidden rounded-lg border'>
                  <Skeleton className='aspect-video rounded-none' />
                  <div className='space-y-3 p-4'>
                    <Skeleton className='h-5 w-2/3' />
                    <Skeleton className='h-4 w-full' />
                    <Skeleton className='h-8 w-full' />
                  </div>
                </div>
              ))}
            </div>
          ) : null}

          {resources.isError ? (
            <Empty className='min-h-64 border'>
              <EmptyHeader>
                <EmptyMedia variant='icon'>
                  <Download />
                </EmptyMedia>
                <EmptyTitle>{t('Resources could not be loaded')}</EmptyTitle>
                <EmptyDescription>
                  {t('Please try again in a moment.')}
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button variant='outline' onClick={() => resources.refetch()}>
                  <RefreshCw data-icon='inline-start' />
                  <span>{t('Retry')}</span>
                </Button>
              </EmptyContent>
            </Empty>
          ) : null}

          {resources.isSuccess && resources.data.length === 0 ? (
            <Empty className='min-h-64 border'>
              <EmptyHeader>
                <EmptyMedia variant='icon'>
                  <Download />
                </EmptyMedia>
                <EmptyTitle>{t('No resources available')}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : null}

          {resources.isSuccess && resources.data.length > 0 ? (
            <ResourceDownloadsGrid items={resources.data} />
          ) : null}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
