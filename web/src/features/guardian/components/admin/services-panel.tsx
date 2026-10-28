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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import { toIntlLocale } from '@/i18n/languages'

import { listAdminGuardianServices } from '../../api'
import { GUARDIAN_QUERY_KEYS } from '../../constants'
import { useGuardianLabels } from '../../hooks/use-guardian-labels'
import { formatCny } from '../../lib/price'
import type { GuardianService } from '../../types'
import { GuardianServiceDialog } from './service-dialog'

export function GuardianServicesPanel() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useGuardianLabels()
  const queryClient = useQueryClient()
  const [editor, setEditor] = useState<GuardianService | 'new' | null>(null)
  const services = useQuery({
    queryKey: GUARDIAN_QUERY_KEYS.adminServices,
    queryFn: listAdminGuardianServices,
  })
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['guardian'] })
  }

  return (
    <TitledCard
      title={t('Service catalog')}
      description={t(
        'Names, specifications, copy and prices shown on the Guardian page. Unlisted services stay attached to their past orders.'
      )}
      disableHoverEffect
      action={
        <Button onClick={() => setEditor('new')}>{t('New service')}</Button>
      }
    >
      {services.isPending && <LoadingState />}
      {services.isError && (
        <ErrorState
          title={t('Failed to load services')}
          onRetry={() => void services.refetch()}
        />
      )}
      {services.isSuccess && services.data.length === 0 && (
        <EmptyState
          title={t('No services yet')}
          description={t(
            'Create the first service to show it on the Guardian page.'
          )}
          className='min-h-40'
        />
      )}
      {services.isSuccess && services.data.length > 0 && (
        <StaticDataTable tableClassName='min-w-[720px]'>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Service name')}</TableHead>
              <TableHead>{t('Price')}</TableHead>
              <TableHead>{t('Route')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Sort order')}</TableHead>
              <TableHead>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {services.data.map((service) => (
              <TableRow key={service.id}>
                <TableCell className='max-w-72 whitespace-normal'>
                  <div className='font-medium break-words'>{service.title}</div>
                  {service.subtitle && (
                    <div className='text-muted-foreground mt-1 text-xs break-words'>
                      {service.subtitle}
                    </div>
                  )}
                </TableCell>
                <TableCell>{formatCny(service.price_cents, locale)}</TableCell>
                <TableCell>{labels.serviceRoute[service.route]}</TableCell>
                <TableCell>
                  <Badge variant={service.enabled ? 'secondary' : 'outline'}>
                    {service.enabled ? t('Listed') : t('Unlisted')}
                  </Badge>
                </TableCell>
                <TableCell>{service.sort_order}</TableCell>
                <TableCell>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => setEditor(service)}
                  >
                    {t('Edit')}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </StaticDataTable>
      )}
      {editor && (
        <GuardianServiceDialog
          service={editor === 'new' ? undefined : editor}
          onClose={() => setEditor(null)}
          onSaved={refresh}
        />
      )}
    </TitledCard>
  )
}
