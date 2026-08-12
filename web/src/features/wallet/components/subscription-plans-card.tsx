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
import {
  Check,
  ChevronsUpDown,
  Crown,
  Layers3,
  RefreshCw,
  Sparkles,
} from 'lucide-react'
import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  StatusBadge,
  dotColorMap,
  textColorMap,
} from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
  updateBillingPreference,
} from '@/features/subscriptions/api'
import { SubscriptionPurchaseDialog } from '@/features/subscriptions/components/dialogs/subscription-purchase-dialog'
import {
  formatDuration,
  formatResetPeriod,
  formatSubscriptionPrice,
} from '@/features/subscriptions/lib'
import type {
  PlanRecord,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { PaymentMethod, TopupInfo } from '../types'

interface SubscriptionPlansCardProps {
  topupInfo: TopupInfo | null
  onAvailabilityChange?: (available: boolean) => void
  userQuota?: number
  onPurchaseSuccess?: () => void | Promise<void>
}

function getEpayMethods(payMethods: PaymentMethod[] = []): PaymentMethod[] {
  return payMethods.filter(
    (m) => m?.type && m.type !== 'stripe' && m.type !== 'creem'
  )
}

function getBillingPreferenceLabel(
  preference: string,
  t: (key: string) => string
): string {
  switch (preference) {
    case 'subscription_first':
      return t('Subscription First')
    case 'wallet_first':
      return t('Wallet First')
    case 'subscription_only':
      return t('Subscription Only')
    case 'wallet_only':
      return t('Wallet Only')
    default:
      return preference
  }
}

export function SubscriptionPlansCard({
  topupInfo,
  onAvailabilityChange,
  userQuota,
  onPurchaseSuccess,
}: SubscriptionPlansCardProps) {
  const { t } = useTranslation()

  const [plans, setPlans] = useState<PlanRecord[]>([])
  const [activeSubscriptions, setActiveSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [allSubscriptions, setAllSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [billingPreference, setBillingPreference] =
    useState('subscription_first')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const [purchaseOpen, setPurchaseOpen] = useState(false)
  const [selectedPlan, setSelectedPlan] = useState<PlanRecord | null>(null)

  const enableStripe = !!topupInfo?.enable_stripe_topup
  const enableCreem = !!topupInfo?.enable_creem_topup
  const enableWaffoPancake = !!topupInfo?.enable_waffo_pancake_topup
  const enableOnlineTopUp = !!topupInfo?.enable_online_topup
  const epayMethods = useMemo(
    () => getEpayMethods(topupInfo?.pay_methods),
    [topupInfo?.pay_methods]
  )

  const fetchPlans = useCallback(async () => {
    try {
      const res = await getPublicPlans()
      if (res.success) {
        setPlans(res.data || [])
      }
    } catch {
      setPlans([])
    }
  }, [])

  const fetchSelfSubscription = useCallback(async () => {
    try {
      const res = await getSelfSubscriptionFull()
      if (res.success && res.data) {
        setBillingPreference(
          res.data.billing_preference || 'subscription_first'
        )
        setActiveSubscriptions(res.data.subscriptions || [])
        setAllSubscriptions(res.data.all_subscriptions || [])
      }
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    const init = async () => {
      setLoading(true)
      await Promise.all([fetchPlans(), fetchSelfSubscription()])
      setLoading(false)
    }
    init()
  }, [fetchPlans, fetchSelfSubscription])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await fetchSelfSubscription()
    } finally {
      setRefreshing(false)
    }
  }

  const handlePreferenceChange = async (pref: string) => {
    const previous = billingPreference
    setBillingPreference(pref)
    try {
      const res = await updateBillingPreference(pref)
      if (res.success) {
        toast.success(t('Updated successfully'))
        const normalized = res.data?.billing_preference || pref
        setBillingPreference(normalized)
      } else {
        toast.error(res.message || t('Update failed'))
        setBillingPreference(previous)
      }
    } catch {
      toast.error(t('Request failed'))
      setBillingPreference(previous)
    }
  }

  const hasActive = activeSubscriptions.length > 0
  const hasAny = allSubscriptions.length > 0
  const isAvailable = loading || plans.length > 0 || hasAny
  const disablePref = !hasActive
  const isSubPref =
    billingPreference === 'subscription_first' ||
    billingPreference === 'subscription_only'
  const displayPref =
    disablePref && isSubPref ? 'wallet_first' : billingPreference

  const planPurchaseCountMap = useMemo(() => {
    const map = new Map<number, number>()
    for (const sub of allSubscriptions) {
      const planId = sub?.subscription?.plan_id
      if (!planId) continue
      map.set(planId, (map.get(planId) || 0) + 1)
    }
    return map
  }, [allSubscriptions])

  useEffect(() => {
    onAvailabilityChange?.(isAvailable)
  }, [isAvailable, onAvailabilityChange])

  const planTitleMap = useMemo(() => {
    const map = new Map<number, string>()
    for (const p of plans) {
      if (p?.plan?.id) {
        map.set(p.plan.id, p.plan.title || '')
      }
    }
    return map
  }, [plans])

  const orderedSubscriptions = useMemo(() => {
    const now = Date.now() / 1000
    return allSubscriptions
      .map((record, index) => ({ record, index }))
      .sort((left, right) => {
        const leftSubscription = left.record.subscription
        const rightSubscription = right.record.subscription
        const leftActive =
          leftSubscription.status === 'active' &&
          leftSubscription.end_time >= now
        const rightActive =
          rightSubscription.status === 'active' &&
          rightSubscription.end_time >= now

        if (leftActive !== rightActive) return leftActive ? -1 : 1
        if (leftSubscription.end_time !== rightSubscription.end_time) {
          return rightSubscription.end_time - leftSubscription.end_time
        }
        return left.index - right.index
      })
      .map(({ record }) => record)
  }, [allSubscriptions])

  const getRemainingDays = (sub: UserSubscriptionRecord) => {
    const endTime = sub?.subscription?.end_time || 0
    if (!endTime) return 0
    const now = Date.now() / 1000
    return Math.max(0, Math.ceil((endTime - now) / 86400))
  }

  const getUsagePercent = (sub: UserSubscriptionRecord) => {
    const total = Number(sub?.subscription?.amount_total || 0)
    const used = Number(sub?.subscription?.amount_used || 0)
    if (total <= 0) return 0
    return Math.round((used / total) * 100)
  }

  if (loading) {
    return (
      <>
        <Card
          data-card-hover='false'
          className='gap-0 overflow-hidden py-0 xl:col-start-1 xl:row-start-2'
        >
          <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
            <Skeleton className='h-6 w-32' />
          </CardHeader>
          <CardContent className='p-3 sm:p-5'>
            <Skeleton className='h-40 w-full' />
          </CardContent>
        </Card>
        <Card
          data-card-hover='false'
          className='gap-0 overflow-hidden py-0 xl:col-start-2 xl:row-span-2 xl:row-start-1 xl:h-full xl:min-h-0 xl:self-stretch xl:[contain:size]'
        >
          <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
            <Skeleton className='h-6 w-32' />
          </CardHeader>
          <CardContent className='p-3 sm:p-5'>
            <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
              {['first', 'second', 'third', 'fourth'].map((key) => (
                <Skeleton key={key} className='h-64 w-full' />
              ))}
            </div>
          </CardContent>
        </Card>
      </>
    )
  }

  if (plans.length === 0 && !hasAny) {
    return null
  }

  return (
    <>
      <TitledCard
        title={t('My Subscriptions')}
        description={
          <span className='flex items-center gap-1.5 font-medium'>
            <span
              className={cn(
                'size-1.5 shrink-0 rounded-full',
                hasActive ? dotColorMap.success : dotColorMap.neutral
              )}
              aria-hidden='true'
            />
            {hasActive ? (
              <span className={cn(textColorMap.success)}>
                {activeSubscriptions.length} {t('active')}
              </span>
            ) : (
              <span>{t('No Active')}</span>
            )}
            {allSubscriptions.length > activeSubscriptions.length && (
              <>
                <span className='text-muted-foreground/30'>·</span>
                <span>
                  {allSubscriptions.length - activeSubscriptions.length}{' '}
                  {t('expired')}
                </span>
              </>
            )}
          </span>
        }
        icon={<Layers3 className='h-4 w-4' />}
        iconTone='info'
        action={
          <div className='flex w-full items-center gap-2 sm:w-auto'>
            <Select
              items={[
                {
                  value: 'subscription_first',
                  label: (
                    <>
                      {getBillingPreferenceLabel('subscription_first', t)}
                      {disablePref ? ` (${t('No Active')})` : ''}
                    </>
                  ),
                },
                {
                  value: 'wallet_first',
                  label: getBillingPreferenceLabel('wallet_first', t),
                },
                {
                  value: 'subscription_only',
                  label: (
                    <>
                      {getBillingPreferenceLabel('subscription_only', t)}
                      {disablePref ? ` (${t('No Active')})` : ''}
                    </>
                  ),
                },
                {
                  value: 'wallet_only',
                  label: getBillingPreferenceLabel('wallet_only', t),
                },
              ]}
              value={displayPref}
              onValueChange={(v) => v !== null && handlePreferenceChange(v)}
            >
              <SelectTrigger className='h-8 flex-1 text-xs sm:w-[140px] sm:flex-none'>
                <SelectValue>
                  {getBillingPreferenceLabel(displayPref, t)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value='subscription_first' disabled={disablePref}>
                    {getBillingPreferenceLabel('subscription_first', t)}
                    {disablePref ? ` (${t('No Active')})` : ''}
                  </SelectItem>
                  <SelectItem value='wallet_first'>
                    {getBillingPreferenceLabel('wallet_first', t)}
                  </SelectItem>
                  <SelectItem value='subscription_only' disabled={disablePref}>
                    {getBillingPreferenceLabel('subscription_only', t)}
                    {disablePref ? ` (${t('No Active')})` : ''}
                  </SelectItem>
                  <SelectItem value='wallet_only'>
                    {getBillingPreferenceLabel('wallet_only', t)}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Button
              variant='ghost'
              size='icon'
              className='h-8 w-8'
              onClick={handleRefresh}
              disabled={refreshing}
              aria-label={t('Refresh')}
              title={t('Refresh')}
            >
              <RefreshCw
                className={`h-3.5 w-3.5 ${refreshing ? 'animate-spin' : ''}`}
              />
            </Button>
          </div>
        }
        disableHoverEffect
        className='xl:col-start-1 xl:row-start-2'
        contentClassName='space-y-3'
      >
        {disablePref && isSubPref && (
          <p className='text-muted-foreground text-xs'>
            {t(
              'Preference saved as {{pref}}, but no active subscription. Wallet will be used automatically.',
              {
                pref:
                  billingPreference === 'subscription_only'
                    ? t('Subscription Only')
                    : t('Subscription First'),
              }
            )}
          </p>
        )}

        {hasAny && (
          <>
            <Separator />
            <div
              role='region'
              aria-label={t('My Subscriptions')}
              tabIndex={0}
              className='max-h-[12.5rem] [scrollbar-gutter:stable] overflow-y-auto overscroll-contain pr-2 pb-1 sm:max-h-[10.5rem]'
              data-subscription-record-list='true'
            >
              <div className='space-y-2'>
                {orderedSubscriptions.map((sub) => {
                  const subscription = sub.subscription
                  const totalAmount = Number(subscription?.amount_total || 0)
                  const usedAmount = Number(subscription?.amount_used || 0)
                  const remainAmount =
                    totalAmount > 0 ? Math.max(0, totalAmount - usedAmount) : 0
                  const planTitle =
                    planTitleMap.get(subscription?.plan_id) || ''
                  const remainDays = getRemainingDays(sub)
                  const usagePercent = getUsagePercent(sub)
                  const now = Date.now() / 1000
                  const isExpired = (subscription?.end_time || 0) < now
                  const isCancelled = subscription?.status === 'cancelled'
                  const isActive =
                    subscription?.status === 'active' && !isExpired
                  const nextResetTime = subscription?.next_reset_time ?? 0
                  let statusBadge = (
                    <StatusBadge
                      label={t('Expired')}
                      variant='neutral'
                      copyable={false}
                    />
                  )
                  if (isActive) {
                    statusBadge = (
                      <StatusBadge
                        label={t('Active')}
                        variant='success'
                        copyable={false}
                      />
                    )
                  } else if (isCancelled) {
                    statusBadge = (
                      <StatusBadge
                        label={t('Cancelled')}
                        variant='neutral'
                        copyable={false}
                      />
                    )
                  }

                  let endTimeLabel = t('Expired at')
                  if (isActive) {
                    endTimeLabel = t('Until')
                  } else if (isCancelled) {
                    endTimeLabel = t('Cancelled at')
                  }

                  return (
                    <div
                      key={subscription?.id}
                      data-subscription-record-id={subscription?.id}
                      className='bg-background min-w-0 rounded-md border p-3 text-xs'
                    >
                      <div className='flex flex-wrap items-start justify-between gap-2'>
                        <div className='flex min-w-0 flex-wrap items-center gap-2'>
                          <span className='font-medium break-words'>
                            {planTitle
                              ? `${planTitle} · ${t('Subscription')} #${subscription?.id}`
                              : `${t('Subscription')} #${subscription?.id}`}
                          </span>
                          {statusBadge}
                        </div>
                        {isActive && (
                          <span className='text-muted-foreground shrink-0'>
                            {t('{{count}} days remaining', {
                              count: remainDays,
                            })}
                          </span>
                        )}
                      </div>
                      <div className='text-muted-foreground mt-1.5'>
                        {endTimeLabel}{' '}
                        {new Date(
                          (subscription?.end_time || 0) * 1000
                        ).toLocaleString()}
                      </div>
                      {isActive && nextResetTime > 0 && (
                        <div className='text-muted-foreground mt-1'>
                          {t('Next reset')}:{' '}
                          {new Date(nextResetTime * 1000).toLocaleString()}
                        </div>
                      )}
                      <div className='text-muted-foreground mt-1'>
                        {t('Total Quota')}:{' '}
                        {totalAmount > 0 ? (
                          <Tooltip>
                            <TooltipTrigger
                              render={<span className='cursor-help' />}
                            >
                              {formatQuota(usedAmount)}/
                              {formatQuota(totalAmount)} · {t('Remaining')}{' '}
                              {formatQuota(remainAmount)}
                            </TooltipTrigger>
                            <TooltipContent>
                              {t('Raw Quota')}: {usedAmount}/{totalAmount} ·{' '}
                              {t('Remaining')} {remainAmount}
                            </TooltipContent>
                          </Tooltip>
                        ) : (
                          t('Unlimited')
                        )}
                        {totalAmount > 0 && (
                          <span className='ml-2'>
                            {t('Used')} {usagePercent}%
                          </span>
                        )}
                      </div>
                      {totalAmount > 0 && isActive && (
                        <Progress value={usagePercent} className='mt-2 h-1.5' />
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          </>
        )}

        {!hasAny && (
          <p className='text-muted-foreground text-xs'>
            {t('Subscribe to a plan for model access')}
          </p>
        )}
      </TitledCard>

      <TitledCard
        title={t('Subscription Plans')}
        description={t('Subscribe to a plan for model access')}
        icon={<Crown className='h-4 w-4' />}
        iconTone='warning'
        action={
          plans.length > 2 ? (
            <span
              className={cn(
                'text-muted-foreground flex items-center gap-1.5 text-xs whitespace-nowrap',
                plans.length <= 4 && 'sm:hidden'
              )}
              data-plan-scroll-hint='true'
            >
              <ChevronsUpDown className='size-3.5' aria-hidden='true' />
              {t('More')}
            </span>
          ) : undefined
        }
        disableHoverEffect
        className='xl:col-start-2 xl:row-span-2 xl:row-start-1 xl:h-full xl:min-h-0 xl:self-stretch xl:[contain:size]'
        contentClassName='min-h-0 space-y-4 xl:flex xl:flex-1 xl:flex-col'
      >
        {plans.length > 0 ? (
          <div
            className='max-h-[36.75rem] [scrollbar-gutter:stable] overflow-y-auto overscroll-contain pr-2 xl:max-h-none xl:min-h-0 xl:flex-1'
            tabIndex={0}
            role='region'
            aria-label={t('Subscription Plans')}
            data-visible-plan-count='2-mobile-4-wide'
            data-subscription-plan-grid='true'
          >
            <div className='grid grid-cols-1 gap-3 p-px pb-3 sm:grid-cols-2'>
              {plans.map((p, index) => {
                const plan = p?.plan
                if (!plan) return null
                const totalAmount = Number(plan.total_amount || 0)
                const isPopular = index === 0 && plans.length > 1
                const limit = Number(plan.max_purchase_per_user || 0)
                const count = planPurchaseCountMap.get(plan.id) || 0
                const reached = limit > 0 && count >= limit

                const benefits = [
                  `${t('Validity Period')}: ${formatDuration(plan, t)}`,
                  formatResetPeriod(plan, t) !== t('No Reset')
                    ? `${t('Quota Reset')}: ${formatResetPeriod(plan, t)}`
                    : null,
                  totalAmount > 0
                    ? `${t('Total Quota')}: ${formatQuota(totalAmount)}`
                    : `${t('Total Quota')}: ${t('Unlimited')}`,
                  limit > 0 ? `${t('Purchase Limit')}: ${limit}` : null,
                  plan.upgrade_group
                    ? `${t('Upgrade Group')}: ${plan.upgrade_group}`
                    : null,
                ].filter(Boolean) as string[]

                return (
                  <Card
                    key={plan.id}
                    data-card-hover='false'
                    data-subscription-plan-id={plan.id}
                    className={cn(
                      'min-h-[17.5rem] min-w-0 gap-0 border border-border py-0',
                      isPopular && 'border-primary/70 shadow-sm'
                    )}
                  >
                    <CardContent className='flex min-h-0 flex-1 flex-col p-3.5 sm:p-4'>
                      <div className='mb-2 flex items-start justify-between gap-3'>
                        <div className='min-w-0'>
                          <h4 className='line-clamp-2 leading-tight font-semibold break-words'>
                            {plan.title || t('Subscription Plans')}
                          </h4>
                          {plan.subtitle && (
                            <p className='text-muted-foreground mt-0.5 line-clamp-2 text-xs leading-4 break-words'>
                              {plan.subtitle}
                            </p>
                          )}
                        </div>
                        {isPopular && (
                          <StatusBadge
                            variant='info'
                            copyable={false}
                            className='shrink-0'
                          >
                            <Sparkles className='h-3 w-3' />
                            {t('Recommended')}
                          </StatusBadge>
                        )}
                      </div>

                      <div className='shrink-0 py-2'>
                        <span className='text-primary text-2xl font-bold'>
                          {formatSubscriptionPrice(plan.price_amount)}
                        </span>
                      </div>

                      <div
                        className='min-h-0 flex-1 space-y-1 pb-2'
                        data-plan-benefits='true'
                      >
                        {benefits.map((label) => (
                          <div
                            key={label}
                            className='text-muted-foreground flex min-w-0 items-start gap-1.5 text-xs leading-4'
                          >
                            <Check className='text-primary mt-0.5 size-3 shrink-0' />
                            <span className='min-w-0 flex-1 break-words'>
                              {label}
                            </span>
                          </div>
                        ))}
                      </div>

                      <Separator className='mb-3 shrink-0' />

                      {reached ? (
                        <Tooltip>
                          <TooltipTrigger render={<div />}>
                            <Button
                              variant='outline'
                              className='w-full'
                              disabled
                            >
                              {t('Limit Reached')}
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>
                            {t('Purchase limit reached')} ({count}/{limit})
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        <Button
                          variant='outline'
                          className='w-full'
                          onClick={() => {
                            setSelectedPlan(p)
                            setPurchaseOpen(true)
                          }}
                        >
                          {t('Subscribe Now')}
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          </div>
        ) : (
          <p className='text-muted-foreground py-4 text-center text-sm'>
            {t('No plans available')}
          </p>
        )}
      </TitledCard>

      <SubscriptionPurchaseDialog
        open={purchaseOpen}
        onOpenChange={(open) => {
          setPurchaseOpen(open)
          if (!open) {
            fetchSelfSubscription()
          }
        }}
        plan={selectedPlan}
        enableStripe={enableStripe}
        enableCreem={enableCreem}
        enableWaffoPancake={enableWaffoPancake}
        enableOnlineTopUp={enableOnlineTopUp}
        epayMethods={epayMethods}
        userQuota={userQuota}
        onPurchaseSuccess={onPurchaseSuccess}
        purchaseLimit={
          selectedPlan?.plan?.max_purchase_per_user
            ? Number(selectedPlan.plan.max_purchase_per_user)
            : undefined
        }
        purchaseCount={
          selectedPlan?.plan?.id
            ? planPurchaseCountMap.get(selectedPlan.plan.id)
            : undefined
        }
      />
    </>
  )
}
