import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { SettingsSection } from '@/features/system-settings/components/settings-section'
import { formatNumber } from '@/lib/format'

import {
  changeLottery,
  listLotteries,
  listWinners,
  saveLottery,
  updateWinner,
} from './api'
import { toDateTimeLocalValue } from './lib/lottery-items'
import type { Lottery } from './types'

const schema = z
  .object({
    title: z.string().trim().min(1).max(80),
    content: z.string().trim().min(1).max(6000),
    prize: z.string().trim().min(1).max(250),
    winner_count: z.number().int().min(1).max(100),
    draw_mode: z.enum(['full', 'time', 'either']),
    target_count: z.number().int().min(0).max(100000),
    draw_time: z.string(),
    require_recharge: z.boolean(),
    min_invites: z.number().int().min(0).max(100000),
    required_group: z.string().max(64),
    email_notify: z.boolean(),
  })
  .refine((v) => v.draw_mode === 'time' || v.target_count >= v.winner_count, {
    path: ['target_count'],
    message: 'Invalid rules',
  })
  .refine(
    (v) => v.draw_mode === 'full' || Date.parse(v.draw_time) > Date.now(),
    { path: ['draw_time'], message: 'Invalid rules' }
  )
type Values = z.infer<typeof schema>
const statuses = {
  draft: 'Draft',
  active: 'In progress',
  drawn: 'Lottery drawn',
  cancelled: 'Cancelled',
} as const

function LotteryEditor({
  lottery,
  onClose,
}: {
  lottery: Lottery | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: lottery
      ? {
          ...lottery,
          draw_time: lottery.draw_at
            ? toDateTimeLocalValue(
                new Date(lottery.draw_at * 1000).toISOString()
              )
            : '',
        }
      : {
          title: '',
          content: '',
          prize: '',
          winner_count: 1,
          draw_mode: 'full',
          target_count: 10,
          draw_time: '',
          require_recharge: false,
          min_invites: 0,
          required_group: '',
          email_notify: true,
        },
  })
  const mode = form.watch('draw_mode')
  const save = useMutation({
    mutationFn: (values: Values) => {
      const { draw_time, ...rules } = values
      return saveLottery({
        ...rules,
        id: lottery?.id ?? 0,
        draw_at: mode === 'full' ? 0 : Math.floor(Date.parse(draw_time) / 1000),
      })
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['lotteries-admin'] })
      onClose()
    },
  })
  return (
    <form
      className='space-y-4 rounded-lg border p-4'
      onSubmit={form.handleSubmit((v) => save.mutate(v))}
    >
      <h3 className='font-semibold'>
        {t(lottery ? 'Edit lottery' : 'Add lottery')}
      </h3>
      <fieldset disabled={save.isPending} className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-1'>
          <Label htmlFor='lottery-title'>{t('Title')}</Label>
          <Input id='lottery-title' {...form.register('title')} />
        </div>
        <div className='space-y-1'>
          <Label htmlFor='lottery-prize'>{t('Prize')}</Label>
          <Input id='lottery-prize' {...form.register('prize')} />
        </div>
        <div className='space-y-1 sm:col-span-2'>
          <Label htmlFor='lottery-content'>{t('Lottery content')}</Label>
          <Textarea id='lottery-content' {...form.register('content')} />
        </div>
        <div className='space-y-1'>
          <Label htmlFor='lottery-mode'>{t('Draw mode')}</Label>
          <NativeSelect id='lottery-mode' {...form.register('draw_mode')}>
            <NativeSelectOption value='full'>
              {t('Draw when full')}
            </NativeSelectOption>
            <NativeSelectOption value='time'>
              {t('Scheduled draw')}
            </NativeSelectOption>
            <NativeSelectOption value='either'>
              {t('Draw at the first condition met')}
            </NativeSelectOption>
          </NativeSelect>
        </div>
        <div className='space-y-1'>
          <Label htmlFor='lottery-winners'>{t('Winner count')}</Label>
          <Input
            id='lottery-winners'
            type='number'
            min={1}
            max={100}
            {...form.register('winner_count', { valueAsNumber: true })}
          />
        </div>
        {mode !== 'time' && (
          <div className='space-y-1'>
            <Label htmlFor='lottery-target'>{t('Participant target')}</Label>
            <Input
              id='lottery-target'
              type='number'
              min={1}
              max={100000}
              {...form.register('target_count', { valueAsNumber: true })}
            />
          </div>
        )}
        {mode !== 'full' && (
          <div className='space-y-1'>
            <Label htmlFor='lottery-time'>{t('Draw time')}</Label>
            <Input
              id='lottery-time'
              type='datetime-local'
              {...form.register('draw_time')}
            />
          </div>
        )}
        <div className='space-y-1'>
          <Label htmlFor='lottery-invites'>{t('Minimum invited users')}</Label>
          <Input
            id='lottery-invites'
            type='number'
            min={0}
            {...form.register('min_invites', { valueAsNumber: true })}
          />
        </div>
        <div className='space-y-1'>
          <Label htmlFor='lottery-group'>{t('Required user group')}</Label>
          <Input id='lottery-group' {...form.register('required_group')} />
        </div>
        <Label className='flex items-center gap-2'>
          <Checkbox
            checked={form.watch('require_recharge')}
            onCheckedChange={(v) =>
              form.setValue('require_recharge', v === true)
            }
          />
          {t('A completed recharge is required')}
        </Label>
        <Label className='flex items-center gap-2'>
          <Checkbox
            checked={form.watch('email_notify')}
            onCheckedChange={(v) => form.setValue('email_notify', v === true)}
          />
          {t('Email winners')}
        </Label>
      </fieldset>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Leave group blank and invites at zero for unrestricted entry. Enabled requirements must all be met.'
        )}
      </p>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Rules are locked after publishing. Scheduled draws award up to the available participant count.'
        )}
      </p>
      {Object.keys(form.formState.errors).length > 0 && (
        <p role='alert' className='text-destructive'>
          {t('Check the lottery rules and future draw time.')}
        </p>
      )}
      <div className='flex gap-2'>
        <Button type='submit' disabled={save.isPending}>
          {t('Save')}
        </Button>
        <Button
          type='button'
          variant='outline'
          disabled={save.isPending}
          onClick={onClose}
        >
          {t('Cancel')}
        </Button>
      </div>
    </form>
  )
}

