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
import { useTranslation } from 'react-i18next'

import { AccountRechargeLink } from '@/components/account-recharge-link'
import type { PlanRecord } from '@/features/subscriptions/types'
import { cn } from '@/lib/utils'

import { SearchBar } from './search-bar'
import { SubscriptionPlanShowcase } from './subscription-plan-showcase'

export interface PricingPageTopProps {
  plans: PlanRecord[]
  plansLoading?: boolean
  isAuthenticated: boolean
  searchInput: string
  onSearchChange: (value: string) => void
  onClearSearch: () => void
}

export function PricingPageTop(props: PricingPageTopProps) {
  const { t } = useTranslation()
  const hasPlanSection = Boolean(props.plansLoading || props.plans.length)

  return (
    <>
      {!hasPlanSection && (
        <div className='mx-auto mb-3 flex w-full max-w-3xl justify-center sm:mb-4 sm:justify-end'>
          <AccountRechargeLink />
        </div>
      )}
      <SubscriptionPlanShowcase
        plans={props.plans}
        isLoading={props.plansLoading}
        isAuthenticated={props.isAuthenticated}
      />
      <header
        className={cn(
          'mx-auto mb-5 w-full max-w-3xl sm:mb-8',
          !hasPlanSection && 'pt-5 sm:pt-10'
        )}
      >
        <SearchBar
          value={props.searchInput}
          onChange={props.onSearchChange}
          onClear={props.onClearSearch}
          placeholder={t('Search model name, provider, endpoint, or tag...')}
        />
      </header>
    </>
  )
}
