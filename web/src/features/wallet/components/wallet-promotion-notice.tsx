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
import { BadgePercent } from 'lucide-react'

import { Alert, AlertDescription } from '@/components/ui/alert'

type WalletPromotionNoticeProps = {
  content?: string
}

export function WalletPromotionNotice(props: WalletPromotionNoticeProps) {
  const content = props.content?.trim()
  if (!content) return null

  return (
    <Alert className='border-destructive/25 bg-destructive/[0.04]'>
      <BadgePercent className='text-destructive' aria-hidden='true' />
      <AlertDescription className='text-foreground/90 break-words whitespace-pre-wrap'>
        {content}
      </AlertDescription>
    </Alert>
  )
}
