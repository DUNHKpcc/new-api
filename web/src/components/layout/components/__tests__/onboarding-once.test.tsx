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
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import {
  ConsoleOnboardingProvider,
  useConsoleOnboarding,
} from '../console-onboarding-context'

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: (
    selector: (state: { auth: { user: { id: number } } }) => unknown
  ) => selector({ auth: { user: { id: 17 } } }),
}))

function Guide() {
  const { isActive, complete } = useConsoleOnboarding('guardian')
  return isActive ? (
    <button type='button' onClick={complete}>
      Guardian guide
    </button>
  ) : null
}
function App(props: { visible?: boolean }) {
  return (
    <ConsoleOnboardingProvider>
      {props.visible !== false ? <Guide /> : null}
    </ConsoleOnboardingProvider>
  )
}
beforeEach(() => localStorage.clear())
afterEach(cleanup)

it('records the first display before dismissal and does not reopen after remount', () => {
  const first = render(<App />)
  expect(screen.getByRole('button', { name: 'Guardian guide' })).toBeTruthy()
  expect(
    JSON.parse(localStorage.getItem('console_onboarding_seen:17') ?? '[]')
  ).toContain('guardian')
  first.unmount()
  render(<App />)
  expect(screen.queryByRole('button', { name: 'Guardian guide' })).toBeNull()
})
it('does not reopen when its navigation entry disappears and returns', () => {
  const app = render(<App />)
  expect(screen.getByRole('button', { name: 'Guardian guide' })).toBeTruthy()
  app.rerender(<App visible={false} />)
  app.rerender(<App />)
  expect(screen.queryByRole('button', { name: 'Guardian guide' })).toBeNull()
})
it('keeps a dismissed guide closed across rerenders', () => {
  const app = render(<App />)
  fireEvent.click(screen.getByRole('button', { name: 'Guardian guide' }))
  app.rerender(<App />)
  expect(screen.queryByRole('button', { name: 'Guardian guide' })).toBeNull()
})
