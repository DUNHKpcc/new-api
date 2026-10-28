import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { DocumentEditor } from '../admin'
import { DocumentBody } from '../document-body'

const i18n = createInstance()
await i18n
  .use(initReactI18next)
  .init({ lng: 'en', resources: { en: { translation: {} } } })
const document = {
  slug: 'guide',
  title: 'Guide',
  category: '',
  content: 'Before SELECT After',
  published: true,
  sort_order: 0,
  version: 1,
  updated_at: 0,
}
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

test('pasting an image uploads WebP and replaces the original selected text', async () => {
  vi.stubGlobal(
    'createImageBitmap',
    vi.fn().mockResolvedValue({ width: 1000, height: 800, close: vi.fn() })
  )
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
    drawImage: vi.fn(),
  } as unknown as CanvasRenderingContext2D)
  vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation(
    (callback) => callback(new Blob(['webp'], { type: 'image/webp' }))
  )
  let complete: ((value: unknown) => void) | undefined
  const post = vi.spyOn(api, 'post').mockImplementation(
    () =>
      new Promise((resolve) => {
        complete = resolve
      })
  )
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <I18nextProvider i18n={i18n}>
        <DocumentEditor
          document={document}
          create={false}
          onSaved={() => undefined}
        />
      </I18nextProvider>
    </QueryClientProvider>
  )
  const editor = screen.getByLabelText(
    'Markdown content'
  ) as HTMLTextAreaElement
  editor.setSelectionRange(7, 13)
  fireEvent.paste(editor, {
    clipboardData: {
      items: [
        {
          kind: 'file',
          type: 'image/png',
          getAsFile: () => new File(['png'], 'shot.png', { type: 'image/png' }),
        },
      ],
    },
  })
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/site-docs-admin/images', {
      image: expect.stringMatching(/^data:image\/webp;base64,/),
    })
  )
  expect(editor.readOnly).toBe(true)
  editor.setSelectionRange(0, 0)
  await act(async () =>
    complete?.({
      data: {
        success: true,
        data: { url: '/api/site-doc-images/hash', size: 4 },
      },
    })
  )
  await waitFor(() =>
    expect(editor.value).toBe(
      'Before \n\n![shot.png](/api/site-doc-images/hash)\n\n After'
    )
  )
  expect(editor.readOnly).toBe(false)
  expect(editor.selectionStart).toBe(editor.value.indexOf(' After'))
  client.clear()
})

test('article headings produce unique matching anchors and ignore fenced code', async () => {
  render(
    <I18nextProvider i18n={i18n}>
      <DocumentBody
        document={{
          ...document,
          content:
            '# Guide\n\n## Setup\n\n### Detail\n\n## Setup\n\n```md\n## Not a section\n```',
        }}
      />
    </I18nextProvider>
  )
  const nav = await screen.findByRole('navigation', { name: 'On this page' })
  const links = [...nav.querySelectorAll('a')]
  expect(links.map((link) => link.textContent)).toEqual([
    'Guide',
    'Setup',
    'Detail',
    'Setup',
  ])
  expect(new Set(links.map((link) => link.hash)).size).toBe(4)
  for (const link of links) {
    expect(window.document.querySelector(link.hash)).not.toBeNull()
  }
})
