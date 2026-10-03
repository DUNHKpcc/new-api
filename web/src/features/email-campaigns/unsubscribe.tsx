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
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { PublicLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { TitledCard } from '@/components/ui/titled-card'

import { confirmUnsubscribe, getUnsubscribeInfo } from './api'
import { EmailCategoryLabel } from './components/email-status'

export function EmailUnsubscribe(props: { token: string }) {
  const { t } = useTranslation()
  const info = useQuery({
    queryKey: ['email-unsubscribe', props.token],
    queryFn: () => getUnsubscribeInfo(props.token),
    enabled: !!props.token,
    retry: false,
    gcTime: 0,
    refetchOnWindowFocus: false,
    meta: { errorToast: false },
  })
  const unsubscribe = useMutation({
    mutationFn: () => confirmUnsubscribe(props.token),
    retry: false,
    meta: { errorToast: false },
  })
  useEffect(() => {
    const meta = document.createElement('meta')
    meta.name = 'referrer'
    meta.content = 'no-referrer'
    document.head.append(meta)
    return () => meta.remove()
  }, [])
  const done = info.data?.unsubscribed || unsubscribe.data?.unsubscribed
  return (
    <PublicLayout
      showAuthButtons={false}
      showNotifications={false}
      navLinks={[]}
      headerProps={{ showNavigation: false }}
    >
      <div className='mx-auto max-w-lg py-8'>
        <TitledCard title={t('Email unsubscribe')} disableHoverEffect>
          {!props.token && (
            <ErrorState
              title={t('Invalid unsubscribe link')}
              description={t(
                'Open the unsubscribe link from an email you received.'
              )}
            />
          )}
          {props.token && info.isPending && <LoadingState />}
          {info.isError && (
            <ErrorState
              title={t('Unable to open this unsubscribe link')}
              description={t(
                'The link may be invalid. Try opening it again from your email.'
              )}
              onRetry={() => void info.refetch()}
            />
          )}
          {info.data && (
            <div className='space-y-4'>
              <p className='font-medium'>
                <EmailCategoryLabel category={info.data.category} />
              </p>
              {done ? (
                <p role='status'>
                  {t(
                    'You are unsubscribed from these emails. You can change your preferences in your profile.'
                  )}
                </p>
              ) : (
                <>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Confirm to stop receiving this category of email. Security and verification emails are unaffected.'
                    )}
                  </p>
                  <Button
                    onClick={() => unsubscribe.mutate()}
                    disabled={unsubscribe.isPending}
                  >
                    {unsubscribe.isPending
                      ? t('Saving...')
                      : t('Confirm unsubscribe')}
                  </Button>
                </>
              )}
              {unsubscribe.isError && (
                <p role='alert' className='text-destructive text-sm'>
                  {t('Unsubscribe failed. Please try again.')}
                </p>
              )}
            </div>
          )}
        </TitledCard>
      </div>
    </PublicLayout>
  )
}
