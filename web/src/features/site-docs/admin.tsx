import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout/components/section-page-layout'
import { LoadingState } from '@/components/loading-state'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { compressImageToWebP } from '@/features/resource-downloads/lib/compress-thumbnail'

import {
  getDocument,
  listDocuments,
  saveDocument,
  uploadDocumentImage,
  type SiteDocument,
} from './api'

export function DocumentEditor(props: {
  document: SiteDocument
  create: boolean
  onSaved: (slug: string) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [preview, setPreview] = useState(false)
  const contentInput = useRef<HTMLTextAreaElement>(null)
  const form = useForm<SiteDocument>({ defaultValues: props.document })
  const save = useMutation({
    mutationFn: (data: SiteDocument) => saveDocument(data, props.create),
    onSuccess: async (doc) => {
      form.reset(doc)
      await queryClient.invalidateQueries({ queryKey: ['site-docs'] })
      toast.success(t('Saved successfully'))
      props.onSaved(doc.slug)
    },
  })
  const upload = useMutation({
    mutationFn: async (input: {
      files: File[]
      content: string
      start: number
      end: number
    }) => {
      const images: string[] = []
      for (const file of input.files) {
        const image = await uploadDocumentImage(
          await compressImageToWebP(file, undefined, undefined, 'document')
        )
        const alt = file.name.replaceAll(/[[\]\r\n\\]/g, '')
        images.push(`![${alt}](${image.url})`)
      }
      return `\n\n${images.join('\n\n')}\n\n`
    },
    onSuccess: (markdown, input) => {
      form.setValue(
        'content',
        `${input.content.slice(0, input.start)}${markdown}${input.content.slice(input.end)}`,
        { shouldDirty: true }
      )
      const caret = input.start + markdown.length
      contentInput.current?.focus()
      contentInput.current?.setSelectionRange(caret, caret)
    },
  })

  const contentField = form.register('content')
  return (
    <form
      className='bg-card flex h-full min-h-0 min-w-0 flex-col overflow-hidden rounded-lg border'
      onSubmit={form.handleSubmit((data) => save.mutate(data))}
    >
      <div className='flex shrink-0 flex-wrap items-center justify-between gap-2 border-b px-4 py-3'>
        <span className='min-w-0 flex-1 truncate text-sm font-semibold'>
          {props.create ? t('New document') : props.document.title}
        </span>
        {upload.isPending && (
          <span role='status' className='text-muted-foreground text-xs'>
            {t('Uploading...')}
          </span>
        )}
        <Button
          type='button'
          size='sm'
          variant='outline'
          disabled={upload.isPending}
          onClick={() => setPreview(!preview)}
        >
          {preview ? t('Edit') : t('Preview')}
        </Button>
        <Button
          size='sm'
          type='submit'
          disabled={save.isPending || upload.isPending}
        >
          {save.isPending ? t('Saving...') : t('Save')}
        </Button>
      </div>
      <div className='min-h-0 flex-1 space-y-5 overflow-y-auto overscroll-contain p-4 sm:p-5'>
        <div className='grid gap-4 sm:grid-cols-2'>
          <div className='space-y-2'>
            <Label htmlFor='doc-title'>{t('Title')}</Label>
            <Input
              id='doc-title'
              maxLength={255}
              required
              {...form.register('title')}
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='doc-slug'>{t('Document URL slug')}</Label>
            <Input
              id='doc-slug'
              readOnly={!props.create}
              required
              pattern='[a-z0-9]+(-[a-z0-9]+)*'
              maxLength={160}
              {...form.register('slug')}
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='doc-category'>{t('Category')}</Label>
            <Input
              id='doc-category'
              maxLength={80}
              {...form.register('category')}
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='doc-order'>{t('Sort order')}</Label>
            <Input
              id='doc-order'
              type='number'
              {...form.register('sort_order', { valueAsNumber: true })}
            />
          </div>
        </div>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <label className='flex items-center gap-2 text-sm'>
            <Controller
              name='published'
              control={form.control}
              render={({ field }) => (
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              )}
            />
            {t('Published')}
          </label>
        </div>
        {preview ? (
          <RichContent content={form.getValues('content') ?? ''} />
        ) : (
          <div className='space-y-2'>
            <Label htmlFor='doc-content'>{t('Markdown content')}</Label>
            <Textarea
              id='doc-content'
              readOnly={upload.isPending}
              onPaste={(event) => {
                const files = [...event.clipboardData.items]
                  .filter(
                    (item) =>
                      item.kind === 'file' && item.type.startsWith('image/')
                  )
                  .map((item) => item.getAsFile())
                  .filter((file): file is File => file !== null)
                if (files.length === 0) return
                event.preventDefault()
                if (upload.isPending) return
                upload.mutate({
                  files,
                  content: form.getValues('content') ?? '',
                  start: event.currentTarget.selectionStart,
                  end: event.currentTarget.selectionEnd,
                })
              }}
              className='min-h-[480px] font-mono text-sm'
              required
              {...contentField}
              ref={(element) => {
                contentField.ref(element)
                contentInput.current = element
              }}
            />
          </div>
        )}
        <p className='text-muted-foreground text-xs'>
          {t(
            'Paste images into the editor to compress to WebP and insert at the selected position.'
          )}
        </p>
        <p className='text-muted-foreground text-xs'>
          {t(
            'Markdown content supports up to 60 KB. Unpublish to hide a document from visitors.'
          )}
        </p>
      </div>
    </form>
  )
}

const emptyDocument: SiteDocument = {
  slug: '',
  title: '',
  category: '',
  content: '',
  published: false,
  sort_order: 0,
  version: 0,
  updated_at: 0,
}
export function DocumentAdmin() {
  const { t } = useTranslation()
  const [selection, setSelection] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const catalog = useQuery({
    queryKey: ['site-docs', 'admin-catalog'],
    queryFn: () => listDocuments(true),
  })
  const selectedSlug = selection ?? catalog.data?.[0]?.slug ?? ''
  const doc = useQuery({
    queryKey: ['site-docs', 'admin-article', selectedSlug],
    queryFn: () => getDocument(selectedSlug, true),
    enabled: Boolean(selectedSlug),
  })
  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Document management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button variant='outline' render={<Link to='/docs' />}>
          {t('View docs')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {catalog.isPending && <LoadingState />}
        {catalog.isError && (
          <ErrorState onRetry={() => void catalog.refetch()} />
        )}
        {catalog.isSuccess && (
          <div className='grid h-full min-h-0 min-w-0 grid-rows-[160px_minmax(0,1fr)] gap-4 md:grid-cols-[230px_minmax(0,1fr)] md:grid-rows-1 xl:grid-cols-[270px_minmax(0,1fr)]'>
            <aside className='flex min-h-0 min-w-0 flex-col gap-3 rounded-lg border p-3'>
              <Button className='w-full' onClick={() => setSelection('')}>
                {t('New document')}
              </Button>
              <Input
                aria-label={t('Search documents')}
                placeholder={t('Search documents')}
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
              <nav
                aria-label={t('Document management')}
                className='min-h-0 flex-1 space-y-1 overflow-y-auto overscroll-contain'
              >
                {catalog.data
                  .filter((item) =>
                    `${item.title} ${item.category}`
                      .toLowerCase()
                      .includes(search.trim().toLowerCase())
                  )
                  .map((item) => (
                    <Button
                      key={item.slug}
                      variant={
                        selectedSlug === item.slug ? 'secondary' : 'ghost'
                      }
                      title={item.title}
                      aria-current={
                        selectedSlug === item.slug ? 'page' : undefined
                      }
                      className='h-10 w-full justify-start gap-2 px-2 text-left'
                      onClick={() => setSelection(item.slug)}
                    >
                      <span className='min-w-0 flex-1 truncate'>
                        {item.title}
                      </span>
                      {!item.published && (
                        <span className='text-muted-foreground ml-2 text-xs'>
                          {t('Draft')}
                        </span>
                      )}
                    </Button>
                  ))}
              </nav>
            </aside>
            {!selectedSlug && (
              <DocumentEditor
                key='new'
                document={emptyDocument}
                create
                onSaved={setSelection}
              />
            )}
            {selectedSlug && doc.isPending && <LoadingState />}
            {selectedSlug && doc.isError && (
              <ErrorState onRetry={() => void doc.refetch()} />
            )}
            {selectedSlug && doc.isSuccess && (
              <DocumentEditor
                key={selectedSlug}
                document={doc.data}
                create={false}
                onSaved={setSelection}
              />
            )}
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
