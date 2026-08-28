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
import { Calendar, Filter, RotateCcw, Search } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  TIME_GRANULARITY_OPTIONS,
  TIME_RANGE_PRESETS,
} from '@/features/dashboard/constants'
import { getRollingDateRange, type TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'

export interface TimeRangeFilterValue {
  start_timestamp?: Date
  end_timestamp?: Date
  time_granularity?: TimeGranularity
  username?: string
}

export interface TimeRangeFilterDialogProps {
  currentFilters: TimeRangeFilterValue
  defaultTimeRangeDays: number
  defaultTimeGranularity: TimeGranularity
  onFilterChange: (filters: TimeRangeFilterValue) => void
  onReset: () => void
  titleKey?: string
  descriptionKey?: string
  showGranularity?: boolean
  showUsername?: boolean
}

function granularityForRangeDays(days: number): TimeGranularity {
  if (days <= 1) return 'hour'
  if (days >= 29) return 'week'
  return 'day'
}

function detectQuickRangeDays(
  filters: TimeRangeFilterValue | undefined
): number | null {
  const start = filters?.start_timestamp
  const end = filters?.end_timestamp
  if (!start || !end) return null
  const days = Math.round((end.getTime() - start.getTime()) / 86_400_000)
  return TIME_RANGE_PRESETS.some((preset) => preset.days === days) ? days : null
}

function SectionDivider({ label }: { label: string }) {
  return (
    <div className='relative'>
      <div className='absolute inset-0 flex items-center'>
        <span className='w-full border-t' />
      </div>
      <div className='relative flex justify-center text-xs uppercase'>
        <span className='bg-background text-muted-foreground px-2'>
          {label}
        </span>
      </div>
    </div>
  )
}

export function TimeRangeFilterDialog({
  currentFilters,
  defaultTimeRangeDays,
  defaultTimeGranularity,
  onFilterChange,
  onReset,
  titleKey = 'Model Analytics Filters',
  descriptionKey = 'Filter the model analytics view by time range and user.',
  showGranularity = true,
  showUsername = false,
}: TimeRangeFilterDialogProps) {
  const { t } = useTranslation()
  const defaults = {
    start_timestamp: undefined,
    end_timestamp: undefined,
    time_granularity: defaultTimeGranularity,
    username: '',
  }
  const [open, setOpen] = useState(false)
  const [filters, setFilters] = useState<TimeRangeFilterValue>(() => ({
    ...defaults,
    ...currentFilters,
  }))
  const [selectedRange, setSelectedRange] = useState<number | null>(() =>
    detectQuickRangeDays(currentFilters)
  )

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setFilters({ ...defaults, ...currentFilters })
      setSelectedRange(detectQuickRangeDays(currentFilters))
    }
    setOpen(nextOpen)
  }

  const handleApply = () => {
    const cleaned = Object.fromEntries(
      Object.entries(filters).filter(([, value]) => {
        if (value === undefined || value === null) return false
        return typeof value !== 'string' || value.trim().length > 0
      })
    ) as TimeRangeFilterValue
    onFilterChange(cleaned)
    setOpen(false)
  }

  const handleReset = () => {
    const { start, end } = getRollingDateRange(defaultTimeRangeDays)
    setFilters({
      ...defaults,
      start_timestamp: start,
      end_timestamp: end,
    })
    setSelectedRange(defaultTimeRangeDays)
    onReset()
    setOpen(false)
  }

  const handleChange = (
    field: keyof TimeRangeFilterValue,
    value: Date | string | undefined
  ) => {
    setFilters((prev) => ({ ...prev, [field]: value }))
    if (field === 'start_timestamp' || field === 'end_timestamp') {
      setSelectedRange(null)
    }
  }

  const handleQuickRange = (days: number) => {
    const { start, end } = getRollingDateRange(days)
    setFilters((prev) => ({
      ...prev,
      start_timestamp: start,
      end_timestamp: end,
      time_granularity: granularityForRangeDays(days),
    }))
    setSelectedRange(days)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      trigger={
        <Button variant='outline' size='sm'>
          <Filter className='mr-2 h-4 w-4' />
          {t('Filter')}
        </Button>
      }
      title={t(titleKey)}
      description={t(descriptionKey)}
      contentClassName='max-sm:h-dvh max-sm:w-screen max-sm:max-w-none max-sm:rounded-none max-sm:p-4 sm:max-w-lg'
      contentHeight='min(48vh, 460px)'
      footerClassName='grid grid-cols-2 gap-2 sm:flex'
      footer={
        <>
          <Button onClick={handleReset} variant='outline' type='button'>
            <RotateCcw className='mr-2 h-4 w-4' />
            {t('Reset')}
          </Button>
          <Button onClick={handleApply} type='submit'>
            <Search className='mr-2 h-4 w-4' />
            {t('Apply Filters')}
          </Button>
        </>
      }
    >
      <ScrollArea className='h-full pr-3 sm:pr-4'>
        <div className='grid gap-2.5 py-2'>
          <div className='grid gap-2'>
            <Label className='flex items-center gap-2'>
              <Calendar className='h-4 w-4' />
              {t('Quick Range')}
            </Label>
            <div className='grid grid-cols-2 gap-2 sm:flex'>
              {TIME_RANGE_PRESETS.map((range) => (
                <Button
                  key={range.days}
                  type='button'
                  size='sm'
                  variant={selectedRange === range.days ? 'default' : 'outline'}
                  onClick={() => handleQuickRange(range.days)}
                  className={cn(
                    'flex-1',
                    selectedRange === range.days &&
                      'ring-ring ring-2 ring-offset-2'
                  )}
                >
                  {t(range.label)}
                </Button>
              ))}
            </div>
          </div>

          <SectionDivider label={t('Custom Time Range')} />
          <div className='grid gap-2.5'>
            <div className='grid gap-2'>
              <Label htmlFor='start_timestamp'>{t('Start Time')}</Label>
              <DateTimePicker
                value={filters.start_timestamp}
                onChange={(date) =>
                  handleChange('start_timestamp', date || undefined)
                }
                placeholder={t('Select start time')}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='end_timestamp'>{t('End Time')}</Label>
              <DateTimePicker
                value={filters.end_timestamp}
                onChange={(date) =>
                  handleChange('end_timestamp', date || undefined)
                }
                placeholder={t('Select end time')}
              />
            </div>
          </div>

          {showGranularity ? (
            <>
              <SectionDivider label={t('Chart Settings')} />
              <div className='grid gap-2'>
                <Label htmlFor='time_granularity'>
                  {t('Time Granularity')}
                </Label>
                <Select
                  items={TIME_GRANULARITY_OPTIONS.map((option) => ({
                    value: option.value,
                    label: t(option.label),
                  }))}
                  value={filters.time_granularity}
                  onValueChange={(value) =>
                    handleChange('time_granularity', value as TimeGranularity)
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t('Select time granularity')} />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {TIME_GRANULARITY_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {t(option.label)}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
            </>
          ) : null}

          {showUsername ? (
            <>
              <SectionDivider label={t('Admin Only')} />
              <div className='grid gap-2'>
                <Label htmlFor='username'>{t('Username')}</Label>
                <Input
                  id='username'
                  placeholder={t('Filter by username')}
                  value={filters.username ?? ''}
                  onChange={(event) =>
                    handleChange('username', event.target.value)
                  }
                />
              </div>
            </>
          ) : null}
        </div>
      </ScrollArea>
    </Dialog>
  )
}