function LotteryWinners({ lottery }: { lottery: Lottery }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [retryEntry, setRetryEntry] = useState<number>()
  const query = useQuery({
    queryKey: ['lottery-winners', lottery.id],
    queryFn: () => listWinners(lottery.id),
  })
  const update = useMutation({
    mutationFn: ({
      entry,
      action,
    }: {
      entry: number
      action: 'fulfill' | 'retry'
    }) => updateWinner(lottery.id, entry, action),
    onSuccess: () => {
      setRetryEntry(undefined)
      return client.invalidateQueries({
        queryKey: ['lottery-winners', lottery.id],
      })
    },
  })
  return (
    <div className='mt-3 max-h-64 space-y-2 overflow-y-auto'>
      {query.isPending && t('Loading...')}
      {query.isError && t('Failed to load data')}
      {query.isSuccess && query.data.length === 0 && t('No winners')}
      {query.data?.map((entry) => (
        <div
          key={entry.id}
          className='flex flex-wrap items-center gap-2 rounded border p-2 text-xs'
        >
          <span>
            {t('User ID')}: {formatNumber(entry.user_id)}
          </span>
          <span>
            {t('Email')}:{' '}
            {t(
              (
                {
                  sent: 'Sent',
                  failed: 'Failed',
                  none: 'Disabled',
                  uncertain: 'Delivery uncertain',
                } as Record<string, string>
              )[entry.email_status] ?? 'Pending'
            )}
          </span>
          <Button
            size='sm'
            variant='outline'
            disabled={entry.fulfilled || update.isPending}
            onClick={() =>
              update.mutate({ entry: entry.id, action: 'fulfill' })
            }
          >
            {t(entry.fulfilled ? 'Prize delivered' : 'Mark prize delivered')}
          </Button>
          {['failed', 'uncertain'].includes(entry.email_status) && (
            <Button
              size='sm'
              variant='outline'
              disabled={update.isPending}
              onClick={() => setRetryEntry(entry.id)}
            >
              {t('Retry email')}
            </Button>
          )}
        </div>
      ))}
      <ConfirmDialog
        open={retryEntry !== undefined}
        onOpenChange={(open) => {
          if (!open) setRetryEntry(undefined)
        }}
        title={t('Retry email')}
        desc={t(
          'Previous delivery may have succeeded. Check with the winner before resending.'
        )}
        isLoading={update.isPending}
        handleConfirm={() => {
          if (retryEntry !== undefined) {
            update.mutate({ entry: retryEntry, action: 'retry' })
          }
        }}
      />
    </div>
  )
}

