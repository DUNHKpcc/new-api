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
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { Menu, X } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { LanguageSwitcher } from '@/components/language-switcher'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { defaultTopNavLinks } from '../config/top-nav.config'
import type { TopNavLink } from '../types'
import { HeaderLogo } from './header-logo'
import {
  publicHeaderDesktopMediaQuery,
  publicHeaderLayoutClasses,
  splitPublicHeaderLinks,
} from './public-header-layout'
import {
  PublicHeaderDesktopLinkMeasurement,
  PublicHeaderDesktopLinks,
  PublicHeaderMobileLinks,
  PublicHeaderOverflowMenu,
} from './public-header-nav'
import { usePublicHeaderNavCapacity } from './use-public-header-nav-capacity'

const AUTH_PROMPT_SECONDS = 5

type AuthPromptTarget = {
  title: string
  href: string
}

export interface PublicHeaderProps {
  navLinks?: TopNavLink[]
  mobileLinks?: TopNavLink[]
  navContent?: React.ReactNode
  showThemeSwitch?: boolean
  showLanguageSwitcher?: boolean
  logo?: React.ReactNode
  siteName?: string
  homeUrl?: string
  leftContent?: React.ReactNode
  rightContent?: React.ReactNode
  showNavigation?: boolean
  showAuthButtons?: boolean
  showNotifications?: boolean
  className?: string
}

