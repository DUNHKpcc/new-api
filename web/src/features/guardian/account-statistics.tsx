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
import { useQuery } from '@tanstack/react-query'
import { Trans, useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampToDate } from '@/lib/format'

import { getGuardianAccountStatistics } from './api'
import { GUARDIAN_QUERY_KEYS } from './constants'
import { useAccountStatusLabels } from './hooks/use-account-status-labels'

export function GuardianAccountStatisticsPanel() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const labels = useAccountStatusLabels()
  const statistics = useQuery({
    queryKey: GUARDIAN_QUERY_KEYS.accountStatistics,
    queryFn: getGuardianAccountStatistics,
    refetchInterval: 30000,
    meta: { errorToast: false },
    retry: 1,
  })
  const data = statistics.data
  const groups = data
    ? [
        {
          key: 'healthy',
          label: t('No ban, no KYC, no phone verification'),
          count: data.healthy_users,
          color: 'bg-emerald-600 dark:bg-emerald-400',
        },
        {
          key: 'partial',
          label: t('One or two conditions flagged'),
          count:
            data.recorded_users - data.healthy_users - data.all_issues_users,
          color: 'bg-orange-500 dark:bg-orange-400',
        },
        {
          key: 'all',
          label: t('All three conditions flagged'),
          count: data.all_issues_users,
          color: 'bg-red-600 dark:bg-red-400',
        },
        {
          key: 'unknown',
          label: t('Not recorded'),
          count: data.purchased_users - data.recorded_users,
          color: 'bg-stone-400 dark:bg-stone-500',
        },
      ]
    : []
  const cells = groups.flatMap((group) =>
    Array.from({ length: Math.min(group.count, 100) }, () => group)
  )
  const cellCount = Math.max(280, Math.ceil(cells.length / 7) * 7)
  return (
    <div>
      <p className='max-w-3xl text-[var(--g-ink-soft)]'>
        {t(
          'Base sales default to no ban, no phone verification and no KYC. Purchased user statuses are recorded by administrators. Refreshes every 30 seconds.'
        )}
      </p>
      {statistics.isPending && <LoadingState />}
      {statistics.isError && (
        <ErrorState
          title={t('Failed to load account status')}
          onRetry={() => void statistics.refetch()}
        />
      )}
      {statistics.isSuccess && data && (
        <>
          <div className='mt-8 flex flex-wrap gap-x-8 gap-y-3 text-sm text-[var(--g-ink-soft)]'>
            <p>
              <Trans
                t={t}
                i18nKey='Purchasers: {{total}}'
                defaults='Purchasers: <highlight>{{total}}</highlight>'
                values={{ total: formatNumber(data.purchased_users, locale) }}
                components={{
                  highlight: (
                    <span className='font-semibold text-[var(--g-accent)]' />
                  ),
                }}
              />
            </p>
            <p>
              {t('Recorded: {{total}}', {
                total: formatNumber(data.recorded_users, locale),
              })}
            </p>
            <p>
              {t('Not recorded: {{total}}', {
                total: formatNumber(
                  data.purchased_users - data.recorded_users,
                  locale
                ),
              })}
            </p>
          </div>
          <section className='g-rule mt-8 rounded-xl border bg-[var(--g-paper-raised)] p-5'>
            <div
              className='overflow-x-auto p-1'
              tabIndex={0}
              role='region'
              aria-label={t('Account status matrix')}
            >
              <TooltipProvider delay={100}>
                <div className='grid w-max grid-flow-col grid-rows-7 gap-1.5'>
                  {Array.from({ length: cellCount }, (_, index) => {
                    const cell = cells[index]
                    const label =
                      cell?.label ||
                      t(
                        'Pale empty squares are placeholders, not account records.'
                      )
                    return (
                      <Tooltip key={index}>
                        <TooltipTrigger
                          render={<span role='img' tabIndex={cell ? 0 : -1} />}
                          aria-label={label}
                          className={`size-5 cursor-help rounded-[4px] border border-black/5 outline-offset-2 hover:ring-2 hover:ring-[var(--g-ink-soft)] focus-visible:outline-2 focus-visible:outline-[var(--g-accent)] dark:border-white/10 ${cell?.color || 'bg-slate-100 dark:bg-slate-800'}`}
                        />
                        <TooltipContent role='tooltip'>{label}</TooltipContent>
                      </Tooltip>
                    )
                  })}
                </div>
              </TooltipProvider>
            </div>
            <dl className='mt-4 flex flex-wrap gap-x-7 gap-y-3'>
              {groups.map((group) => (
                <div
                  key={group.key}
                  className='flex items-center gap-2 text-sm'
                >
                  <dt className='flex items-center gap-2 text-[var(--g-ink-soft)]'>
                    <span
                      aria-hidden='true'
                      className={`size-3 shrink-0 rounded-[3px] ${group.color}`}
                    />
                    {group.label}
                  </dt>
                  <dd className='font-semibold tabular-nums'>
                    {formatNumber(group.count, locale)}
                    {group.count > 100 && (
                      <span className='ml-1 text-xs text-[var(--g-ink-soft)]'>
                        (+{formatNumber(group.count - 100, locale)})
                      </span>
                    )}
                  </dd>
                </div>
              ))}
            </dl>
            <p className='mt-4 text-xs text-[var(--g-ink-soft)]'>
              {t(
                'Each square represents one sample. Up to 100 squares per group; remaining samples are shown as +N.'
              )}{' '}
              {t('Pale empty squares are placeholders, not account records.')}
            </p>
          </section>
          <dl className='mt-6 grid gap-5 sm:grid-cols-3'>
            {labels.map((item) => (
              <div
                key={item.key}
                className='g-rule rounded-2xl border bg-[var(--g-paper-raised)] p-6'
              >
                <dt className='text-sm text-[var(--g-ink-soft)]'>
                  {item.label}
                </dt>
                <dd className='g-display mt-4 text-5xl text-[var(--g-accent)]'>
                  {data.recorded_users === 0
                    ? '—'
                    : formatNumber(data[item.metric], locale)}
                </dd>
              </div>
            ))}
          </dl>
          {(data.updated_at > 0 || data.recorded_users === 0) && (
            <p className='mt-5 text-sm text-[var(--g-ink-soft)]'>
              {data.updated_at
                ? t('Last recorded: {{time}}', {
                    time: formatTimestampToDate(data.updated_at),
                  })
                : t('No account statuses recorded yet')}
            </p>
          )}
        </>
      )}
      <div className='g-rule mt-10 space-y-3 border-t pt-6 text-sm leading-7 text-[var(--g-ink-soft)]'>
        <p>
          {t(
            'Totals include base sales and distinct purchasers. Repeat purchases by the same user count once.'
          )}
        </p>
        <p>
          {t(
            'Counts describe currently recorded conditions. Categories may overlap. Unrecorded users are not assumed unaffected.'
          )}
        </p>
        <p>
          {t(
            'Only aggregate counts are public. No usernames, phone numbers or identity documents are displayed.'
          )}
        </p>
      </div>
    </div>
  )
}
