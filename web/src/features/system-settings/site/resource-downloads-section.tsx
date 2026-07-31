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
import { Plus } from 'lucide-react'
import { nanoid } from 'nanoid'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { compressResourceThumbnail } from '@/features/resource-downloads/lib/compress-thumbnail'
import {
  MAX_RESOURCE_DOWNLOAD_ITEMS,
  isHttpDownloadUrl,
  parseResourceDownloadItems,
} from '@/features/resource-downloads/lib/resource-downloads'
import type { ResourceDownloadItem } from '@/features/resource-downloads/types'

import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { ResourceDownloadItemEditor } from './resource-download-item-editor'

type ResourceDownloadsSectionProps = {
  value: string
}

export function ResourceDownloadsSection(props: ResourceDownloadsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formRef = useRef<HTMLFormElement>(null)
  const initialItems = useMemo(
    () => parseResourceDownloadItems(props.value),
    [props.value]
  )
  const initialSerialized = useMemo(
    () => JSON.stringify(initialItems),
    [initialItems]
  )
  const [items, setItems] = useState(initialItems)
  const [uploadingId, setUploadingId] = useState<string | null>(null)
  const serialized = JSON.stringify(items)

  useEffect(() => {
    setItems(initialItems)
  }, [initialItems])

  const updateItem = (index: number, item: ResourceDownloadItem) => {
    setItems((current) =>
      current.map((existing, itemIndex) =>
        itemIndex === index ? item : existing
      )
    )
  }

  const moveItem = (index: number, direction: -1 | 1) => {
    setItems((current) => {
      const target = index + direction
      if (target < 0 || target >= current.length) return current
      const next = [...current]
      ;[next[index], next[target]] = [next[target], next[index]]
      return next
    })
  }

  const uploadThumbnail = async (index: number, file: File) => {
    const item = items[index]
    if (!item) return

    setUploadingId(item.id)
    try {
      const thumbnail = await compressResourceThumbnail(file)
      setItems((current) =>
        current.map((existing) =>
          existing.id === item.id ? { ...existing, thumbnail } : existing
        )
      )
      toast.success(t('Thumbnail uploaded'))
    } catch {
      toast.error(t('Thumbnail upload failed'))
    } finally {
      setUploadingId(null)
    }
  }

  const saveItems = async () => {
    const normalized = items.map((item) => ({
      ...item,
      name: item.name.trim(),
      description: item.description.trim(),
      url: item.url.trim(),
    }))
    const invalidIndex = normalized.findIndex(
      (item) =>
        !item.name ||
        !isHttpDownloadUrl(item.url) ||
        !item.thumbnail.startsWith('data:image/webp;base64,')
    )
    if (invalidIndex !== -1) {
      toast.error(
        t(
          'Complete the name, download URL, and thumbnail for resource {{number}}.',
          {
            number: invalidIndex + 1,
          }
        )
      )
      return
    }

    setItems(normalized)
    await updateOption.mutateAsync({
      key: 'ResourceDownloadItems',
      value: JSON.stringify(normalized),
    })
  }

  const addItem = () => {
    if (items.length >= MAX_RESOURCE_DOWNLOAD_ITEMS) return
    setItems((current) => [
      ...current,
      {
        id: nanoid(),
        name: '',
        description: '',
        url: '',
        thumbnail: '',
      },
    ])
  }

  return (
    <SettingsSection title={t('Resource Downloads')}>
      <SettingsPageFormActions
        onSave={() => formRef.current?.requestSubmit()}
        onReset={() => setItems(initialItems)}
        isSaving={updateOption.isPending}
        isSaveDisabled={
          serialized === initialSerialized || uploadingId !== null
        }
        isResetDisabled={serialized === initialSerialized}
        saveLabel='Save Changes'
        resetLabel='Reset'
      />

      <form
        ref={formRef}
        className='space-y-4'
        onSubmit={(event) => {
          event.preventDefault()
          void saveItems()
        }}
      >
        <div className='flex items-center justify-end'>
          <Button
            type='button'
            variant='outline'
            onClick={addItem}
            disabled={items.length >= MAX_RESOURCE_DOWNLOAD_ITEMS}
          >
            <Plus data-icon='inline-start' />
            <span>{t('Add resource')}</span>
          </Button>
        </div>

        {items.length === 0 ? (
          <Empty className='min-h-48 border'>
            <EmptyHeader>
              <EmptyTitle>{t('No resources configured')}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='space-y-3'>
            {items.map((item, index) => (
              <ResourceDownloadItemEditor
                key={item.id}
                item={item}
                index={index}
                total={items.length}
                uploading={uploadingId !== null}
                onChange={(nextItem) => updateItem(index, nextItem)}
                onThumbnailChange={(file) => {
                  void uploadThumbnail(index, file)
                }}
                onMove={(direction) => moveItem(index, direction)}
                onRemove={() =>
                  setItems((current) =>
                    current.filter((existing) => existing.id !== item.id)
                  )
                }
              />
            ))}
          </div>
        )}
      </form>
    </SettingsSection>
  )
}
