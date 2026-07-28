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
import { useTranslation } from 'react-i18next'

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import type { UsersView } from '../types'

interface UsersViewTabsProps {
  value: UsersView
  onValueChange: (value: UsersView) => void
}

export function UsersViewTabs(props: UsersViewTabsProps) {
  const { t } = useTranslation()

  const handleValueChange = (value: string) => {
    if (value === 'all' || value === 'pcc_agent') {
      props.onValueChange(value)
    }
  }

  return (
    <Tabs value={props.value} onValueChange={handleValueChange}>
      <TabsList aria-label={t('User view')}>
        <TabsTrigger value='all'>{t('All Users')}</TabsTrigger>
        <TabsTrigger value='pcc_agent'>
          {t('PccAgent Authorized Users')}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}
