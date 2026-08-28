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
import { TimeRangeFilterDialog } from '@/components/time-range-filter-dialog'
import { cleanFilters } from '@/features/dashboard/lib'
import type {
  DashboardChartPreferences,
  DashboardFilters,
} from '@/features/dashboard/types'
import { useAuthStore } from '@/stores/auth-store'

interface ModelsFilterProps {
  preferences: DashboardChartPreferences
  currentFilters: DashboardFilters
  onFilterChange: (filters: DashboardFilters) => void
  onReset: () => void
  titleKey?: string
  descriptionKey?: string
}

export function ModelsFilter(props: ModelsFilterProps) {
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = Boolean(user?.role && user.role >= 10)

  return (
    <TimeRangeFilterDialog
      currentFilters={props.currentFilters}
      defaultTimeRangeDays={props.preferences.defaultTimeRangeDays}
      defaultTimeGranularity={props.preferences.defaultTimeGranularity}
      onFilterChange={(filters) =>
        props.onFilterChange(
          cleanFilters(filters as Record<string, unknown>) as DashboardFilters
        )
      }
      onReset={props.onReset}
      titleKey={props.titleKey}
      descriptionKey={props.descriptionKey}
      showGranularity
      showUsername={isAdmin}
    />
  )
}
