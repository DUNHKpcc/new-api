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
import type { NavIconBadge } from '../types'

type NavItemIconProps = {
  icon?: React.ElementType
  badge?: NavIconBadge
}

export function NavItemIcon(props: NavItemIconProps) {
  if (!props.icon) return null

  const Icon = props.icon
  const BadgeIcon = props.badge?.icon

  return (
    <span className='relative flex size-4 shrink-0'>
      <Icon className='size-4' aria-hidden='true' />
      {BadgeIcon && (
        <span
          className='border-sidebar bg-destructive text-destructive-foreground absolute -top-1 -right-1 flex size-3.5 items-center justify-center rounded-full border-2 shadow-xs [&_svg]:size-2.5!'
          aria-hidden='true'
        >
          <BadgeIcon />
        </span>
      )}
      {props.badge && <span className='sr-only'>{props.badge.label}</span>}
    </span>
  )
}
