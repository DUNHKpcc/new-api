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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { Laptop, Loader2, Unplug } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import {
  getDesktopGrants,
  revokeDesktopGrant,
} from '@/features/desktop-authorization/api'
import { PccAgentLogo } from '@/features/desktop-authorization/pcc-agent-logo'
import type { DesktopGrant } from '@/features/desktop-authorization/types'

const desktopGrantQueryKey = ['profile', 'desktop-grants'] as const

function grantStatusLabel(status: DesktopGrant['status']): string {
  switch (status) {
    case 'active':
      return 'Active'
    case 'expired':
      return 'Expired'
    default:
      return 'Revoked'
  }
}

export function DesktopGrantsCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [revokeTarget, setRevokeTarget] = useState<DesktopGrant | null>(null)
  const grantsQuery = useQuery({
    queryKey: desktopGrantQueryKey,
    queryFn: async () => {
      const response = await getDesktopGrants()
      if (!response.success) {
        throw new Error(
          response.message || t('Failed to load authorized devices')
        )
      }
      return response.data ?? []
    },
  })
  const revokeMutation = useMutation({
    mutationFn: async (publicID: string) => {
      const response = await revokeDesktopGrant(publicID)
      if (!response.success) {
        throw new Error(response.message || t('Failed to revoke device'))
      }
    },
    onSuccess: async () => {
      setRevokeTarget(null)
      toast.success(t('Device authorization revoked'))
      await queryClient.invalidateQueries({ queryKey: desktopGrantQueryKey })
    },
    onError: () => toast.error(t('Failed to revoke device')),
  })

  let content: ReactNode
  if (grantsQuery.isLoading) {
    content = (
      <div className='space-y-3'>
        <Skeleton className='h-20 w-full' />
        <Skeleton className='h-20 w-full' />
      </div>
    )
  } else if (grantsQuery.isError) {
    content = (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Laptop aria-hidden='true' />
          </EmptyMedia>
          <EmptyTitle>{t('Unable to load authorized devices')}</EmptyTitle>
          <EmptyDescription>
            {t('Refresh the list and try again.')}
          </EmptyDescription>
        </EmptyHeader>
        <Button
          type='button'
          variant='outline'
          onClick={() => grantsQuery.refetch()}
        >
          {t('Retry')}
        </Button>
      </Empty>
    )
  } else if (!grantsQuery.data || grantsQuery.data.length === 0) {
    content = (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Laptop aria-hidden='true' />
          </EmptyMedia>
          <EmptyTitle>{t('No authorized desktop devices')}</EmptyTitle>
          <EmptyDescription>
            {t('Devices you authorize for PCC Agent will appear here.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  } else {
    content = (
      <div className='flex flex-col'>
        {grantsQuery.data.map((grant, index) => (
          <div key={grant.public_id}>
            {index > 0 && <Separator />}
            <div className='flex min-w-0 items-center gap-3 py-3'>
              <PccAgentLogo className='size-9 rounded-lg shadow-sm' />
              <div className='min-w-0 flex-1'>
                <div className='flex min-w-0 flex-wrap items-center gap-2'>
                  <p className='truncate text-sm font-medium'>
                    {grant.device_name}
                  </p>
                  <Badge
                    variant={
                      grant.status === 'active' ? 'secondary' : 'outline'
                    }
                  >
                    {t(grantStatusLabel(grant.status))}
                  </Badge>
                </div>
                <p className='text-muted-foreground truncate text-xs'>
                  {[grant.platform, grant.app_version]
                    .filter(Boolean)
                    .join(' · ')}
                </p>
                <p className='text-muted-foreground mt-1 text-xs'>
                  {t('Authorized {{time}}', {
                    time: dayjs
                      .unix(grant.created_time)
                      .format('YYYY-MM-DD HH:mm'),
                  })}
                </p>
                <div className='text-muted-foreground mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                  <span>
                    {t('Last used:')}{' '}
                    {dayjs
                      .unix(grant.last_used_time)
                      .format('YYYY-MM-DD HH:mm')}
                  </span>
                  <span>
                    {t('Expires at')}{' '}
                    {dayjs.unix(grant.expired_time).format('YYYY-MM-DD HH:mm')}
                  </span>
                </div>
              </div>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Revoke device authorization')}
                title={t('Revoke device authorization')}
                disabled={grant.status !== 'active'}
                onClick={() => setRevokeTarget(grant)}
              >
                <Unplug aria-hidden='true' />
              </Button>
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <>
      <TitledCard
        title={t('Authorized desktop devices')}
        description={t(
          'Review and revoke PCC Agent devices connected to your account.'
        )}
        icon={<PccAgentLogo className='size-full rounded-lg' />}
        iconClassName='bg-transparent p-0'
        disableHoverEffect
      >
        {content}
      </TitledCard>

      <AlertDialog
        open={revokeTarget !== null}
        onOpenChange={(open) => {
          if (!open) setRevokeTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <Unplug aria-hidden='true' />
            </AlertDialogMedia>
            <AlertDialogTitle>
              {t('Revoke this device authorization?')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'PCC Agent on this device will immediately lose access to your account and models.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokeMutation.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={revokeMutation.isPending}
              onClick={() => {
                if (revokeTarget) {
                  revokeMutation.mutate(revokeTarget.public_id)
                }
              }}
            >
              {revokeMutation.isPending ? (
                <Loader2 className='animate-spin' aria-hidden='true' />
              ) : (
                <Unplug aria-hidden='true' />
              )}
              {t('Revoke')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
