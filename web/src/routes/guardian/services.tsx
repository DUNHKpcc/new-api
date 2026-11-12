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
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { GuardianServices } from '@/features/guardian/services'

const searchSchema = z.object({
  reseller: z.string().max(32).optional(),
  category: z
    .enum(['setup', 'mac_mini', 'account_status'])
    .optional()
    .catch(undefined),
  billing_period: z
    .enum(['monthly', 'quarterly', 'yearly'])
    .optional()
    .catch(undefined),
  claude_max: z.literal('yes').optional().catch(undefined),
  claude_pro: z.literal('yes').optional().catch(undefined),
  claude_ultra: z.literal('yes').optional().catch(undefined),
  service: z.number().int().positive().optional().catch(undefined),
})

export const Route = createFileRoute('/guardian/services')({
  validateSearch: searchSchema,
  component: GuardianServicesRoute,
})

function GuardianServicesRoute() {
  const search = Route.useSearch()
  return (
    <GuardianServices
      resellerCode={search.reseller}
      requestedBillingPeriod={search.billing_period}
      requestedClaudeMax={search.claude_max === 'yes'}
      requestedClaudePro={search.claude_pro === 'yes'}
      requestedClaudeUltra={search.claude_ultra === 'yes'}
      requestedServiceId={search.service}
      category={search.category}
    />
  )
}
