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
import { Link, useRouterState } from '@tanstack/react-router'
import { Menu } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

import type { TopNavLink } from '../types'
import { appHeaderLayoutClasses } from './app-header-layout'
import { isPublicNavLinkActive } from './public-header-layout'

type TopNavProps = React.HTMLAttributes<HTMLElement> & {
  links: TopNavLink[]
}

/**
 * 顶部导航栏组件
 * 在大屏幕显示水平导航，在小屏幕显示下拉菜单
 */
export function TopNav({ className, links, ...props }: TopNavProps) {
  const { t } = useTranslation()
  const pathname = useRouterState().location.pathname
  // 规范化链接，确保所有可选属性都有默认值
  const normalizedLinks = useMemo(
    () =>
      links.map((link) => ({
        disabled: false,
        external: false,
        ...link,
      })),
    [links]
  )

  return (
    <>
      {/* Compact menu protects the enlarged console topbar at laptop widths. */}
      <div className={appHeaderLayoutClasses.topNav.compact}>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger
            render={
              <Button
                size='icon'
                variant='ghost'
                className={appHeaderLayoutClasses.topNav.compactTrigger}
                aria-label={t('Toggle navigation menu')}
              />
            }
          >
            <Menu aria-hidden='true' />
          </DropdownMenuTrigger>
          <DropdownMenuContent side='bottom' align='start'>
            {normalizedLinks.map((link) => {
              const isActive = isPublicNavLinkActive(pathname, link)

              return (
                <DropdownMenuItem
                  key={`${link.title}-${link.href}`}
                  render={
                    link.external ? (
                      <a
                        href={link.href}
                        target='_blank'
                        rel='noopener noreferrer'
                        className={!isActive ? 'text-muted-foreground' : ''}
                      >
                        {link.title}
                      </a>
                    ) : (
                      <Link
                        to={link.href}
                        className={!isActive ? 'text-muted-foreground' : ''}
                        disabled={link.disabled}
                      >
                        {link.title}
                      </Link>
                    )
                  }
                />
              )
            })}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* 桌面端水平导航 */}
      <nav
        className={cn(appHeaderLayoutClasses.topNav.desktop, className)}
        {...props}
      >
        {normalizedLinks.map((link) => {
          const isActive = isPublicNavLinkActive(pathname, link)

          return link.external ? (
            <a
              key={`${link.title}-${link.href}`}
              href={link.href}
              target='_blank'
              rel='noopener noreferrer'
              className={cn(
                appHeaderLayoutClasses.topNav.link,
                isActive
                  ? appHeaderLayoutClasses.topNav.linkActive
                  : 'text-muted-foreground'
              )}
            >
              {link.title}
            </a>
          ) : (
            <Link
              key={`${link.title}-${link.href}`}
              to={link.href}
              disabled={link.disabled}
              aria-current={isActive ? 'page' : undefined}
              className={cn(
                appHeaderLayoutClasses.topNav.link,
                isActive
                  ? appHeaderLayoutClasses.topNav.linkActive
                  : 'text-muted-foreground'
              )}
            >
              {link.title}
            </Link>
          )
        })}
      </nav>
    </>
  )
}
