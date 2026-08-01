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
  ChevronDown,
  Megaphone,
} from 'lucide-react'
import { AnimatePresence, motion } from 'motion/react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useNotifications } from '@/hooks/use-notifications'
import { getAnnouncementColorClass } from '@/lib/colors'
import { formatDateTimeObject } from '@/lib/time'
import { cn } from '@/lib/utils'

import { globalNotificationCenterLayout } from './layout'
import {
  getNotificationPreview,
  type NotificationFeedItem,
} from './notification-feed'
import { useFloatingNotificationPosition } from './use-floating-notification-position'

const NOTIFICATION_FEED_ID = 'global-notification-feed'

type NotificationItemProps = {
  item: NotificationFeedItem
  expanded: boolean
  onExpandedChange: (key: string) => void
  onMarkRead: (item: NotificationFeedItem) => void
}

function NotificationItem(props: NotificationItemProps) {
  const { t } = useTranslation()
  const detailId = `notification-detail-${props.item.key.replaceAll(
    /[^a-zA-Z0-9_-]/g,
    '-'
  )}`
  const preview = getNotificationPreview(props.item.content)
  const publishedAt = props.item.publishDate
    ? new Date(props.item.publishDate)
    : null
  const publishedLabel =
    publishedAt && !Number.isNaN(publishedAt.getTime())
      ? formatDateTimeObject(publishedAt)
      : ''
  let sourceLabel = t('Timeline')
  let sourceMarker: ReactNode = (
    <span
      className={cn(
        'size-2 shrink-0 rounded-full',
        getAnnouncementColorClass(props.item.type)
      )}
      aria-hidden='true'
    />
  )
  if (props.item.source === 'notice') {
    sourceLabel = t('Notice')
    sourceMarker = (
      <span
        className='bg-primary size-2 shrink-0 rounded-full'
        aria-hidden='true'
      />
    )
  } else if (props.item.source === 'discount') {
    sourceLabel = t('Discount')
    sourceMarker = (
      <BadgePercent
        className='text-destructive size-3.5 shrink-0'
        aria-hidden='true'
      />
    )
  }

  return (
    <article
      className={cn(
        'relative border-t first:border-t-0',
        props.item.unread && 'bg-primary/[0.04]'
      )}
    >
      <div className='flex items-start'>
        <button
          type='button'
          className='hover:bg-muted/40 focus-visible:ring-ring/50 min-w-0 flex-1 px-3 py-3 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-inset'
          aria-expanded={props.expanded}
          aria-controls={detailId}
          onClick={() => props.onExpandedChange(props.item.key)}
        >
          <div className='flex items-center gap-2'>
            {sourceMarker}
            <span className='text-muted-foreground text-xs font-medium'>
              {sourceLabel}
            </span>
            {publishedLabel ? (
              <time className='text-muted-foreground/70 ms-auto truncate text-xs'>
                {publishedLabel}
              </time>
            ) : null}
          </div>
          <div className='mt-1.5 flex items-start gap-2'>
            <p
              className={cn(
                'line-clamp-2 min-w-0 flex-1 text-sm leading-5',
                props.item.unread && 'font-medium'
              )}
            >
              {preview}
            </p>
            <ChevronDown
              className={cn(
                'text-muted-foreground mt-0.5 size-4 shrink-0 transition-transform',
                props.expanded && 'rotate-180'
              )}
              aria-hidden='true'
            />
          </div>
        </button>

        {props.item.unread ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant='ghost'
                  size='icon-sm'
                  className='me-2 mt-2'
                  aria-label={t('Mark as read')}
                  onClick={() => props.onMarkRead(props.item)}
                />
              }
            >
              <Check />
            </TooltipTrigger>
            <TooltipContent>{t('Mark as read')}</TooltipContent>
          </Tooltip>
        ) : null}
      </div>

      {props.expanded ? (
        <div id={detailId} className='bg-muted/20 border-t px-3 py-3 text-sm'>
          <RichContent breaks content={props.item.content} />
          {props.item.extra ? (
            <div className='text-muted-foreground mt-2 text-xs'>
              <RichContent breaks content={props.item.extra} />
            </div>
          ) : null}
        </div>
      ) : null}
    </article>
  )
}

