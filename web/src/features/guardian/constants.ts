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
import type { GuardianOrderStatus } from './types'

export const GUARDIAN_LINKS = {
  repo: 'https://github.com/DUNHKpcc/Claude-Guardian',
  wiki: 'https://github.com/DUNHKpcc/Claude-Guardian/wiki',
  issues: 'https://github.com/DUNHKpcc/Claude-Guardian/issues',
  telegram: 'https://t.me/+q4D4_Qf_FIs5NDk1',
} as const

export const GUARDIAN_REPO_API =
  'https://api.github.com/repos/DUNHKpcc/Claude-Guardian'

export const GUARDIAN_INSTALL_COMMAND = `git clone https://github.com/DUNHKpcc/Claude-Guardian.git
(
  set -eu
  skill_root="$HOME/.codex/skills"
  mkdir -p "$skill_root"
  if [ -e "$skill_root/claude-guardian" ] || [ -L "$skill_root/claude-guardian" ]; then
    echo 'Target exists, stopping. Back up your current Skill first.' >&2
    exit 1
  fi
  cp -R Claude-Guardian/skills/claude-guardian "$skill_root/claude-guardian"
)`

export const GUARDIAN_QUERY_KEYS = {
  catalog: ['guardian', 'catalog'] as const,
  myOrders: ['guardian', 'orders', 'self'] as const,
  adminServices: ['guardian', 'admin', 'services'] as const,
  adminOrders: ['guardian', 'admin', 'orders'] as const,
  githubStars: ['guardian', 'github-stars'] as const,
}

export const GUARDIAN_MAX_FEATURES = 12

/** Status changes an administrator may make; mirrors the backend contract. */
export const GUARDIAN_ADMIN_TRANSITIONS: Record<
  GuardianOrderStatus,
  GuardianOrderStatus[]
> = {
  pending_payment: ['cancelled'],
  paid: ['scheduled', 'in_progress', 'refunded'],
  scheduled: ['in_progress', 'refunded'],
  in_progress: ['delivered', 'refunded'],
  delivered: ['refunded'],
  expired: [],
  cancelled: [],
  refunded: [],
}

/** The delivery path shown as a progress track on each order. */
export const GUARDIAN_ORDER_TRACK: GuardianOrderStatus[] = [
  'pending_payment',
  'paid',
  'scheduled',
  'in_progress',
  'delivered',
]
