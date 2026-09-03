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
import { ArrowRight, Crown, Gauge, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AccountRechargeLink } from '@/components/account-recharge-link'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from '@/components/ui/carousel'
import { Skeleton } from '@/components/ui/skeleton'
import {
  formatDuration,
  formatResetPeriod,
  formatSubscriptionPrice,
} from '@/features/subscriptions/lib'
import type { PlanRecord } from '@/features/subscriptions/types'
import { formatQuota } from '@/lib/format'

import type { PricingModel } from '../types'
import { SubscriptionPlanEstimate } from './subscription-plan-estimate'

export interface SubscriptionPlanShowcaseProps {
  plans: PlanRecord[]
  isLoading?: boolean
  isAuthenticated: boolean
  models?: PricingModel[]
  quotaPerUnit?: number
  subscriptionDisplayModels?: string
}

function SubscriptionPlanCard(props: {
  record: PlanRecord
  isAuthenticated: boolean
  models?: PricingModel[]
  quotaPerUnit?: number
  subscriptionDisplayModels?: string
}) {
  const { t } = useTranslation()
  const plan = props.record.plan
  const totalAmount = Number(plan.total_amount || 0)
  const quota = totalAmount > 0 ? formatQuota(totalAmount) : t('Unlimited')
  const subscribeLink = props.isAuthenticated ? (
    <Link to='/wallet' />
  ) : (
    <Link to='/sign-in' search={{ redirect: '/wallet' }} />
  )

  return (
    <Card
      size='sm'
      data-card-hover='false'
      data-subscription-plan-id={plan.id}
      className='h-full min-w-0 gap-0 py-0'
    >
      <CardContent className='flex min-h-[12.5rem] flex-1 flex-col p-4'>
        <div className='flex min-w-0 items-start justify-between gap-3'>
          <div className='min-w-0'>
            <h3 className='line-clamp-2 text-base leading-snug font-semibold break-words'>
              {plan.title || t('Subscription Plans')}
            </h3>
            {plan.subtitle && (
              <p className='text-muted-foreground mt-1 line-clamp-2 text-xs leading-4 break-words'>
                {plan.subtitle}
              </p>
            )}
          </div>
          <span className='bg-warning/10 text-warning flex size-8 shrink-0 items-center justify-center rounded-md'>
            <Crown className='size-4' aria-hidden='true' />
          </span>
        </div>

        <div className='mt-4 flex flex-wrap items-baseline gap-x-2 gap-y-1'>
          <span className='text-2xl leading-none font-bold tabular-nums'>
            {formatSubscriptionPrice(plan.price_amount)}
          </span>
          <span className='text-muted-foreground text-xs'>
            / {formatDuration(plan, t)}
          </span>
        </div>

        <div className='border-border/60 mt-4 grid grid-cols-2 gap-3 border-y py-3'>
          <div className='min-w-0'>
            <span className='text-muted-foreground flex items-center gap-1 text-[11px] leading-none'>
              <Gauge className='size-3' aria-hidden='true' />
              {t('Quota per reset')}
            </span>
            <strong className='mt-1.5 block truncate text-xs font-semibold tabular-nums'>
              {quota}
            </strong>
          </div>
          <div className='min-w-0'>
            <span className='text-muted-foreground flex items-center gap-1 text-[11px] leading-none'>
              <RefreshCw className='size-3' aria-hidden='true' />
              {t('Quota Reset')}
            </span>
            <strong className='mt-1.5 block truncate text-xs font-semibold'>
              {formatResetPeriod(plan, t)}
            </strong>
          </div>
        </div>

        <SubscriptionPlanEstimate
          plan={plan}
          models={props.models}
          quotaPerUnit={props.quotaPerUnit}
          subscriptionDisplayModels={props.subscriptionDisplayModels}
        />

        <Button
          variant='default'
          className='bg-warning text-warning-foreground hover:bg-warning/85 border-warning mt-3 w-full justify-between font-semibold shadow-sm'
          render={subscribeLink}
        >
          <span>{t('Subscribe Now')}</span>
          <ArrowRight data-icon='inline-end' aria-hidden='true' />
        </Button>
      </CardContent>
    </Card>
  )
}

export function SubscriptionPlanShowcase(props: SubscriptionPlanShowcaseProps) {
  const { t } = useTranslation()

  if (!props.isLoading && props.plans.length === 0) {
    return null
  }

  return (
    <section
      className='mb-5 pt-5 sm:mb-8 sm:pt-10'
      aria-labelledby='pricing-subscription-plans-title'
      data-subscription-plan-showcase='true'
    >
      <Carousel
        opts={{ align: 'start', containScroll: 'trimSnaps' }}
        aria-label={t('Subscription Plans')}
        data-subscription-plan-carousel='true'
      >
        <div className='mb-3 flex min-w-0 items-end justify-between gap-4'>
          <div className='min-w-0'>
            <div className='flex min-w-0 flex-wrap items-center gap-3'>
              <h2
                id='pricing-subscription-plans-title'
                className='flex items-center gap-2 text-base font-semibold sm:text-lg'
              >
                <Crown
                  className='text-warning size-4 shrink-0'
                  aria-hidden='true'
                />
                {t('Subscription Plans')}
              </h2>
              <AccountRechargeLink />
            </div>
            <p className='text-muted-foreground mt-1 text-xs sm:text-sm'>
              {t('Subscribe to a plan for model access')}
            </p>
          </div>
          <div className='flex shrink-0 items-center gap-2'>
            <CarouselPrevious
              className='static translate-y-0'
              aria-label={t('Previous slide')}
            />
            <CarouselNext
              className='static translate-y-0'
              aria-label={t('Next slide')}
            />
          </div>
        </div>

        <CarouselContent className='-ml-3' data-subscription-plan-track='true'>
          {props.isLoading
            ? ['first', 'second', 'third'].map((key) => (
                <CarouselItem
                  key={key}
                  className='basis-[88%] pl-3 sm:basis-1/2 xl:basis-1/3 2xl:basis-1/4'
                >
                  <Skeleton className='h-[12.5rem] w-full' />
                </CarouselItem>
              ))
            : props.plans.map((record) => (
                <CarouselItem
                  key={record.plan.id}
                  className='basis-[88%] pl-3 sm:basis-1/2 xl:basis-1/3 2xl:basis-1/4'
                >
                  <SubscriptionPlanCard
                    record={record}
                    isAuthenticated={props.isAuthenticated}
                    models={props.models}
                    quotaPerUnit={props.quotaPerUnit}
                    subscriptionDisplayModels={props.subscriptionDisplayModels}
                  />
                </CarouselItem>
              ))}
        </CarouselContent>
      </Carousel>
    </section>
  )
}
