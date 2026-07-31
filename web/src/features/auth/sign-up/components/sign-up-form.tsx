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
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2 } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { z } from 'zod'

import { PasswordInput } from '@/components/password-input'
import { Turnstile } from '@/components/turnstile'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { createOAuthFlow, register } from '@/features/auth/api'
import { LegalConsent } from '@/features/auth/components/legal-consent'
import { OAuthProviders } from '@/features/auth/components/oauth-providers'
import { registerFormSchema } from '@/features/auth/constants'
import { useAuthRedirect } from '@/features/auth/hooks/use-auth-redirect'
import { useEmailVerification } from '@/features/auth/hooks/use-email-verification'
import { useTurnstile } from '@/features/auth/hooks/use-turnstile'
import {
  getAffiliateCode,
  saveAffiliateCode,
} from '@/features/auth/lib/storage'
import {
  clearWeChatRegistrationVerification,
  getWeChatRegistrationVerification,
  resolveWeChatRegistrationVerificationState,
  saveWeChatRegistrationReturnTo,
  WECHAT_REGISTRATION_RESET_CODES,
  type WeChatRegistrationVerification as StoredWeChatRegistrationVerification,
} from '@/features/auth/lib/wechat-registration-verification'
import { useStatus } from '@/hooks/use-status'
import { buildWeChatOAuthUrl } from '@/lib/oauth'
import { getServerErrorMessageKey } from '@/lib/server-error-message'
import { cn } from '@/lib/utils'

import { WeChatRegistrationVerification } from './wechat-registration-verification'

type SignUpFormProps = React.HTMLAttributes<HTMLFormElement> & {
  redirectTo?: string
}