export function LotteryAdmin() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [before, setBefore] = useState<number>()
  const [history, setHistory] = useState<(number | undefined)[]>([])
  const query = useQuery({
    queryKey: ['lotteries-admin', before],
    queryFn: () => listLotteries(before),
  })
  const [editing, setEditing] = useState<Lottery | null | undefined>()
  const [winners, setWinners] = useState<number>()
  const [confirm, setConfirm] = useState<{
    id: number
    action: 'publish' | 'cancel'
  }>()
  const change = useMutation({
    mutationFn: ({
      id,
      action,
    }: {
      id: number
      action: 'publish' | 'cancel'
    }) => changeLottery(id, action),
    onSuccess: async () => {
      setConfirm(undefined)
      await Promise.all([
        client.invalidateQueries({ queryKey: ['lotteries-admin'] }),
        client.invalidateQueries({ queryKey: ['lottery-items'] }),
      ])
    },
  })
  return (
    <SettingsSection title={t('Lottery')}>
      <div className='mb-4 flex gap-2'>
        <Button onClick={() => setEditing(null)}>{t('Add lottery')}</Button>
        <Button
          variant='outline'
          onClick={() => {
            void query.refetch()
            void client.invalidateQueries({ queryKey: ['lottery-winners'] })
          }}
        >
          {t('Refresh')}
        </Button>
      </div>
      {editing !== undefined && (
        <LotteryEditor
          key={editing?.id ?? 'new'}
          lottery={editing}
          onClose={() => setEditing(undefined)}
        />
      )}
      <div className='mt-4 max-h-[65vh] space-y-3 overflow-y-auto pr-1'>
        {query.isPending && t('Loading...')}
        {query.isError && t('Failed to load data')}
        {query.isSuccess &&
          query.data.length === 0 &&
          t('No lottery content configured')}
        {query.data?.map((lottery) => (
          <article key={lottery.id} className='rounded-lg border p-4'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <h3 className='min-w-0 font-semibold break-words'>
                {lottery.title}
              </h3>
              <span className='text-muted-foreground text-sm'>
                {t(statuses[lottery.status])}
              </span>
            </div>
            <p className='mt-1 text-sm'>
              {lottery.prize} · {t('Lottery participants')}:{' '}
              {formatNumber(lottery.entry_count)}
            </p>
            <div className='mt-3 flex flex-wrap gap-2'>
              {lottery.status === 'draft' && (
                <>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => setEditing(lottery)}
                  >
                    {t('Edit')}
                  </Button>
                  <Button
                    size='sm'
                    onClick={() =>
                      setConfirm({ id: lottery.id, action: 'publish' })
                    }
                  >
                    {t('Publish')}
                  </Button>
                </>
              )}
              {(lottery.status === 'draft' || lottery.status === 'active') && (
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() =>
                    setConfirm({ id: lottery.id, action: 'cancel' })
                  }
                >
                  {t('Cancel lottery')}
                </Button>
              )}
              {lottery.status === 'drawn' && (
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() =>
                    setWinners(winners === lottery.id ? undefined : lottery.id)
                  }
                >
                  {t('Winning information')}
                </Button>
              )}
            </div>
            {winners === lottery.id && <LotteryWinners lottery={lottery} />}
          </article>
        ))}
      </div>
      <div className='mt-3 flex gap-2'>
        <Button
          variant='outline'
          disabled={history.length === 0}
          onClick={() => {
            setBefore(history.at(-1))
            setHistory(history.slice(0, -1))
          }}
        >
          {t('Previous')}
        </Button>
        <Button
          variant='outline'
          disabled={query.data?.length !== 50}
          onClick={() => {
            setHistory([...history, before])
            setBefore(query.data?.at(-1)?.id)
          }}
        >
          {t('Next')}
        </Button>
      </div>
      <ConfirmDialog
        open={!!confirm}
        onOpenChange={(open) => {
          if (!open) setConfirm(undefined)
        }}
        title={t(confirm?.action === 'publish' ? 'Publish' : 'Cancel lottery')}
        desc={t(
          confirm?.action === 'publish'
            ? 'Rules are locked after publishing. Scheduled draws award up to the available participant count.'
            : 'Cancellation closes entry without drawing winners.'
        )}
        isLoading={change.isPending}
        handleConfirm={() => {
          if (confirm) change.mutate(confirm)
        }}
      />
    </SettingsSection>
  )
}
