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
import { Calendar as CalendarIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { getDatePickerLocale } from '@/components/date-picker-locale'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

type DatePickerProps = {
  selected: Date | undefined
  onSelect: (date: Date | undefined) => void
  placeholder?: string
  ariaLabel?: string
  className?: string
  formatSelected?: (date: Date) => string
  granularity?: 'day' | 'month'
}

export function DatePicker(props: DatePickerProps) {
  const { t, i18n } = useTranslation()
  const placeholderText = props.placeholder ?? t('Pick a date')
  const calendarLocale = getDatePickerLocale(
    i18n.resolvedLanguage ?? i18n.language
  )
  const monthGranularity = props.granularity === 'month'
  const selectedText = props.selected
    ? (props.formatSelected?.(props.selected) ??
      dayjs(props.selected).format('YYYY-MM-DD'))
    : null

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant='outline'
            data-empty={!props.selected}
            data-granularity={props.granularity ?? 'day'}
            aria-label={props.ariaLabel}
            className={cn(
              'data-[empty=true]:text-muted-foreground w-[240px] justify-start text-start font-normal',
              props.className
            )}
          />
        }
      >
        {selectedText ?? <span>{placeholderText}</span>}
        <CalendarIcon className='ms-auto h-4 w-4 opacity-50' />
      </PopoverTrigger>
      <PopoverContent className='w-auto p-0'>
        {monthGranularity ? (
          <Calendar
            captionLayout='dropdown'
            month={props.selected}
            onMonthChange={(date) =>
              props.onSelect(new Date(date.getFullYear(), date.getMonth(), 1))
            }
            startMonth={new Date(1900, 0, 1)}
            endMonth={new Date()}
            locale={calendarLocale}
            classNames={{ month_grid: 'hidden' }}
          />
        ) : (
          <Calendar
            mode='single'
            captionLayout='dropdown'
            selected={props.selected}
            onSelect={props.onSelect}
            locale={calendarLocale}
            disabled={(date: Date) =>
              date > new Date() || date < new Date('1900-01-01')
            }
          />
        )}
      </PopoverContent>
    </Popover>
  )
}