type CompactNotificationPreviewProps = {
  item: NotificationFeedItem
  unreadCount?: number
  onOpen: (key: string) => void
}

function CompactNotificationPreview(props: CompactNotificationPreviewProps) {
  const { t } = useTranslation()
  const preview = getNotificationPreview(props.item.content, 88)
  const label = props.item.source === 'discount' ? t('Discount') : t('Unread')

  return (
    <button
      type='button'
      className={globalNotificationCenterLayout.preview}
      aria-label={`${label}: ${preview}`}
      onClick={() => props.onOpen(props.item.key)}
    >
      <span className='flex items-center gap-2'>
        {props.item.source === 'discount' ? (
          <BadgePercent
            className='text-destructive size-4 shrink-0'
            aria-hidden='true'
          />
        ) : (
          <span
            className='bg-destructive size-2 shrink-0 rounded-full'
            aria-hidden='true'
          />
        )}
        <span className='text-destructive text-xs font-semibold'>{label}</span>
        {props.unreadCount ? (
          <Badge
            variant='destructive'
            className='ms-auto h-5 min-w-5 px-1.5 tabular-nums'
          >
            {props.unreadCount > 99 ? '99+' : props.unreadCount}
          </Badge>
        ) : null}
      </span>
      <span className={globalNotificationCenterLayout.previewText}>
        {preview}
      </span>
    </button>
  )
}

