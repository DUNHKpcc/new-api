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
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { ExternalRevenueList } from './components/external-revenue-list'
import { PlatformOrders } from './components/platform-orders'
import { RevenueOverview } from './components/revenue-overview'
import type { RevenueSectionId } from './section-registry'

const SECTIONS: Array<{ id: RevenueSectionId; label: string }> = [
  { id: 'overview', label: 'Revenue overview' },
  { id: 'platform-orders', label: 'Platform orders' },
  { id: 'external', label: 'External revenue' },
]

export function RevenueManagement() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const params = useParams({ from: '/_authenticated/revenue/$section' })
  const section = params.section as RevenueSectionId
  let content = <RevenueOverview />
  if (section === 'platform-orders') content = <PlatformOrders />
  if (section === 'external') content = <ExternalRevenueList />

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Revenue management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <Tabs
          value={section}
          onValueChange={(value) => {
            if (!value) return
            void navigate({
              to: '/revenue/$section',
              params: { section: value as RevenueSectionId },
            })
          }}
          className='gap-4'
        >
          <div className='overflow-x-auto pb-1'>
            <TabsList variant='line'>
              {SECTIONS.map((item) => (
                <TabsTrigger key={item.id} value={item.id}>
                  {t(item.label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>
          {content}
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
