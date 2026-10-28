import assert from 'node:assert/strict'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { createInstance } from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { describe, test } from 'vitest'

import { SiteDocs } from '..'
import { DocumentAdmin } from '../admin'

async function renderDocsPage(article: string, admin = false) {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    resources: { en: { translation: {} } },
  })

  const queryClient = new QueryClient({
    defaultOptions: { queries: { staleTime: Infinity } },
  })
  const docs = [
    {
      slug: 'first',
      title: 'First guide',
      category: 'Guides',
      published: true,
      sort_order: 0,
      version: 1,
      updated_at: 1,
    },
    {
      slug: 'second',
      title: 'Second guide',
      category: 'Guides',
      published: true,
      sort_order: 1,
      version: 1,
      updated_at: 1,
    },
  ]
  queryClient.setQueryData(['site-docs', 'catalog'], docs)
  queryClient.setQueryData(['site-docs', 'article', 'first'], {
    ...docs[0],
    content: 'First article body',
  })
  queryClient.setQueryData(['site-docs', 'article', 'second'], {
    ...docs[1],
    content: 'Second article body\n\n![Guide](/site-doc-assets/example.webp)',
  })
  queryClient.setQueryData(['site-docs', 'admin-catalog'], docs)
  queryClient.setQueryData(['site-docs', 'admin-article', 'first'], {
    ...docs[0],
    content: 'Editable body',
  })
  const TestPage = () =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        I18nextProvider,
        { i18n },
        admin
          ? createElement(DocumentAdmin)
          : createElement(SiteDocs, { article })
      )
    )
  const rootRoute = createRootRoute()
  const supportRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/docs',
    component: TestPage,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([supportRoute]),
    history: createMemoryHistory({ initialEntries: ['/docs'] }),
  })

  await router.load()
  return renderToStaticMarkup(createElement(RouterProvider, { router }))
}

describe('site documentation', () => {
  test('renders the requested article without including other article bodies', async () => {
    const markup = await renderDocsPage('second')
    assert.match(markup, /Second article body/)
    assert.doesNotMatch(markup, /First article body/)
    assert.match(markup, /First guide/)
    assert.match(markup, /Second guide/)
    assert.match(markup, /aria-current="page"/)
    assert.match(markup, /Search documents/)
    assert.match(markup, /loading="lazy"/)
    assert.match(markup, /decoding="async"/)
  })
})

test('admin keeps the document list compact and both panes independently scrollable', async () => {
  const markup = await renderDocsPage('first', true)
  const container = window.document.createElement('div')
  container.innerHTML = markup
  const list = container.querySelector('nav[aria-label="Document management"]')
  assert.ok(list?.classList.contains('overflow-y-auto'))
  assert.ok(list?.querySelector('button .truncate'))
  const editor = container.querySelector('textarea')
  assert.ok(editor?.closest('.overflow-y-auto'))
  assert.ok(container.querySelector('form')?.classList.contains('min-h-0'))
})
