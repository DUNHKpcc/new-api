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
  BadgePercent,
  Bell,
  Check,
  CheckCheck,
  Gift,
  Megaphone,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import type { NotificationFeedItem } from '@/components/notifications/notification-feed'
import { partitionNotificationItems } from '@/components/notifications/notification-sections'
import { RichContent } from '@/components/rich-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LOTTERY_IMAGE_ASPECT_CLASS } from '@/features/lottery/lib/image-layout'
import { useNotifications } from '@/hooks/use-notifications'
import { getAnnouncementColorClass } from '@/lib/colors'
import { formatDateTimeObject } from '@/lib/time'
import { cn } from '@/lib/utils'

type NotificationListProps = {
  items: NotificationFeedItem[]
  loading: boolean
  emptyMessage: string
  emptyIcon?: ReactNode
  onMarkRead: (item: NotificationFeedItem) => void
}

function NotificationList(props: NotificationListProps) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <div className='text-muted-foreground flex h-[min(52vh,28rem)] items-center justify-center text-sm'>
        {t('Loading...')}
      </div>
    )
  }

  if (props.items.length === 0) {
    return (
      <div className='text-muted-foreground flex h-48 flex-col items-center justify-center gap-2 text-sm'>
        {props.emptyIcon ?? <Megaphone className='size-5' aria-hidden='true' />}
        <span>{props.emptyMessage}</span>
      </div>
    )
  }

  return (
    <ScrollArea className='h-[min(52vh,28rem)]'>
      <div className='divide-y'>
        {props.items.map((item) => {
          let sourceLabel = t('Timeline')
          let sourceMarker: ReactNode = (
            <span
              className={cn(
                'size-2 shrink-0 rounded-full',
                getAnnouncementColorClass(item.type)
              )}
              aria-hidden='true'
            />
          )
          if (item.source === 'notice') {
            sourceLabel = t('Notice')
            sourceMarker = (
              <Bell
                className='text-primary size-3.5 shrink-0'
                aria-hidden='true'
              />
            )
          } else if (item.source === 'discount') {
            sourceLabel = t('Discount')
            sourceMarker = (
              <BadgePercent
                className='text-destructive size-3.5 shrink-0'
                aria-hidden='true'
              />
            )
          } else if (item.source === 'lottery') {
            sourceLabel = t('Lottery')
            sourceMarker = (
              <Gift
                className='text-primary size-3.5 shrink-0'
                aria-hidden='true'
              />
            )
          }

          const publishedAt = item.publishDate
            ? new Date(item.publishDate)
            : null
          const publishedLabel =
            publishedAt && !Number.isNaN(publishedAt.getTime())
              ? formatDateTimeObject(publishedAt)
              : ''

          return (
            <article
              key={item.key}
              className={cn('px-3 py-3', item.unread && 'bg-primary/[0.04]')}
            >
              <div className='mb-2 flex items-center gap-2'>
                {sourceMarker}
                <span className='text-muted-foreground text-xs font-medium'>
                  {sourceLabel}
                </span>
                {publishedLabel ? (
                  <time className='text-muted-foreground/70 ms-auto text-xs'>
                    {publishedLabel}
                  </time>
                ) : null}
                {item.unread ? (
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon-sm'
                    className='ms-auto'
                    aria-label={t('Mark as read')}
                    onClick={() => props.onMarkRead(item)}
                  >
                    <Check aria-hidden='true' />
                  </Button>
                ) : null}
              </div>
              {item.title ? (
                <h3 className='mb-2 text-sm font-semibold break-words'>
                  {item.title}
                </h3>
              ) : null}
              {item.image ? (
                <img
                  src={item.image}
                  alt={item.title || t('Lottery image')}
                  className={cn(
                    'mb-3 w-full rounded-md border object-cover',
                    LOTTERY_IMAGE_ASPECT_CLASS
                  )}
                />
              ) : null}
              <div className='text-sm leading-6 break-words'>
                <RichContent breaks content={item.content} />
              </div>
              {item.extra ? (
                <div className='text-muted-foreground mt-2 text-xs'>
                  <RichContent breaks content={item.extra} />
                </div>
              ) : null}
              {item.winnerInfo ? (
                <div className='bg-muted/40 mt-3 rounded-md border p-2.5 text-xs'>
                  <p className='mb-1 font-medium'>{t('Winning information')}</p>
                  <RichContent breaks content={item.winnerInfo} />
                </div>
              ) : null}
            </article>
          )
        })}
      </div>
    </ScrollArea>
  )
}

type NotificationPopoverProps = {
  className?: string
}

export function NotificationPopover(props: NotificationPopoverProps) {
  const { t } = useTranslation()
  const notifications = useNotifications()
  const [open, setOpen] = useState(false)
  const [activeTab, setActiveTab] = useState('notice')
  const sections = partitionNotificationItems(notifications.items)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='ghost'
            size='icon'
            className={cn('relative size-9', props.className)}
            aria-label={t('System Announcements')}
            data-notification-entry='header'
          />
        }
      >
        <Bell className='size-[1.2rem]' aria-hidden='true' />
        {notifications.unreadCount > 0 ? (
          <Badge
            variant='destructive'
            className='absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center px-1 text-[10px] font-semibold tabular-nums'
            aria-hidden='true'
          >
            {notifications.unreadCount > 99 ? '99+' : notifications.unreadCount}
          </Badge>
        ) : null}
      </PopoverTrigger>

      <PopoverContent
        align='end'
        sideOffset={8}
        className='w-[min(26rem,calc(100vw-1rem))] gap-3 p-3'
      >
        <PopoverHeader className='gap-1 px-1'>
          <PopoverTitle>{t('System Announcements')}</PopoverTitle>
          <p className='text-muted-foreground text-xs'>
            {t('Latest platform updates and notices')}
          </p>
        </PopoverHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className='grid w-full grid-cols-3'>
            <TabsTrigger value='notice' className='gap-1.5'>
              <Bell className='size-3.5' aria-hidden='true' />
              {t('Notice')}
            </TabsTrigger>
            <TabsTrigger value='timeline' className='gap-1.5'>
              <Megaphone className='size-3.5' aria-hidden='true' />
              {t('Timeline')}
            </TabsTrigger>
            <TabsTrigger value='lottery' className='gap-1.5'>
              <Gift className='size-3.5' aria-hidden='true' />
              {t('Lottery')}
            </TabsTrigger>
          </TabsList>

          <TabsContent value='notice' className='mt-2'>
            <NotificationList
              items={sections.notice}
              loading={notifications.loading}
              emptyMessage={t('No announcements at this time')}
              onMarkRead={notifications.markAsRead}
            />
          </TabsContent>

          <TabsContent value='timeline' className='mt-2'>
            <NotificationList
              items={sections.timeline}
              loading={notifications.loading}
              emptyMessage={t('No system announcements')}
              onMarkRead={notifications.markAsRead}
            />
          </TabsContent>

          <TabsContent value='lottery' className='mt-2'>
            <NotificationList
              items={sections.lottery}
              loading={notifications.loading}
              emptyMessage={t('No lottery content at this time')}
              emptyIcon={<Gift className='size-5' aria-hidden='true' />}
              onMarkRead={notifications.markAsRead}
            />
          </TabsContent>
        </Tabs>

        <div className='flex justify-between gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={notifications.loading || notifications.unreadCount === 0}
            onClick={notifications.markAllAsRead}
          >
            <CheckCheck aria-hidden='true' />
            {t('Mark all as read')}
          </Button>
          <Button type='button' size='sm' onClick={() => setOpen(false)}>
            {t('Close')}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
