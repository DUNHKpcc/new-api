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
import axios from 'axios'

import { api, refreshAuthentication, type ApiRequestConfig } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { buildQueryParams } from './lib/query-params'
import { parseTaskArtifactsResponse } from './lib/task-artifacts'
import type {
  GetLogsParams,
  GetLogsResponse,
  GetLogStatsParams,
  GetLogStatsResponse,
  GetMidjourneyLogsParams,
  GetTaskLogsParams,
  TaskArtifactsResponse,
  UserInfo,
} from './types'

// ============================================================================
// Generic API Helpers
// ============================================================================

function buildApiPath(endpoint: string, isAdmin: boolean): string {
  if (!isAdmin) return `${endpoint.replace(/\/$/, '')}/self`
  if (endpoint === '/api/log' || endpoint === '/api/mj') {
    return `${endpoint}/`
  }
  return endpoint
}

function isInsufficientPrivilegeError(error: unknown): boolean {
  if (!axios.isAxiosError(error) || error.response?.status !== 403) {
    return false
  }
  const responseData = error.response.data
  return (
    typeof responseData === 'object' &&
    responseData !== null &&
    'code' in responseData &&
    responseData.code === 'AUTH_INSUFFICIENT_PRIVILEGE'
  )
}

/**
 * If a stale admin role selected an admin endpoint, refresh the server-backed
 * user role before falling back to the user's own data. A 403 is never
 * bypassed while the refreshed role is still administrative.
 */
export async function fetchWithPrivilegeFallback<T>(
  adminRequest: () => Promise<T>,
  selfRequest: () => Promise<T>,
  isAdmin: boolean,
  refresh = refreshAuthentication
): Promise<T> {
  if (!isAdmin) return selfRequest()

  try {
    return await adminRequest()
  } catch (error) {
    if (!isInsufficientPrivilegeError(error)) throw error

    const refreshOutcome = await refresh()
    const currentRole = useAuthStore.getState().auth.user?.role
    if (
      refreshOutcome.kind === 'authenticated' &&
      typeof currentRole === 'number' &&
      currentRole < ROLE.ADMIN
    ) {
      return selfRequest()
    }

    throw error
  }
}

function buildQueryParamsForScope(
  params: Record<string, unknown>,
  includeAdminFilters: boolean
): URLSearchParams {
  const scopedParams = { ...params }
  if (!includeAdminFilters) {
    delete scopedParams.username
    delete scopedParams.channel
  }
  return buildQueryParams({
    p: scopedParams.p || 1,
    page_size: scopedParams.page_size || 20,
    ...scopedParams,
  })
}

async function fetchLogs<T>(
  endpoint: string,
  params: T,
  isAdmin: boolean
): Promise<GetLogsResponse> {
  const paramRecord = params as unknown as Record<string, unknown>
  const adminQueryParams = buildQueryParamsForScope(paramRecord, true)
  const selfQueryParams = buildQueryParamsForScope(paramRecord, false)
  return fetchWithPrivilegeFallback(
    async () => {
      const res = await api.get(
        `${buildApiPath(endpoint, true)}?${adminQueryParams}`
      )
      return res.data
    },
    async () => {
      const res = await api.get(
        `${buildApiPath(endpoint, false)}?${selfQueryParams}`
      )
      return res.data
    },
    isAdmin
  )
}

async function fetchLogStats<T>(
  endpoint: string,
  params: T,
  isAdmin: boolean
): Promise<GetLogStatsResponse> {
  const paramRecord = params as unknown as Record<string, unknown>
  const adminQueryParams = buildQueryParamsForScope(paramRecord, true)
  const selfQueryParams = buildQueryParamsForScope(paramRecord, false)
  const buildStatsPath = (scope: boolean) =>
    `${buildApiPath(endpoint, scope).replace(/\/$/, '')}/stat`
  return fetchWithPrivilegeFallback(
    async () => {
      const res = await api.get(`${buildStatsPath(true)}?${adminQueryParams}`)
      return res.data
    },
    async () => {
      const res = await api.get(`${buildStatsPath(false)}?${selfQueryParams}`)
      return res.data
    },
    isAdmin
  )
}

// ============================================================================
// Common Log APIs
// ============================================================================

export const getAllLogs = (params: GetLogsParams = {}) =>
  fetchLogs('/api/log', params, true)

export const getUserLogs = (
  params: Omit<GetLogsParams, 'username' | 'channel'> = {}
) => fetchLogs('/api/log', params, false)

export const getLogStats = (params: GetLogStatsParams = {}) =>
  fetchLogStats('/api/log', params, true)

export const getUserLogStats = (
  params: Omit<GetLogStatsParams, 'username' | 'channel'> = {}
) => fetchLogStats('/api/log', params, false)

export async function getUserInfo(
  userId: number
): Promise<{ success: boolean; message?: string; data?: UserInfo }> {
  const res = await api.get(`/api/user/${userId}`)
  return res.data
}

// ============================================================================
// MjProxy (Drawing) Logs API
// ============================================================================

export const getAllMidjourneyLogs = (params: GetMidjourneyLogsParams) =>
  fetchLogs('/api/mj', params, true)

export const getUserMidjourneyLogs = (params: GetMidjourneyLogsParams) =>
  fetchLogs('/api/mj', params, false)

// ============================================================================
// Task Logs API
// ============================================================================

export const getAllTaskLogs = (params: GetTaskLogsParams) =>
  fetchLogs('/api/task', params, true)

export const getUserTaskLogs = (params: GetTaskLogsParams) =>
  fetchLogs('/api/task', params, false)

const taskArtifactRequestConfig = {
  skipBusinessError: true,
  skipErrorHandler: true,
} satisfies ApiRequestConfig

export async function getTaskArtifacts(taskId: string) {
  const response = await api.get<TaskArtifactsResponse>(
    `/api/task/${encodeURIComponent(taskId)}/artifacts`,
    taskArtifactRequestConfig
  )
  return parseTaskArtifactsResponse(response.data)
}
