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
import { cn } from '@/lib/utils'

export type GuardianTier = 'A' | 'B' | 'C'

const TIER_COLOR: Record<GuardianTier, string> = {
  A: 'text-[var(--g-tier-a)] border-[color-mix(in_srgb,var(--g-tier-a)_40%,transparent)]',
  B: 'text-[var(--g-tier-b)] border-[color-mix(in_srgb,var(--g-tier-b)_40%,transparent)]',
  C: 'text-[var(--g-tier-c)] border-[color-mix(in_srgb,var(--g-tier-c)_40%,transparent)]',
}

export function TierChip(props: { tier: GuardianTier; className?: string }) {
  return (
    <span
      className={cn(
        'g-mono inline-flex size-6 shrink-0 items-center justify-center rounded-full border text-[0.7rem] font-bold',
        TIER_COLOR[props.tier],
        props.className
      )}
    >
      {props.tier}
    </span>
  )
}
