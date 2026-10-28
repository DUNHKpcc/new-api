import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { PublicLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

import { getDocument, listDocuments } from './api'
import { DocumentBody } from './document-body'

export function SiteDocs(props: { article?: string }) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const catalog = useQuery({
    queryKey: ['site-docs', 'catalog'],
    queryFn: () => listDocuments(),
    staleTime: 60_000,
  })
  const slug = props.article || catalog.data?.[0]?.slug || ''
  const article = useQuery({
    queryKey: ['site-docs', 'article', slug],
    queryFn: () => getDocument(slug),
    enabled: Boolean(slug),
    staleTime: 60_000,
  })
  const groups = useMemo(() => {
    const result = new Map<string, NonNullable<typeof catalog.data>>()
    const keyword = search.trim().toLocaleLowerCase()
    for (const doc of catalog.data ?? []) {
      if (
        keyword &&
        !`${doc.title} ${doc.category}`.toLocaleLowerCase().includes(keyword)
      ) {
        continue
      }
      const group = result.get(doc.category) ?? []
      group.push(doc)
      result.set(doc.category, group)
    }
    return [...result]
  }, [catalog.data, search])

  return (
    <PublicLayout>
      <div className='mx-auto w-full max-w-[1440px] py-8 sm:py-12'>
        <h1 className='mb-8 text-3xl font-semibold'>{t('Docs')}</h1>
        {catalog.isPending && <LoadingState />}
        {catalog.isError && (
          <ErrorState onRetry={() => void catalog.refetch()} />
        )}
        {catalog.isSuccess && (
          <div className='grid min-w-0 gap-6 lg:grid-cols-[200px_minmax(0,1fr)_160px] xl:grid-cols-[240px_minmax(0,1fr)_190px] xl:gap-10'>
            <aside className='min-w-0 lg:sticky lg:top-24 lg:max-h-[calc(100svh-8rem)] lg:self-start lg:overflow-y-auto'>
              <Input
                aria-label={t('Search documents')}
                placeholder={t('Search documents')}
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
              <nav aria-label={t('Docs')} className='mt-5 space-y-5'>
                {groups.map(([category, docs]) => (
                  <div key={category}>
                    {category && (
                      <h2 className='text-muted-foreground mb-2 text-xs font-semibold'>
                        {category}
                      </h2>
                    )}
                    <ul className='space-y-1'>
                      {docs.map((doc) => (
                        <li key={doc.slug}>
                          <Link
                            to='/docs'
                            search={{ article: doc.slug }}
                            resetScroll
                            aria-current={
                              slug === doc.slug ? 'page' : undefined
                            }
                            className={cn(
                              'block rounded-md px-3 py-2 text-sm leading-6 hover:bg-muted',
                              slug === doc.slug &&
                                'bg-primary/10 text-primary font-medium'
                            )}
                          >
                            {doc.title}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
                {groups.length === 0 && (
                  <p className='text-muted-foreground text-sm'>
                    {t('No documents found')}
                  </p>
                )}
              </nav>
            </aside>
            {!slug && <p>{t('No documents found')}</p>}
            {slug && article.isPending && <LoadingState />}
            {slug && article.isError && (
              <ErrorState onRetry={() => void article.refetch()} />
            )}
            {slug && article.isSuccess && (
              <DocumentBody key={slug} document={article.data} />
            )}
          </div>
        )}
      </div>
    </PublicLayout>
  )
}
