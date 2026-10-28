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
import { Link } from '@tanstack/react-router'
import { ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { GuardianOrdersPanel } from './components/admin/orders-panel'
import { GuardianServicesPanel } from './components/admin/services-panel'

export function GuardianAdmin() {
  const { t } = useTranslation()

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Guardian services')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          className='gap-1.5'
          render={<Link to='/guardian' target='_blank' />}
        >
          {t('View page')}
          <ArrowUpRight aria-hidden='true' />
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs defaultValue='orders'>
          <TabsList>
            <TabsTrigger value='orders'>{t('Orders')}</TabsTrigger>
            <TabsTrigger value='services'>{t('Service catalog')}</TabsTrigger>
          </TabsList>
          <TabsContent value='orders' className='mt-4'>
            <GuardianOrdersPanel />
          </TabsContent>
          <TabsContent value='services' className='mt-4'>
            <GuardianServicesPanel />
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
