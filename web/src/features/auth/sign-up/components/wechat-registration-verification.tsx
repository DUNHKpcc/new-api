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

import { CircleAlert, CircleCheck, Loader2, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconWeChat } from '@/assets/brand-icons'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import type { WeChatRegistrationVerificationState } from '@/features/auth/lib/wechat-registration-verification'

type WeChatRegistrationVerificationProps = {
  state: WeChatRegistrationVerificationState
  emailVerificationRequired: boolean
  redirecting: boolean
  onVerify: () => void
}

export function WeChatRegistrationVerification(
  props: WeChatRegistrationVerificationProps
) {
  const { t } = useTranslation()

  if (props.state === 'not-required') return null

  if (props.state === 'unavailable') {
    return (
      <Alert variant='destructive'>
        <CircleAlert aria-hidden='true' />
        <AlertTitle>{t('WeChat verification unavailable')}</AlertTitle>
        <AlertDescription>
          {t(
            'WeChat verification is required but temporarily unavailable. Please contact the administrator.'
          )}
        </AlertDescription>
      </Alert>
    )
  }

  if (props.state === 'verified') {
    return (
      <Alert className='border-emerald-200 bg-emerald-50/70 dark:border-emerald-900/60 dark:bg-emerald-950/30'>
        <CircleCheck
          className='text-emerald-600 dark:text-emerald-400'
          aria-hidden='true'
        />
        <AlertTitle>{t('WeChat verification completed')}</AlertTitle>
        <AlertDescription className='space-y-3'>
          <p>
            {t(
              'Your WeChat identity is verified. Finish registration within 10 minutes before this verification expires.'
            )}
          </p>
          <p>
            {t(
              'The verified WeChat account will be linked to the new account and can be used to sign in later.'
            )}
          </p>
          {props.emailVerificationRequired && (
            <p>
              {t(
                'Email verification is also required. Your account is created only after both checks pass.'
              )}
            </p>
          )}
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={props.redirecting}
            onClick={props.onVerify}
            className='w-full justify-center gap-2'
          >
            {props.redirecting ? (
              <Loader2 className='size-4 animate-spin' aria-hidden='true' />
            ) : (
              <RefreshCw className='size-4' aria-hidden='true' />
            )}
            {t('Verify again')}
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <Alert className='border-sky-200 bg-sky-50/70 dark:border-sky-900/60 dark:bg-sky-950/30'>
      <IconWeChat className='size-4 text-[#07C160]' aria-hidden='true' />
      <AlertTitle>
        {props.emailVerificationRequired
          ? t('Two verification steps required')
          : t('WeChat verification required')}
      </AlertTitle>
      <AlertDescription className='space-y-3'>
        <p>
          {props.emailVerificationRequired
            ? t(
                'First verify with WeChat, then enter the email code below. Your account is created only after both checks pass.'
              )
            : t('Verify with WeChat before creating your account.')}
        </p>
        <p>
          {t(
            'The verified WeChat account will be linked to the new account and can be used to sign in later.'
          )}
        </p>
        <Button
          type='button'
          variant='outline'
          disabled={props.redirecting}
          onClick={props.onVerify}
          className='w-full justify-center gap-2'
        >
          {props.redirecting ? (
            <Loader2 className='size-4 animate-spin' aria-hidden='true' />
          ) : (
            <IconWeChat className='size-4' aria-hidden='true' />
          )}
          {props.redirecting
            ? t('Redirecting to WeChat...')
            : t('Verify with WeChat')}
        </Button>
      </AlertDescription>
    </Alert>
  )
}
