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
import assert from 'node:assert/strict'
import { after, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
domWindow.document.write('<!doctype html><html><body></body></html>')
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Node',
  'Element',
  'MutationObserver',
] as const
const previousGlobalDescriptors = new Map<
  (typeof domGlobals)[number],
  PropertyDescriptor | undefined
>()

for (const key of domGlobals) {
  previousGlobalDescriptors.set(
    key,
    Object.getOwnPropertyDescriptor(globalThis, key)
  )
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
const previousActEnvironment = reactTestGlobals.IS_REACT_ACT_ENVIRONMENT
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { useNotifications } = await import('../use-notifications')

after(() => {
  for (const key of domGlobals) {
    const descriptor = previousGlobalDescriptors.get(key)
    if (descriptor) {
      Object.defineProperty(globalThis, key, descriptor)
    } else {
      Reflect.deleteProperty(globalThis, key)
    }
  }
  reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = previousActEnvironment
  domWindow.close()
})

test('periodically refreshes every remote notification source', async () => {
  const queryClient = new QueryClient()
  queryClient.setQueryData(['status'], { announcements_enabled: false })
  queryClient.setQueryData(['notice'], { success: true, data: '' })
  queryClient.setQueryData(['lottery-items'], { success: true, data: [] })

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  const Probe = () => {
    useNotifications()
    return null
  }

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <Probe />
      </QueryClientProvider>
    )
  })

  for (const queryKey of [['status'], ['notice'], ['lottery-items']]) {
    const query = queryClient.getQueryCache().find({ queryKey })
    const options = query?.options as
      | { refetchInterval?: number | false }
      | undefined
    assert.equal(options?.refetchInterval, 60 * 1000)
  }

  await act(async () => root.unmount())
  container.remove()
  queryClient.clear()
})
