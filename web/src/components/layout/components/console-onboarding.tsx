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
import {
  ArrowRight,
  Download,
  HandCoins,
  X,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverArrow,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { SidebarMenuAction, useSidebar } from '@/components/ui/sidebar'

import {
  useConsoleOnboarding,
  type ConsoleOnboardingStep,
} from './console-onboarding-context'

type ConsoleOnboardingDestination = '/affiliate' | '/resource-downloads'

type ConsoleOnboardingCardProps = {
  icon: LucideIcon
  title: string
  description: string
  actionLabel: string
  destination?: ConsoleOnboardingDestination
  dismissLabel: string
  onAction: () => void
  onDismiss: () => void
}

export function ConsoleOnboardingPanel(props: ConsoleOnboardingCardProps) {
  return (
    <>
      <PopoverArrow className='before:border-foreground before:border-2' />
      <ConsoleOnboardingCard {...props} />
    </>
  )
}

export function ConsoleOnboardingCard(props: ConsoleOnboardingCardProps) {
  const Icon = props.icon
  const action = props.destination ? (
    <Button
      size='sm'
      className='w-full'
      render={<Link to={props.destination} onClick={props.onAction} />}
    >
      {props.actionLabel}
      <ArrowRight className='ms-auto' aria-hidden='true' />
    </Button>
  ) : (
    <Button size='sm' className='w-full' onClick={props.onAction}>
      {props.actionLabel}
      <ArrowRight className='ms-auto' aria-hidden='true' />
    </Button>
  )

  return (
    <>
      <PopoverHeader>
        <div className='flex items-start gap-2'>
          <span className='bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md'>
            <Icon className='size-4' aria-hidden='true' />
          </span>
          <div className='min-w-0 flex-1'>
            <PopoverTitle>{props.title}</PopoverTitle>
            <PopoverDescription>{props.description}</PopoverDescription>
          </div>
          <Button
            type='button'
            variant='ghost'
            size='icon-xs'
            aria-label={props.dismissLabel}
            title={props.dismissLabel}
            onClick={props.onDismiss}
          >
            <X aria-hidden='true' />
          </Button>
        </div>
      </PopoverHeader>
      {action}
    </>
  )
}

type ConsoleOnboardingGuideProps = Omit<
  ConsoleOnboardingCardProps,
  'onAction' | 'onDismiss'
> & {
  step: ConsoleOnboardingStep
  triggerLabel: string
}

export function ConsoleOnboardingGuide(props: ConsoleOnboardingGuideProps) {
  const { isActive, complete } = useConsoleOnboarding(props.step)
  const { setOpenMobile } = useSidebar()
  const Icon = props.icon

  if (!isActive) return null

  const completeAndClose = () => {
    complete()
    setOpenMobile(false)
  }

  return (
    <Popover
      open={isActive}
      onOpenChange={(open) => {
        if (!open) complete()
      }}
    >
      <PopoverTrigger
        render={
          <SidebarMenuAction
            showOnHover={false}
            aria-label={props.triggerLabel}
            title={props.triggerLabel}
            className='text-primary hover:text-primary group-data-[collapsible=icon]:top-0! group-data-[collapsible=icon]:right-0! group-data-[collapsible=icon]:flex!'
          />
        }
      >
        <Icon className='size-4' aria-hidden='true' />
      </PopoverTrigger>
      <PopoverContent
        side='right'
        align='start'
        sideOffset={8}
        collisionPadding={12}
        className='border-foreground w-[min(18rem,calc(100vw-1.5rem))] border-2'
      >
        <ConsoleOnboardingPanel
          icon={props.icon}
          title={props.title}
          description={props.description}
          actionLabel={props.actionLabel}
          destination={props.destination}
          dismissLabel={props.dismissLabel}
          onAction={completeAndClose}
          onDismiss={complete}
        />
      </PopoverContent>
    </Popover>
  )
}

export function AffiliateOnboarding() {
  const { t } = useTranslation()

  return (
    <ConsoleOnboardingGuide
      step='affiliate'
      icon={HandCoins}
      title={t('Earn rewards with referrals')}
      description={t(
        'Become an affiliate to earn commissions. Invite users and receive quota rewards.'
      )}
      actionLabel={t('Open Affiliate Center')}
      destination='/affiliate'
      triggerLabel={t('Affiliate onboarding')}
      dismissLabel={t('Dismiss affiliate guide')}
    />
  )
}

export function ResourceDownloadsOnboarding() {
  const { t } = useTranslation()

  return (
    <ConsoleOnboardingGuide
      step='resource-downloads'
      icon={Download}
      title={t('Resource Downloads')}
      description={t(
        'Download clients, examples, and other helpful resources from here.'
      )}
      actionLabel={t('Open Resource Downloads')}
      destination='/resource-downloads'
      triggerLabel={t('Resource downloads onboarding')}
      dismissLabel={t('Dismiss resource downloads guide')}
    />
  )
}
