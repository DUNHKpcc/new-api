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
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react'

import {
  canPublicHeaderShowAllLinks,
  publicHeaderDesktopMediaQuery,
} from './public-header-layout'

export function usePublicHeaderNavCapacity(layoutKey: string) {
  const barRef = useRef<HTMLElement>(null)
  const brandRef = useRef<HTMLDivElement>(null)
  const actionsRef = useRef<HTMLDivElement>(null)
  const navMeasurementRef = useRef<HTMLDivElement>(null)
  const [showAllLinks, setShowAllLinks] = useState(false)

  const resolveCapacity = useCallback(() => {
    if (!window.matchMedia(publicHeaderDesktopMediaQuery).matches) {
      setShowAllLinks(false)
      return
    }

    const bar = barRef.current
    const brand = brandRef.current
    const actions = actionsRef.current
    const navMeasurement = navMeasurementRef.current
    if (!bar || !brand || !actions || !navMeasurement) return

    const nextShowAllLinks = canPublicHeaderShowAllLinks({
      barWidth: bar.getBoundingClientRect().width,
      navWidth: navMeasurement.getBoundingClientRect().width,
      leftWidth: brand.getBoundingClientRect().width,
      rightWidth: actions.getBoundingClientRect().width,
    })
    setShowAllLinks((current) =>
      current === nextShowAllLinks ? current : nextShowAllLinks
    )
  }, [])

  useLayoutEffect(() => {
    resolveCapacity()
  }, [layoutKey, resolveCapacity])

  useEffect(() => {
    const observedElements = [
      barRef.current,
      brandRef.current,
      actionsRef.current,
      navMeasurementRef.current,
    ].filter((element): element is HTMLElement => element !== null)
    const resizeObserver =
      typeof ResizeObserver === 'undefined'
        ? null
        : new ResizeObserver(resolveCapacity)
    observedElements.forEach((element) => resizeObserver?.observe(element))

    const desktopMediaQuery = window.matchMedia(publicHeaderDesktopMediaQuery)
    desktopMediaQuery.addEventListener('change', resolveCapacity)
    window.addEventListener('resize', resolveCapacity)

    return () => {
      resizeObserver?.disconnect()
      desktopMediaQuery.removeEventListener('change', resolveCapacity)
      window.removeEventListener('resize', resolveCapacity)
    }
  }, [layoutKey, resolveCapacity])

  return {
    actionsRef,
    barRef,
    brandRef,
    navMeasurementRef,
    showAllLinks,
  }
}