export function SignUpForm({
  redirectTo,
  className,
  ...props
}: SignUpFormProps) {
  const { t } = useTranslation()
  const [isLoading, setIsLoading] = useState(false)
  const [verificationCode, setVerificationCode] = useState('')
  const [agreedToLegal, setAgreedToLegal] = useState(false)
  const [turnstileWidgetKey, setTurnstileWidgetKey] = useState(0)
  const [isWeChatRedirecting, setIsWeChatRedirecting] = useState(false)
  const [weChatVerification, setWeChatVerification] =
    useState<StoredWeChatRegistrationVerification | null>(() =>
      getWeChatRegistrationVerification()
    )
  const legalConsentErrorMessage = t('Please agree to the legal terms first')

  const { status } = useStatus()
  const {
    isTurnstileEnabled,
    turnstileSiteKey,
    turnstileToken,
    setTurnstileToken,
    validateTurnstile,
  } = useTurnstile()
  const { redirectToLogin } = useAuthRedirect()
  const {
    isSending: isSendingCode,
    secondsLeft,
    isActive,
    sendCode,
  } = useEmailVerification({
    turnstileToken,
    validateTurnstile,
  })

  const form = useForm<z.infer<typeof registerFormSchema>>({
    resolver: zodResolver(registerFormSchema),
    defaultValues: {
      username: '',
      email: '',
      password: '',
      confirmPassword: '',
    },
  })

  const emailValue = form.watch('email')
  const emailVerificationRequired = !!status?.email_verification
  const weChatVerificationRequired = !!status?.wechat_registration_verification
  const weChatVerificationAvailable =
    !!status?.wechat_login && !!status?.wechat_app_id
  const weChatVerificationState = resolveWeChatRegistrationVerificationState(
    weChatVerificationRequired,
    weChatVerificationAvailable,
    weChatVerification
  )
  const weChatVerificationPending =
    weChatVerificationRequired && weChatVerificationState !== 'verified'
  const hasUserAgreement = Boolean(status?.user_agreement_enabled)
  const hasPrivacyPolicy = Boolean(status?.privacy_policy_enabled)
  const requiresLegalConsent = hasUserAgreement || hasPrivacyPolicy
  const oauthRegisterEnabled =
    status?.oauth_register_enabled ??
    status?.data?.oauth_register_enabled ??
    true
  const turnstileReady = !isTurnstileEnabled || Boolean(turnstileToken)

  useEffect(() => {
    if (requiresLegalConsent) {
      setAgreedToLegal(false)
    } else {
      setAgreedToLegal(true)
    }
  }, [requiresLegalConsent])

  useEffect(() => {
    const aff = new URLSearchParams(window.location.search).get('aff')?.trim()
    if (aff) {
      saveAffiliateCode(aff)
    }
  }, [])

  useEffect(() => {
    if (!weChatVerification) return
    const expiresIn = weChatVerification.expiresAt * 1000 - Date.now()
    if (expiresIn <= 0) {
      clearWeChatRegistrationVerification()
      setWeChatVerification(null)
      return
    }
    const timeout = window.setTimeout(() => {
      clearWeChatRegistrationVerification()
      setWeChatVerification(null)
      toast.error(t('WeChat verification expired. Please verify again.'))
    }, expiresIn)
    return () => window.clearTimeout(timeout)
  }, [t, weChatVerification])

  async function onSubmit(data: z.infer<typeof registerFormSchema>) {
    if (requiresLegalConsent && !agreedToLegal) {
      toast.error(legalConsentErrorMessage)
      return
    }

    // Validate email verification if required
    if (emailVerificationRequired) {
      if (!data.email) {
        toast.error(t('Please enter your email'))
        return
      }
      if (!verificationCode) {
        toast.error(t('Please enter the verification code'))
        return
      }
    }
    if (weChatVerificationRequired && weChatVerificationState !== 'verified') {
      toast.error(t('Please complete WeChat verification first'))
      return
    }

    if (!validateTurnstile()) return

    setIsLoading(true)
    try {
      const res = await register({
        username: data.username,
        password: data.password,
        email: data.email || undefined,
        verification_code: verificationCode || undefined,
        wechat_verification_token:
          weChatVerificationRequired && weChatVerificationState === 'verified'
            ? weChatVerification?.token
            : undefined,
        aff_code: getAffiliateCode(),
        turnstile: turnstileToken,
      })

      if (res?.success) {
        clearWeChatRegistrationVerification()
        setWeChatVerification(null)
        toast.success(t('Account created! Please sign in'))
        redirectToLogin(redirectTo)
      } else {
        if (res?.code && WECHAT_REGISTRATION_RESET_CODES.has(res.code)) {
          clearWeChatRegistrationVerification()
          setWeChatVerification(null)
        }
        const messageKey = getServerErrorMessageKey(res)
        toast.error(
          messageKey
            ? t(messageKey)
            : res?.message || t('Failed to create account')
        )
      }
    } catch {
      // Errors are handled by global interceptor
    } finally {
      setIsLoading(false)
    }
  }

  async function handleWeChatVerification() {
    if (!weChatVerificationAvailable || !status?.wechat_app_id) {
      toast.error(
        t(
          'WeChat verification is required but temporarily unavailable. Please contact the administrator.'
        )
      )
      return
    }
    setIsWeChatRedirecting(true)
    try {
      saveWeChatRegistrationReturnTo(redirectTo)
      const state = await createOAuthFlow('wechat', 'register')
      window.location.assign(buildWeChatOAuthUrl(status.wechat_app_id, state))
    } catch {
      saveWeChatRegistrationReturnTo()
      toast.error(t('Failed to start WeChat verification'))
      setIsWeChatRedirecting(false)
    }
  }

  async function handleSendVerificationCode() {
    if (await sendCode(emailValue || '')) {
      setTurnstileToken('')
      setTurnstileWidgetKey((current) => current + 1)
    }
  }

  let verificationCodeAction: ReactNode = t('Send code')
  if (isActive) {
    verificationCodeAction = t('Resend ({{seconds}}s)', {
      seconds: secondsLeft,
    })
  } else if (isSendingCode) {
    verificationCodeAction = <Loader2 className='h-4 w-4 animate-spin' />
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className={cn('grid gap-4 pb-20 sm:pb-0', className)}
        {...props}
      >
        {weChatVerificationRequired && (
          <WeChatRegistrationVerification
            state={weChatVerificationState}
            emailVerificationRequired={emailVerificationRequired}
            redirecting={isWeChatRedirecting}
            onVerify={handleWeChatVerification}
          />
        )}

        {/* Username Field */}
        <FormField
          control={form.control}
          name='username'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Username')}</FormLabel>
              <FormControl>
                <Input
                  placeholder={t('Enter your username')}
                  disabled={weChatVerificationPending}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        {/* Password Field */}
        <FormField
          control={form.control}
          name='password'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Password')}</FormLabel>
              <FormControl>
                <PasswordInput
                  placeholder={t('Enter password (8-20 characters)')}
                  disabled={weChatVerificationPending}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        {/* Confirm Password Field */}
        <FormField
          control={form.control}
          name='confirmPassword'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Confirm password')}</FormLabel>
              <FormControl>
                <PasswordInput
                  placeholder={t('Confirm password')}
                  disabled={weChatVerificationPending}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        {/* Email Verification Section */}
        {emailVerificationRequired && (
          <>
            {/* Email Field */}
            <FormField
              control={form.control}
              name='email'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Email (required for verification)')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('name@example.com')}
                      type='email'
                      disabled={weChatVerificationPending}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Verification Code Field */}
            <div className='flex items-end gap-2'>
              <div className='flex-1'>
                <Input
                  placeholder={t('Verification code')}
                  value={verificationCode}
                  disabled={weChatVerificationPending}
                  onChange={(e) => setVerificationCode(e.target.value)}
                />
              </div>
              <Button
                variant='outline'
                type='button'
                disabled={
                  isLoading ||
                  weChatVerificationPending ||
                  isSendingCode ||
                  isActive ||
                  !emailValue ||
                  !turnstileReady
                }
                onClick={handleSendVerificationCode}
              >
                {verificationCodeAction}
              </Button>
            </div>
            {weChatVerificationRequired && (
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Email verification is also required. Your account is created only after both checks pass.'
                )}
              </p>
            )}
          </>
        )}

        {/* Turnstile */}
        {isTurnstileEnabled && (
          <div className='mt-2'>
            <Turnstile
              key={turnstileWidgetKey}
              siteKey={turnstileSiteKey}
              onVerify={setTurnstileToken}
            />
          </div>
        )}

        <LegalConsent
          status={status}
          checked={agreedToLegal}
          onCheckedChange={setAgreedToLegal}
          className='mt-1'
        />

        {/* Submit Button */}
        <Button
          type='submit'
          className='mt-2 w-full justify-center gap-2'
          disabled={
            isLoading ||
            (requiresLegalConsent && !agreedToLegal) ||
            !turnstileReady ||
            (weChatVerificationRequired &&
              weChatVerificationState !== 'verified')
          }
        >
          {isLoading ? <Loader2 className='h-4 w-4 animate-spin' /> : null}
          {t('Create account')}
        </Button>

        {oauthRegisterEnabled && !weChatVerificationRequired && (
          <OAuthProviders
            status={status}
            redirectTo={redirectTo}
            disabled={isLoading || (requiresLegalConsent && !agreedToLegal)}
            className='pt-2'
          />
        )}
      </form>
    </Form>
  )
}
