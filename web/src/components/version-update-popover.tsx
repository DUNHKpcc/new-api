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
import { History } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

type VersionUpdatePopoverProps = {
  className?: string
}

export function VersionUpdatePopover(props: VersionUpdatePopoverProps) {
  const { t } = useTranslation()
  const { status, loading } = useStatus()
  const details = status?.version_update_details?.trim() ?? ''
  const version = status?.version?.trim() ?? ''
  const label = t('Version update details')

  const trigger = (
    <Button
      type='button'
      variant='ghost'
      size='icon'
      className={cn('size-9', props.className)}
      aria-label={label}
      data-version-update-entry='header'
    >
      <History className='size-[1.2rem]' aria-hidden='true' />
    </Button>
  )

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger render={<PopoverTrigger render={trigger} />} />
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
      <PopoverContent
        align='end'
        sideOffset={8}
        className='w-[min(28rem,calc(100vw-1rem))] gap-3 p-3'
      >
        <PopoverHeader className='gap-2 px-1'>
          <div className='flex min-w-0 items-center gap-2'>
            <div className='bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md'>
              <History className='size-4' aria-hidden='true' />
            </div>
            <PopoverTitle className='min-w-0 truncate'>{label}</PopoverTitle>
            {version ? (
              <Badge variant='outline' className='ms-auto shrink-0 font-mono'>
                {version}
              </Badge>
            ) : null}
          </div>
        </PopoverHeader>

        {loading ? (
          <div className='text-muted-foreground flex h-40 items-center justify-center text-sm'>
            {t('Loading...')}
          </div>
        ) : null}
        {!loading && details ? (
          <div className='max-h-[min(52vh,28rem)] overflow-y-auto px-1 pe-2 text-sm leading-6'>
            <RichContent breaks content={details} />
          </div>
        ) : null}
        {!loading && !details ? (
          <div className='text-muted-foreground flex h-40 flex-col items-center justify-center gap-2 text-sm'>
            <History className='size-5' aria-hidden='true' />
            <span>{t('No version update details have been published')}</span>
          </div>
        ) : null}
      </PopoverContent>
    </Popover>
  )
}
