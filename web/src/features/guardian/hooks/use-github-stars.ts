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
import { useQuery } from '@tanstack/react-query'

import { GUARDIAN_QUERY_KEYS, GUARDIAN_REPO_API } from '../constants'

const STORAGE_KEY = 'guardian:github-stars:v1'
/** Unauthenticated GitHub API calls are limited to 60 per hour per IP. */
const STALE_MS = 60 * 60 * 1000

type CachedStars = { count: number; at: number }

function isValidCount(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0
}

export function readCachedStars(): CachedStars | undefined {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return undefined
    const parsed = JSON.parse(raw) as Partial<CachedStars>
    if (!isValidCount(parsed.count) || typeof parsed.at !== 'number') {
      return undefined
    }
    return { count: parsed.count, at: parsed.at }
  } catch {
    return undefined
  }
}

function writeCachedStars(count: number) {
  try {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ count, at: Date.now() })
    )
  } catch {
    // Storage can be unavailable in private windows; the live value still shows.
  }
}

export async function fetchGithubStars(): Promise<number> {
  const response = await fetch(GUARDIAN_REPO_API, {
    headers: { Accept: 'application/vnd.github+json' },
  })
  if (!response.ok) {
    throw new Error(`GitHub responded with ${response.status}`)
  }
  const body = (await response.json()) as { stargazers_count?: unknown }
  if (!isValidCount(body.stargazers_count)) {
    throw new Error('GitHub response has no star count')
  }
  writeCachedStars(body.stargazers_count)
  return body.stargazers_count
}

/** Live star count of the Claude-Guardian repository, or the last one seen. */
export function useGithubStars() {
  const cached = readCachedStars()
  return useQuery({
    queryKey: GUARDIAN_QUERY_KEYS.githubStars,
    queryFn: fetchGithubStars,
    initialData: cached?.count,
    initialDataUpdatedAt: cached?.at,
    staleTime: STALE_MS,
    retry: false,
    refetchOnWindowFocus: false,
    meta: { errorToast: false },
  })
}
