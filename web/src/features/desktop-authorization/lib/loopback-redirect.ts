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
const callbackPathPattern = /^\/oauth\/callback\/[A-Za-z0-9_-]{22,}$/

export function isSafeDesktopLoopbackRedirect(value: unknown): value is string {
  if (typeof value !== 'string') return false

  let redirect: URL
  try {
    redirect = new URL(value)
  } catch {
    return false
  }

  const port = Number.parseInt(redirect.port, 10)
  const hasResult =
    redirect.searchParams.has('code') || redirect.searchParams.has('error')
  return (
    redirect.protocol === 'http:' &&
    redirect.hostname === '127.0.0.1' &&
    redirect.username === '' &&
    redirect.password === '' &&
    redirect.hash === '' &&
    port >= 1024 &&
    port <= 65_535 &&
    callbackPathPattern.test(redirect.pathname) &&
    hasResult &&
    redirect.searchParams.has('state')
  )
}