export function PublicHeader(props: PublicHeaderProps) {
  const {
    navLinks = defaultTopNavLinks,
    showThemeSwitch = true,
    showLanguageSwitcher = true,
    logo: customLogo,
    siteName: customSiteName,
    homeUrl = '/',
    showNavigation = true,
    showAuthButtons = true,
    showNotifications = true,
  } = props

  const { i18n, t } = useTranslation()
  const navigate = useNavigate()
  const [scrolled, setScrolled] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [authPromptTarget, setAuthPromptTarget] =
    useState<AuthPromptTarget | null>(null)
  const [authPromptSecondsLeft, setAuthPromptSecondsLeft] =
    useState(AUTH_PROMPT_SECONDS)
  const { auth } = useAuthStore()
  const {
    systemName,
    logo: systemLogo,
    loading,
    logoLoaded,
  } = useSystemConfig()
  const dynamicLinks = useTopNavLinks()
  const routerState = useRouterState()
  const pathname = routerState.location.pathname

  const user = auth.user
  const isAuthenticated = !!user
  const displaySiteName = customSiteName || systemName
  const links = dynamicLinks.length > 0 ? dynamicLinks : navLinks
  const mobileLinks = props.mobileLinks ?? links
  const { primaryLinks, overflowLinks } = useMemo(
    () => splitPublicHeaderLinks(links),
    [links]
  )
  const hasDesktopUtilities =
    showNotifications || showLanguageSwitcher || showThemeSwitch
  const navLayoutKey = `${showNavigation}:${props.navContent ? 'custom' : 'default'}:${
    i18n.resolvedLanguage || i18n.language
  }:${links.map((link) => `${link.href}:${link.title}`).join('|')}`
  const navCapacity = usePublicHeaderNavCapacity(navLayoutKey)

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 20)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    const desktopMediaQuery = window.matchMedia(publicHeaderDesktopMediaQuery)
    const closeMobileMenuOnDesktop = (event: MediaQueryListEvent) => {
      if (event.matches) setMobileOpen(false)
    }

    desktopMediaQuery.addEventListener('change', closeMobileMenuOnDesktop)
    return () => {
      desktopMediaQuery.removeEventListener('change', closeMobileMenuOnDesktop)
    }
  }, [])

  useEffect(() => {
    if (!mobileOpen) return

    const previousOverflow = document.body.style.getPropertyValue('overflow')
    const previousPriority = document.body.style.getPropertyPriority('overflow')
    document.body.style.setProperty('overflow', 'hidden', 'important')

    return () => {
      if (previousOverflow) {
        document.body.style.setProperty(
          'overflow',
          previousOverflow,
          previousPriority
        )
      } else {
        document.body.style.removeProperty('overflow')
      }
    }
  }, [mobileOpen])

  useEffect(() => {
    setMobileOpen(false)
  }, [pathname])

  useEffect(() => {
    if (!authPromptTarget) return

    const intervalId = window.setInterval(() => {
      setAuthPromptSecondsLeft((seconds) => Math.max(seconds - 1, 0))
    }, 1000)

    const timeoutId = window.setTimeout(() => {
      const redirect = authPromptTarget.href
      setAuthPromptTarget(null)
      navigate({ to: '/sign-in', search: { redirect } })
    }, AUTH_PROMPT_SECONDS * 1000)

    return () => {
      window.clearInterval(intervalId)
      window.clearTimeout(timeoutId)
    }
  }, [authPromptTarget, navigate])

  const closeAuthPrompt = useCallback(() => {
    setAuthPromptTarget(null)
    setAuthPromptSecondsLeft(AUTH_PROMPT_SECONDS)
  }, [])

  const navigateToSignIn = useCallback(() => {
    const redirect = authPromptTarget?.href || '/'
    setAuthPromptTarget(null)
    navigate({ to: '/sign-in', search: { redirect } })
  }, [authPromptTarget?.href, navigate])

  const handleNavLinkClick = useCallback(
    (
      event: React.MouseEvent<HTMLAnchorElement>,
      link: TopNavLink,
      closeMobile = false
    ) => {
      if (link.disabled) {
        event.preventDefault()
        return
      }

      if (link.requiresAuth && !isAuthenticated) {
        event.preventDefault()
        if (closeMobile) {
          setMobileOpen(false)
        }
        setAuthPromptSecondsLeft(AUTH_PROMPT_SECONDS)
        setAuthPromptTarget({
          title: t(link.title),
          href: link.href,
        })
        return
      }

      if (closeMobile) {
        setMobileOpen(false)
      }
    },
    [isAuthenticated, t]
  )

  let logoContent: React.ReactNode = (
    <HeaderLogo
      src={systemLogo}
      alt=''
      loading={loading}
      logoLoaded={logoLoaded}
      className='size-full rounded-sm object-contain'
    />
  )
  if (loading) {
    logoContent = <Skeleton className='size-full rounded-sm' />
  } else if (customLogo) {
    logoContent = customLogo
  }

  let desktopAuthControl: React.ReactNode = (
    <Button
      size='sm'
      className={publicHeaderLayoutClasses.authButton}
      render={<Link to='/sign-in' />}
    >
      {t('Sign in')}
    </Button>
  )
  if (loading) {
    desktopAuthControl = (
      <Skeleton className={publicHeaderLayoutClasses.authSkeleton} />
    )
  } else if (isAuthenticated) {
    desktopAuthControl = <ProfileDropdown />
  }

  return (
    <>
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <header
          className={cn(
            publicHeaderLayoutClasses.header.base,
            scrolled
              ? publicHeaderLayoutClasses.header.scrolled
              : publicHeaderLayoutClasses.header.idle,
            props.className
          )}
        >
          <div className={publicHeaderLayoutClasses.shell.base}>
            <nav
              ref={navCapacity.barRef}
              className={cn(
                publicHeaderLayoutClasses.bar.base,
                scrolled
                  ? publicHeaderLayoutClasses.bar.scrolled
                  : publicHeaderLayoutClasses.bar.idle
              )}
            >
              <div
                ref={navCapacity.brandRef}
                className={publicHeaderLayoutClasses.left}
              >
                <Link
                  to={homeUrl}
                  className={publicHeaderLayoutClasses.brand.link}
                >
                  <div className={publicHeaderLayoutClasses.brand.mark}>
                    {logoContent}
                  </div>
                  <span className={publicHeaderLayoutClasses.brand.name}>
                    {loading ? (
                      <Skeleton className='h-5 w-20' />
                    ) : (
                      displaySiteName
                    )}
                  </span>
                </Link>
                {props.leftContent}
              </div>

              {showNavigation &&
                (props.navContent ? (
                  <div className='hidden h-full items-stretch justify-self-center lg:flex'>
                    {props.navContent}
                  </div>
                ) : (
                  <>
                    <div
                      ref={navCapacity.navMeasurementRef}
                      aria-hidden='true'
                      className={
                        publicHeaderLayoutClasses.desktopNavMeasurement
                      }
                    >
                      <PublicHeaderDesktopLinkMeasurement links={links} />
                    </div>
                    <div className={publicHeaderLayoutClasses.desktopNav}>
                      <PublicHeaderDesktopLinks
                        links={primaryLinks}
                        pathname={pathname}
                        onLinkClick={handleNavLinkClick}
                      />
                      {navCapacity.showAllLinks ? (
                        <PublicHeaderDesktopLinks
                          links={overflowLinks}
                          pathname={pathname}
                          onLinkClick={handleNavLinkClick}
                        />
                      ) : (
                        <PublicHeaderOverflowMenu
                          links={overflowLinks}
                          pathname={pathname}
                          onLinkClick={handleNavLinkClick}
                        />
                      )}
                    </div>
                  </>
                ))}

              <div
                ref={navCapacity.actionsRef}
                className={publicHeaderLayoutClasses.desktopActions}
              >
                {props.rightContent}
                {props.rightContent && hasDesktopUtilities && (
                  <div className='bg-foreground/12 mx-2 h-6 w-px' />
                )}
                {hasDesktopUtilities && (
                  <div className={publicHeaderLayoutClasses.utilityActions}>
                    {showNotifications && <NotificationPopover />}
                    {showLanguageSwitcher && <LanguageSwitcher />}
                    {showThemeSwitch && <ThemeSwitch />}
                  </div>
                )}

                {showAuthButtons && (
                  <>
                    <div className='bg-foreground/12 mx-2 h-6 w-px' />
                    {desktopAuthControl}
                  </>
                )}
              </div>

              <div className={publicHeaderLayoutClasses.mobileActions}>
                {props.rightContent}
                {showNotifications && <NotificationPopover />}
                {showAuthButtons && !loading && isAuthenticated && (
                  <ProfileDropdown />
                )}
                <SheetTrigger
                  render={
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      aria-label={t('Toggle navigation menu')}
                      aria-expanded={mobileOpen}
                      aria-controls='public-mobile-navigation'
                    />
                  }
                >
                  <Menu aria-hidden='true' />
                </SheetTrigger>
              </div>
            </nav>
          </div>
        </header>

        <SheetContent
          id='public-mobile-navigation'
          side='right'
          showCloseButton={false}
          className={publicHeaderLayoutClasses.mobilePanel}
        >
          <SheetHeader className={publicHeaderLayoutClasses.mobilePanelHeader}>
            <SheetTitle className={publicHeaderLayoutClasses.mobilePanelBrand}>
              <span className='flex size-8 shrink-0 items-center justify-center'>
                {logoContent}
              </span>
              <span className='min-w-0 truncate'>{displaySiteName}</span>
            </SheetTitle>
            <SheetDescription className='sr-only'>
              {t('Toggle navigation menu')}
            </SheetDescription>
            <SheetClose
              render={
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  className='size-10 rounded-md'
                  aria-label={t('Close')}
                />
              }
            >
              <X aria-hidden='true' />
            </SheetClose>
          </SheetHeader>

          {showNavigation && (
            <nav className={publicHeaderLayoutClasses.mobileNav}>
              <PublicHeaderMobileLinks
                links={mobileLinks}
                pathname={pathname}
                onLinkClick={handleNavLinkClick}
              />
            </nav>
          )}

          <SheetFooter className={publicHeaderLayoutClasses.mobileFooter}>
            {(showLanguageSwitcher || showThemeSwitch) && (
              <div className='flex flex-col gap-2'>
                {showLanguageSwitcher && (
                  <div
                    className={publicHeaderLayoutClasses.mobileLanguageAction}
                  >
                    <span>{t('Change language')}</span>
                    <LanguageSwitcher />
                  </div>
                )}
                {showThemeSwitch && (
                  <div
                    className={publicHeaderLayoutClasses.mobileLanguageAction}
                  >
                    <span>{t('Toggle theme')}</span>
                    <ThemeSwitch />
                  </div>
                )}
              </div>
            )}
            {showAuthButtons && (
              <Link
                to={isAuthenticated ? '/dashboard' : '/sign-in'}
                onClick={() => setMobileOpen(false)}
                className={publicHeaderLayoutClasses.mobileAuthButton}
              >
                {isAuthenticated ? t('Go to Dashboard') : t('Sign in')}
              </Link>
            )}
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <Dialog
        open={!!authPromptTarget}
        onOpenChange={(open) => {
          if (!open) {
            closeAuthPrompt()
          }
        }}
        title={t('Sign in required')}
        description={t('Please sign in to view {{module}}.', {
          module: authPromptTarget?.title || '',
        })}
        contentClassName='sm:max-w-md'
        contentHeight='auto'
        footer={
          <>
            <Button variant='outline' onClick={closeAuthPrompt}>
              {t('Cancel')}
            </Button>
            <Button onClick={navigateToSignIn}>{t('Sign in now')}</Button>
          </>
        }
      >
        <div className='bg-muted/40 text-muted-foreground rounded-lg px-3 py-2 text-sm'>
          {t('Redirecting to sign in in {{seconds}} seconds.', {
            seconds: authPromptSecondsLeft,
          })}
        </div>
      </Dialog>
    </>
  )
}
