import { memo, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { cn } from '@/lib/utils'

import type { SiteDocument } from './api'

// Keep the rendered body stable when the derived table of contents updates.
const DocumentMarkdown = memo(RichContent)

type Heading = { id: string; text: string; level: number }

export function DocumentBody(props: { document: SiteDocument }) {
  const { t } = useTranslation()
  const bodyRef = useRef<HTMLDivElement>(null)
  const [headings, setHeadings] = useState<Heading[]>([])
  useEffect(() => {
    const elements = bodyRef.current?.querySelectorAll('h1, h2, h3, h4') ?? []
    const items = Array.from(elements, (element, index) => {
      const id = `doc-${props.document.slug}-section-${index + 1}`
      element.id = id
      element.classList.add('scroll-mt-28')
      return {
        id,
        text: element.textContent ?? '',
        level: Number(element.tagName[1]),
      }
    })
    setHeadings(items)
  }, [props.document.slug, props.document.content])

  const content = props.document.content ?? ''
  const startsWithTitle = content.startsWith(`# ${props.document.title}\n`)
  return (
    <>
      <article className='min-w-0 lg:col-start-2 lg:row-start-1'>
        {!startsWithTitle && (
          <h2 className='mb-6 text-2xl font-semibold'>
            {props.document.title}
          </h2>
        )}
        <div ref={bodyRef}>
          <DocumentMarkdown lazyImages content={content} />
        </div>
      </article>
      <aside className='hidden min-w-0 self-start border-l pl-4 lg:sticky lg:top-24 lg:col-start-3 lg:row-start-1 lg:block lg:max-h-[calc(100svh-8rem)] lg:overflow-y-auto'>
        <h2 className='mb-3 text-sm font-semibold'>{t('On this page')}</h2>
        <nav aria-label={t('On this page')}>
          <ul className='space-y-2'>
            {headings.map((heading) => (
              <li key={heading.id}>
                <a
                  href={`#${heading.id}`}
                  className={cn(
                    'text-muted-foreground hover:text-foreground block text-xs leading-5 break-words',
                    heading.level >= 3 && 'pl-3'
                  )}
                >
                  {heading.text}
                </a>
              </li>
            ))}
          </ul>
        </nav>
      </aside>
    </>
  )
}
