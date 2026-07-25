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

import { useMutation, useQuery } from '@tanstack/react-query'
import { Check, Clock3, Loader2, ShieldCheck, X } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'

import {
  decideDesktopAuthorization,
  getDesktopAuthorizationRequest,
} from './api'
import { isSafeDesktopLoopbackRedirect } from './lib/loopback-redirect'
import { PccAgentLogo } from './pcc-agent-logo'

interface DesktopAuthorizationScreenProps {
  requestToken: string
}

const permissionLabels: Record<string, string> = {
  relay: 'Use supported AI models through your account',
  'account.read': 'Read your account balance and subscription status',
  'usage.read': 'Read your usage history',
}

export function DesktopAuthorizationScreen(
  props: DesktopAuthorizationScreenProps
) {
  const { t } = useTranslation()
  const requestQuery = useQuery({
    queryKey: ['desktop-authorization', props.requestToken],
    queryFn: () => getDesktopAuthorizationRequest(props.requestToken),
    retry: false,
  })
  const decisionMutation = useMutation({
    mutationFn: async (decision: 'allow' | 'deny') => {
      const result = await decideDesktopAuthorization(
        props.requestToken,
        decision
      )
      if (!isSafeDesktopLoopbackRedirect(result.redirect_uri)) {
        throw new Error('DESKTOP_INVALID_REDIRECT_URI')
      }
      window.location.assign(result.redirect_uri)
    },
  })

  let content: ReactNode
  if (requestQuery.isLoading) {
    content = (
      <div
        className='space-y-4'
        aria-label={t('Loading authorization request')}
      >
        <Skeleton className='h-20 w-full' />
        <Skeleton className='h-36 w-full' />
        <Skeleton className='h-10 w-full' />
      </div>
    )
  } else if (requestQuery.isError || !requestQuery.data) {
    content = (
      <div className='space-y-4 py-4 text-center'>
        <div className='bg-destructive/10 text-destructive mx-auto flex size-10 items-center justify-center rounded-lg'>
          <X className='size-5' aria-hidden='true' />
        </div>
        <div className='space-y-1'>
          <h2 className='text-base font-medium'>
            {t('This authorization request is unavailable')}
          </h2>
          <p className='text-muted-foreground text-sm'>
            {t('Return to PCC Agent and start a new browser authorization.')}
          </p>
        </div>
      </div>
    )
  } else {
    const request = requestQuery.data
    const tokenDays = Math.round(request.token_ttl / 86_400)
    content = (
      <div className='space-y-5'>
        <div className='bg-muted/40 flex min-w-0 items-center gap-3 rounded-lg border p-3'>
          <PccAgentLogo className='size-10 rounded-lg border shadow-sm' />
          <div className='min-w-0'>
            <p className='truncate text-sm font-medium'>
              {request.device_name}
            </p>
            <p className='text-muted-foreground truncate text-xs'>
              {[request.platform, request.app_version]
                .filter(Boolean)
                .join(' · ')}
            </p>
          </div>
        </div>

        <section
          aria-labelledby='desktop-permissions-title'
          className='space-y-3'
        >
          <h2 id='desktop-permissions-title' className='text-sm font-medium'>
            {t('PCC Agent will be able to')}
          </h2>
          <ul className='space-y-2'>
            {request.scopes.map((scope) => (
              <li key={scope} className='flex items-start gap-2 text-sm'>
                <Check
                  className='text-primary mt-0.5 size-4 shrink-0'
                  aria-hidden='true'
                />
                <span>{t(permissionLabels[scope] ?? scope)}</span>
              </li>
            ))}
          </ul>
        </section>

        <section aria-labelledby='desktop-models-title' className='space-y-2'>
          <h2 id='desktop-models-title' className='text-sm font-medium'>
            {t('Allowed models')}
          </h2>
          {request.allowed_models.length > 0 ? (
            <div className='flex flex-wrap gap-1.5'>
              {request.allowed_models.slice(0, 8).map((model) => (
                <Badge key={model} variant='outline'>
                  {model}
                </Badge>
              ))}
              {request.allowed_models.length > 8 && (
                <Badge variant='secondary'>
                  {t('{{count}} more', {
                    count: request.allowed_models.length - 8,
                  })}
                </Badge>
              )}
            </div>
          ) : (
            <p className='text-muted-foreground text-sm'>
              {t('No models are currently available for this account.')}
            </p>
          )}
        </section>

        <div className='text-muted-foreground flex items-start gap-2 border-t pt-4 text-xs'>
          <Clock3 className='mt-0.5 size-4 shrink-0' aria-hidden='true' />
          <p>
            {t('This device authorization expires in {{count}} days.', {
              count: tokenDays,
            })}
          </p>
        </div>

        {decisionMutation.isError && (
          <p className='text-destructive text-sm' role='alert'>
            {t('Authorization could not be completed. Please try again.')}
          </p>
        )}

        <div className='flex flex-col-reverse gap-2 sm:flex-row sm:justify-end'>
          <Button
            type='button'
            variant='outline'
            disabled={decisionMutation.isPending}
            onClick={() => decisionMutation.mutate('deny')}
          >
            <X aria-hidden='true' />
            {t('Deny')}
          </Button>
          <Button
            type='button'
            disabled={
              decisionMutation.isPending || request.allowed_models.length === 0
            }
            onClick={() => decisionMutation.mutate('allow')}
          >
            {decisionMutation.isPending ? (
              <Loader2 className='animate-spin' aria-hidden='true' />
            ) : (
              <ShieldCheck aria-hidden='true' />
            )}
            {t('Allow')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <Main>
      <div className='min-h-0 flex-1 overflow-auto px-3 py-6 sm:px-6 sm:py-10'>
        <TitledCard
          className='mx-auto max-w-2xl'
          title={t('Authorize PCC Agent')}
          description={t(
            'Review this device and the account access it is requesting.'
          )}
          icon={<PccAgentLogo className='size-full rounded-lg' />}
          iconClassName='bg-transparent p-0'
          disableHoverEffect
        >
          {content}
        </TitledCard>
      </div>
    </Main>
  )
}
