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
type RecoveryStorage = Pick<Storage, 'getItem' | 'setItem'>

const STALE_ASSET_RELOAD_PREFIX = 'app:stale-asset-reload:'
const STALE_ASSET_MESSAGES = [
  /Loading chunk .+ failed/i,
  /Loading CSS chunk .+ failed/i,
  /Failed to fetch dynamically imported module/i,
  /Importing a module script failed/i,
]

export function isStaleAssetError(error: unknown): boolean {
  if (typeof error === 'string') {
    return STALE_ASSET_MESSAGES.some((pattern) => pattern.test(error))
  }
  if (typeof error !== 'object' || error === null) return false

  const candidate = error as {
    name?: unknown
    message?: unknown
    cause?: unknown
  }
  if (candidate.name === 'ChunkLoadError') return true
  const message = candidate.message
  if (
    typeof message === 'string' &&
    STALE_ASSET_MESSAGES.some((pattern) => pattern.test(message))
  ) {
    return true
  }

  return candidate.cause !== error && isStaleAssetError(candidate.cause)
}

export function attemptStaleAssetRecovery(
  error: unknown,
  buildRevision: string,
  storage: RecoveryStorage,
  reload: () => void
): boolean {
  if (!isStaleAssetError(error)) return false

  const recoveryKey = `${STALE_ASSET_RELOAD_PREFIX}${buildRevision}`
  try {
    if (storage.getItem(recoveryKey) === '1') return false
    storage.setItem(recoveryKey, '1')
  } catch {
    return false
  }

  reload()
  return true
}
