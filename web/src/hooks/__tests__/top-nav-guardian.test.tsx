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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useTopNavLinks } from '../use-top-nav-links'

beforeEach(() => {
  vi.stubGlobal('localStorage', {
    getItem: () => null,
    setItem: () => undefined,
    removeItem: () => undefined,
  })
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function linksFor(headerNavModules: object | null) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {
    docs_link: 'https://example.test/old-docs',
    HeaderNavModules: headerNavModules ? JSON.stringify(headerNavModules) : '',
  })
  function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
  }
  const { result } = renderHook(() => useTopNavLinks(), { wrapper: Wrapper })
  return result.current.map((link) => link.href)
}

describe('Guardian top navigation entry', () => {
  it('shows the Guardian page link when HeaderNavModules has no guardian key', () => {
    expect(linksFor(null)).toContain('/guardian')
    expect(linksFor({ home: true, pricing: { enabled: false } })).toContain(
      '/guardian'
    )
  })

  it('hides the Guardian page link when HeaderNavModules sets guardian to false', () => {
    expect(linksFor({ guardian: false })).not.toContain('/guardian')
  })
})

it('keeps documentation inside the site and respects the visibility setting', () => {
  expect(linksFor(null)).toContain('/docs')
  expect(linksFor(null)).not.toContain('https://example.test/old-docs')
  expect(linksFor({ docs: false })).not.toContain('/docs')
})