export function GlobalNotificationCenter() {
  const { t } = useTranslation()
  const notifications = useNotifications()
  const [expanded, setExpanded] = useState(false)
  const [expandedItemKey, setExpandedItemKey] = useState<string | null>(null)
  const floatingPosition = useFloatingNotificationPosition(expanded)
  const unreadItems = notifications.items.filter((item) => item.unread)

  const handleItemExpandedChange = (key: string) => {
    setExpandedItemKey((currentKey) => (currentKey === key ? null : key))
  }

  const handlePreviewOpen = (key: string) => {
    setExpandedItemKey(key)
    setExpanded(true)
  }

  let feedContent: ReactNode
  if (notifications.loading) {
    feedContent = (
      <div className='text-muted-foreground flex min-h-24 items-center justify-center px-4 text-sm'>
        {t('Loading...')}
      </div>
    )
  } else if (notifications.items.length === 0) {
    feedContent = (
      <div className='text-muted-foreground flex min-h-24 flex-col items-center justify-center gap-2 px-4 text-sm'>
        <Megaphone className='size-5' aria-hidden='true' />
        <span>{t('No system announcements')}</span>
      </div>
    )
  } else {
    feedContent = notifications.items.map((item) => (
      <NotificationItem
        key={item.key}
        item={item}
        expanded={expandedItemKey === item.key}
        onExpandedChange={handleItemExpandedChange}
        onMarkRead={notifications.markAsRead}
      />
    ))
  }

  return (
    <aside
      ref={floatingPosition.rootRef}
      className={cn(
        globalNotificationCenterLayout.root,
        floatingPosition.dragging
          ? 'transition-none'
          : 'transition-[bottom] duration-200'
      )}
      style={floatingPosition.style}
      aria-label={t('Notifications')}
      data-floating-action='notification'
    >
      <TooltipProvider delay={150}>
        <AnimatePresence
          initial={false}
          mode={globalNotificationCenterLayout.presenceMode}
        >
          {expanded ? (
            <motion.div
              key='expanded-notification-center'
              initial={globalNotificationCenterLayout.motion.initial}
              animate={globalNotificationCenterLayout.motion.animate}
              exit={globalNotificationCenterLayout.motion.exit}
              transition={globalNotificationCenterLayout.motion.transition}
              style={{
                transformOrigin: globalNotificationCenterLayout.motionOrigin,
              }}
              className={globalNotificationCenterLayout.panel}
            >
              <div className='flex h-11 items-center border-b px-2'>
                <button
                  type='button'
                  className='hover:bg-muted focus-visible:ring-ring/50 flex h-8 min-w-0 flex-1 items-center gap-2 rounded-md px-2 text-left transition-colors outline-none focus-visible:ring-2'
                  aria-expanded='true'
                  aria-controls={NOTIFICATION_FEED_ID}
                  onClick={() => setExpanded(false)}
                >
                  <Bell className='size-4 shrink-0' aria-hidden='true' />
                  <span className='truncate text-sm font-semibold'>
                    {t('Notifications')}
                  </span>
                  {notifications.unreadCount > 0 ? (
                    <Badge
                      variant='destructive'
                      className='h-5 min-w-5 px-1.5 tabular-nums'
                    >
                      {notifications.unreadCount > 99
                        ? '99+'
                        : notifications.unreadCount}
                    </Badge>
                  ) : null}
                </button>

                {notifications.unreadCount > 0 ? (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          variant='ghost'
                          size='icon-sm'
                          aria-label={t('Mark all as read')}
                          onClick={notifications.markAllAsRead}
                        />
                      }
                    >
                      <CheckCheck />
                    </TooltipTrigger>
                    <TooltipContent>{t('Mark all as read')}</TooltipContent>
                  </Tooltip>
                ) : null}

                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        variant='ghost'
                        size='icon-sm'
                        aria-label={t('Collapse')}
                        onClick={() => setExpanded(false)}
                      />
                    }
                  >
                    <ChevronDown className='rotate-180' />
                  </TooltipTrigger>
                  <TooltipContent>{t('Collapse')}</TooltipContent>
                </Tooltip>
              </div>

              <div
                id={NOTIFICATION_FEED_ID}
                className={globalNotificationCenterLayout.feed}
              >
                {feedContent}
              </div>
            </motion.div>
          ) : (
            <motion.div
              key='compact-notification-center'
              initial={globalNotificationCenterLayout.motion.initial}
              animate={globalNotificationCenterLayout.motion.animate}
              exit={globalNotificationCenterLayout.motion.exit}
              transition={globalNotificationCenterLayout.motion.transition}
              style={{
                transformOrigin: globalNotificationCenterLayout.motionOrigin,
              }}
              className='flex flex-col items-end gap-2'
            >
              {unreadItems.length > 0 ? (
                <div
                  className={globalNotificationCenterLayout.unreadList}
                  role='region'
                  aria-label={t('Notifications')}
                >
                  {unreadItems.map((item, index) => (
                    <CompactNotificationPreview
                      key={item.key}
                      item={item}
                      unreadCount={
                        index === 0 ? notifications.unreadCount : undefined
                      }
                      onOpen={handlePreviewOpen}
                    />
                  ))}
                </div>
              ) : null}

              <Button
                variant='outline'
                className={globalNotificationCenterLayout.trigger}
                aria-expanded='false'
                aria-controls={NOTIFICATION_FEED_ID}
                aria-keyshortcuts='ArrowUp ArrowDown'
                onPointerDown={floatingPosition.handlePointerDown}
                onPointerMove={floatingPosition.handlePointerMove}
                onPointerUp={floatingPosition.finishDragging}
                onPointerCancel={floatingPosition.cancelDragging}
                onKeyDown={floatingPosition.handleKeyDown}
                onClick={() => {
                  if (floatingPosition.shouldOpenFromTrigger()) {
                    setExpanded(true)
                  }
                }}
              >
                <Bell
                  className={globalNotificationCenterLayout.triggerIcon}
                  aria-hidden='true'
                />
                <span className='text-xs leading-none'>
                  {t('Notifications')}
                </span>
                {notifications.unreadCount > 0 ? (
                  <span
                    className='bg-destructive ring-background absolute -top-1 -right-1 size-2.5 rounded-full ring-2'
                    aria-hidden='true'
                  />
                ) : null}
              </Button>
            </motion.div>
          )}
        </AnimatePresence>
      </TooltipProvider>
    </aside>
  )
}
