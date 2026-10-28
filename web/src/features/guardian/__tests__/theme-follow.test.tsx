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
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { AnchorHTMLAttributes, ReactNode } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { ThemeProvider } from '@/context/theme-provider'
import { THEME_STORAGE_KEYS } from '@/lib/theme-storage'

import { GuardianShell } from '../components/guardian-shell'

vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  Link: (props: AnchorHTMLAttributes<HTMLAnchorElement> & { to?: string }) => (
    <a href={props.to}>{props.children}</a>
  ),
}))

let store: Map<string, string>

beforeEach(() => {
  store = new Map()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  })
  document.documentElement.classList.remove('dark', 'light')
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  document.documentElement.classList.remove('dark', 'light')
})

function renderShell(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ThemeProvider>
        <GuardianShell>{children}</GuardianShell>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

it('opens in the dark mode the main site saved', async () => {
  store.set(THEME_STORAGE_KEYS.mode, 'dark')
  renderShell(<p>content</p>)
  await waitFor(() => expect(document.documentElement).toHaveClass('dark'))
})

it('switches the whole site to dark from the Guardian header and saves the shared preference', async () => {
  store.set(THEME_STORAGE_KEYS.mode, 'light')
  renderShell(<p>content</p>)
  await waitFor(() => expect(document.documentElement).toHaveClass('light'))

  await userEvent.click(screen.getByRole('button', { name: 'Toggle theme' }))
  await userEvent.click(await screen.findByRole('menuitem', { name: 'Dark' }))

  await waitFor(() => expect(document.documentElement).toHaveClass('dark'))
  expect(store.get(THEME_STORAGE_KEYS.mode)).toBe('dark')
})
