import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { formatNumber } from '@/lib/format'
import { formatDateTimeObject } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

import { getLotteryEntry, joinLottery } from './api'
import type { Lottery } from './types'

export function LotteryCard({ lottery }: { lottery: Lottery }) {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.auth.user)
  const client = useQueryClient()
  const entry = useQuery({
    queryKey: ['lottery-entry', lottery.id, user?.id, lottery.status],
    queryFn: () => getLotteryEntry(lottery.id),
    enabled: !!user,
    staleTime: 30_000,
  })
  const join = useMutation({
    mutationFn: () => joinLottery(lottery.id),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: ['lottery-items'] }),
        client.invalidateQueries({ queryKey: ['lottery-entry', lottery.id] }),
      ])
    },
  })
  const closed =
    lottery.status !== 'active' ||
    (lottery.draw_at > 0 && lottery.draw_at * 1000 <= Date.now()) ||
    (lottery.target_count > 0 && lottery.entry_count >= lottery.target_count)
  return (
    <div className='bg-muted/20 mt-3 space-y-3 rounded-lg border p-3 text-sm'>
      <div className='font-semibold'>
        {t('Prize')}: {lottery.prize}
      </div>
      <p>
        {t('Lottery participants')}: {formatNumber(lottery.entry_count)}
        {lottery.target_count > 0
          ? ` / ${formatNumber(lottery.target_count)}`
          : ''}{' '}
        · {t('Winner count')}: {formatNumber(lottery.winner_count)}
      </p>
      {lottery.draw_at > 0 && (
        <p>
          {t('Draw time')}:{' '}
          {formatDateTimeObject(new Date(lottery.draw_at * 1000))}
        </p>
      )}
      <p className='text-muted-foreground text-xs'>
        {t(
          {
            full: 'Draw when full',
            time: 'Scheduled draw',
            either: 'Draw at the first condition met',
          }[lottery.draw_mode]
        )}
      </p>
      <p className='text-muted-foreground text-xs'>
        {t('One entry per account. Prizes are delivered by the administrator.')}
      </p>
      {lottery.require_recharge && (
        <p>{t('A completed recharge is required')}</p>
      )}
      {lottery.min_invites > 0 && (
        <p>
          {t('Minimum invited users')}: {formatNumber(lottery.min_invites)}
        </p>
      )}
      {lottery.required_group && (
        <p>
          {t('Required user group')}: {lottery.required_group}
        </p>
      )}
      {lottery.email_notify && (
        <p className='text-muted-foreground text-xs'>
          {t('Winners receive email at their bound address.')}
        </p>
      )}
      {entry.data?.winner && (
        <p className='text-primary font-semibold' role='status'>
          {t('You won! Contact support to claim your prize.')}{' '}
          <Link to='/support'>{t('Customer Service')}</Link>
        </p>
      )}
      {!entry.data?.winner && lottery.status === 'drawn' && (
        <p role='status'>
          {t(entry.data ? 'Not selected this time' : 'Lottery drawn')}
        </p>
      )}
      {lottery.status !== 'drawn' && !!entry.data && (
        <p role='status'>{t('Entered successfully')}</p>
      )}
      {lottery.status !== 'drawn' && !entry.data && user && (
        <Button
          size='sm'
          disabled={
            closed ||
            entry.isPending ||
            entry.isError ||
            join.isPending ||
            (lottery.email_notify && !user.email)
          }
          onClick={() => join.mutate()}
        >
          {t('Enter lottery')}
        </Button>
      )}
      {!user && lottery.status !== 'drawn' && (
        <Link to='/sign-in'>{t('Login')}</Link>
      )}
      {user && lottery.email_notify && !user.email && (
        <p className='text-destructive'>
          {t('Bind an email address before entering')}
        </p>
      )}
      <Button
        size='sm'
        variant='ghost'
        disabled={entry.isFetching}
        onClick={() => {
          void client.invalidateQueries({ queryKey: ['lottery-items'] })
          void client.invalidateQueries({
            queryKey: ['lottery-entry', lottery.id],
          })
        }}
      >
        {t('Refresh')}
      </Button>
    </div>
  )
}
