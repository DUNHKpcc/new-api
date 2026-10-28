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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { GUARDIAN_REPO_API } from '../constants'
import { fetchGithubStars, readCachedStars } from '../hooks/use-github-stars'

let store: Map<string, string>

beforeEach(() => {
  store = new Map()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function respond(status: number, body: unknown) {
  return vi
    .spyOn(globalThis, 'fetch')
    .mockResolvedValue(new Response(JSON.stringify(body), { status }))
}

describe('fetchGithubStars', () => {
  it('reads the star count from the repository API and remembers it', async () => {
    const fetchSpy = respond(200, { stargazers_count: 1142 })

    await expect(fetchGithubStars()).resolves.toBe(1142)

    expect(fetchSpy).toHaveBeenCalledWith(GUARDIAN_REPO_API, expect.anything())
    expect(readCachedStars()?.count).toBe(1142)
  })

  it('rejects a rate-limited response without overwriting the remembered count', async () => {
    store.set('guardian:github-stars:v1', JSON.stringify({ count: 900, at: 1 }))
    respond(403, { message: 'API rate limit exceeded' })

    await expect(fetchGithubStars()).rejects.toThrow('403')
    expect(readCachedStars()).toEqual({ count: 900, at: 1 })
  })

  it.each([
    ['a missing count', {}],
    ['a negative count', { stargazers_count: -1 }],
    ['a fractional count', { stargazers_count: 1.5 }],
    ['a string count', { stargazers_count: '1142' }],
  ])('rejects %s instead of showing it', async (_label, body) => {
    respond(200, body)
    await expect(fetchGithubStars()).rejects.toThrow()
    expect(readCachedStars()).toBeUndefined()
  })
})

describe('readCachedStars', () => {
  it('ignores a corrupted remembered value', () => {
    store.set('guardian:github-stars:v1', '{not json')
    expect(readCachedStars()).toBeUndefined()
    store.set('guardian:github-stars:v1', JSON.stringify({ count: 'many' }))
    expect(readCachedStars()).toBeUndefined()
  })
})
