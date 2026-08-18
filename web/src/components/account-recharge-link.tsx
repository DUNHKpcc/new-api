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
import { ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

const ACCOUNT_RECHARGE_URL = 'https://dpccgaming.xyz/payment'

type AccountRechargeLinkProps = {
  className?: string
}

export function AccountRechargeLink(props: AccountRechargeLinkProps) {
  const { t } = useTranslation()

  return (
    <Button
      variant='outline'
      size='sm'
      className={cn('max-w-full', props.className)}
      render={
        <a
          data-account-recharge-link='true'
          href={ACCOUNT_RECHARGE_URL}
          target='_blank'
          rel='noopener noreferrer'
        />
      }
    >
      <span className='flex shrink-0 items-center gap-1.5' aria-hidden='true'>
        <span
          data-account-recharge-icon='codex'
          className='flex size-4 items-center justify-center'
        >
          {getLobeIcon('Codex.Color', 14)}
        </span>
        <span
          data-account-recharge-icon='claude'
          className='flex size-4 items-center justify-center'
        >
          {getLobeIcon('Claude.Color', 14)}
        </span>
      </span>
      <span className='whitespace-nowrap'>
        {t('Official accounts / recharge')}
      </span>
      <ExternalLink
        data-icon='inline-end'
        className='text-muted-foreground'
        aria-hidden='true'
      />
    </Button>
  )
}
