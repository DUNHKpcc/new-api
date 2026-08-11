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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  BadgeDollarSign,
  CheckCircle2,
  ChevronDown,
  Copy,
  Gift,
  RefreshCw,
  SendToBack,
  Share2,
  Users,
  WalletCards,
} from 'lucide-react'
import { useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { getCurrencyLabel } from '@/lib/currency'
import { formatTimestamp, parseQuotaFromDollars } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  activateAffiliate,
  getAffiliateCommissions,
  getAffiliateOverview,
  transferAffiliateCommissions,
  transferAffiliateInviteRewards,
} from './api'
import {
  createAffiliateIdempotencyKey,
  absoluteAffiliateLink,
  formatAffiliateMinor,
  formatAffiliateQuota,
  formatAffiliateRate,
  hasPositiveAffiliateInteger,
} from './lib'
import type { AffiliateCommission } from './types'

const COMMISSION_PAGE_SIZE = 20
const COMMISSION_STATUS_LABELS: Record<AffiliateCommission['status'], string> =
  {
    pending: 'Commission pending',
    available: 'Commission available',
    transferred: 'Commission transferred',
    reversed: 'Commission reversed',
  }

function CommissionStatusBadge(props: {
  status: AffiliateCommission['status']
}) {
  const { t } = useTranslation()
  const variant = props.status === 'reversed' ? 'destructive' : 'secondary'
  return (
    <Badge variant={variant}>{t(COMMISSION_STATUS_LABELS[props.status])}</Badge>
  )
}

function AffiliateStat(props: {
  label: string
  value: string
  icon: typeof Gift
}) {
  const Icon = props.icon
  return (
    <div className='bg-muted/35 flex min-w-0 items-center gap-3 rounded-lg p-3'>
      <Icon
        className='text-muted-foreground size-4 shrink-0'
        aria-hidden='true'
      />
      <div className='min-w-0'>
        <p className='text-muted-foreground text-xs break-words'>
          {props.label}
        </p>
        <p className='font-semibold break-all tabular-nums'>{props.value}</p>
      </div>
    </div>
  )
}

export function CashWithdrawalComingSoon() {
  const { t } = useTranslation()

  return (
    <div
      data-slot='affiliate-cash-withdrawal-coming-soon'
      className='border-border/60 grid gap-3 border-t pt-4'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <WalletCards
          className='text-muted-foreground size-4'
          aria-hidden='true'
        />
        <span className='text-sm font-medium'>
          {t('Alipay cash withdrawal')}
        </span>
        <Badge variant='secondary'>{t('Coming soon')}</Badge>
      </div>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Cash withdrawal uses the verified payment amount; transfer to balance uses the purchased balance.'
        )}
      </p>
      <Button type='button' variant='outline' disabled>
        <WalletCards aria-hidden='true' />
        {t('Alipay withdrawal is not available yet')}
      </Button>
    </div>
  )
}

export function AffiliateDetailsDisclosure(props: { children: ReactNode }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return (
    <>
      <Button
        type='button'
        variant='ghost'
        size='sm'
        className='border-border/60 w-full justify-between border'
        aria-expanded={open}
        aria-controls='affiliate-commission-summary-details'
        onClick={() => setOpen((value) => !value)}
      >
        {open ? t('Collapse') : t('View details')}
        <ChevronDown
          className={cn('size-4 transition-transform', open && 'rotate-180')}
          aria-hidden='true'
        />
      </Button>
      {open ? (
        <div id='affiliate-commission-summary-details' className='grid gap-3'>
          {props.children}
        </div>
      ) : null}
    </>
  )
}

export function AffiliateCommissionList(props: {
  items: AffiliateCommission[]
  loading: boolean
}) {
  const { t } = useTranslation()
  if (props.loading) {
    return <Skeleton className='h-52 w-full' />
  }
  if (props.items.length === 0) {
    return (
      <div className='text-muted-foreground flex min-h-40 items-center justify-center text-sm'>
        {t('No commissions yet')}
      </div>
    )
  }

  return (
    <div
      data-slot='affiliate-commission-scroll-area'
      className='max-h-[min(58svh,36rem)] [scrollbar-gutter:stable] overflow-y-auto overscroll-contain pr-1'
    >
      <div className='hidden md:block'>
        <Table className='min-w-[1240px]'>
          <TableHeader>
            <TableRow className='bg-muted/40 hover:bg-muted/40'>
              <TableHead>{t('Username')}</TableHead>
              <TableHead>{t('Actual payment')}</TableHead>
              <TableHead>{t('Purchased balance')}</TableHead>
              <TableHead>{t('Commission Rate')}</TableHead>
              <TableHead>{t('Cash withdrawal value')}</TableHead>
              <TableHead>{t('Balance reward')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Available At')}</TableHead>
              <TableHead>{t('Created At')}</TableHead>
              <TableHead>{t('Reversal reason')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {props.items.map((item) => (
              <TableRow key={item.id}>
                <TableCell>{item.referred_user}</TableCell>
                <TableCell>
                  {formatAffiliateMinor(
                    item.paid_amount_minor,
                    item.paid_currency
                  )}
                </TableCell>
                <TableCell>
                  {formatAffiliateQuota(item.purchased_quota)}
                </TableCell>
                <TableCell>
                  {formatAffiliateRate(item.commission_rate_bps)}
                </TableCell>
                <TableCell>
                  {formatAffiliateMinor(
                    item.commission_amount_minor,
                    item.paid_currency
                  )}
                </TableCell>
                <TableCell>{formatAffiliateQuota(item.reward_quota)}</TableCell>
                <TableCell>
                  <CommissionStatusBadge status={item.status} />
                </TableCell>
                <TableCell>
                  {item.available_at ? formatTimestamp(item.available_at) : '-'}
                </TableCell>
                <TableCell>{formatTimestamp(item.created_at)}</TableCell>
                <TableCell>
                  <div className='max-w-64 min-w-40'>
                    <span className='break-words whitespace-pre-wrap'>
                      {item.status === 'reversed'
                        ? item.reverse_reason || '-'
                        : '-'}
                    </span>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <div className='divide-y md:hidden'>
        {props.items.map((item) => (
          <div key={item.id} className='grid gap-2 py-3'>
            <div className='flex items-center justify-between gap-3'>
              <div className='min-w-0'>
                <span className='text-muted-foreground block text-xs'>
                  {t('Username')}
                </span>
                <span className='font-medium'>{item.referred_user}</span>
              </div>
              <CommissionStatusBadge status={item.status} />
            </div>
            <div className='text-muted-foreground grid grid-cols-2 gap-x-3 gap-y-1 text-xs'>
              <span>{t('Actual payment')}</span>
              <span className='text-foreground min-w-0 text-right break-all'>
                {formatAffiliateMinor(
                  item.paid_amount_minor,
                  item.paid_currency
                )}
              </span>
              <span>{t('Purchased balance')}</span>
              <span className='text-foreground min-w-0 text-right break-all'>
                {formatAffiliateQuota(item.purchased_quota)}
              </span>
              <span>{t('Cash withdrawal value')}</span>
              <span className='text-foreground min-w-0 text-right break-all'>
                {formatAffiliateMinor(
                  item.commission_amount_minor,
                  item.paid_currency
                )}
              </span>
              <span>{t('Balance reward')}</span>
              <span className='text-foreground min-w-0 text-right break-all'>
                {formatAffiliateQuota(item.reward_quota)}
              </span>
              <span>{t('Available At')}</span>
              <span className='text-foreground text-right'>
                {item.available_at ? formatTimestamp(item.available_at) : '-'}
              </span>
              {item.status === 'reversed' ? (
                <>
                  <span>{t('Reversal reason')}</span>
                  <span className='text-foreground min-w-0 text-right break-words whitespace-pre-wrap'>
                    {item.reverse_reason || '-'}
                  </span>
                </>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export function AffiliateCenter() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [signupTransferAmount, setSignupTransferAmount] = useState('')
  const [page, setPage] = useState(1)
  const signupTransferKeyRef = useRef<string | null>(null)
  const commissionTransferKeyRef = useRef<string | null>(null)
  const overviewQuery = useQuery({
    queryKey: ['affiliate-overview'],
    queryFn: getAffiliateOverview,
  })
  const commissionsQuery = useQuery({
    queryKey: ['affiliate-commissions', page],
    queryFn: () => getAffiliateCommissions(page, COMMISSION_PAGE_SIZE),
    enabled: overviewQuery.data?.success === true,
  })
  const { copyToClipboard } = useCopyToClipboard({
    successMessage: t('Copied to clipboard'),
    errorMessage: t('Failed to copy invitation link'),
  })
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['affiliate-overview'] })
    void queryClient.invalidateQueries({ queryKey: ['affiliate-commissions'] })
  }
  const activateMutation = useMutation({
    mutationFn: activateAffiliate,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t('Affiliate program activated'))
        refresh()
        return
      }
      toast.error(result.message || t('Failed to activate affiliate program'))
    },
    onError: () => toast.error(t('Failed to activate affiliate program')),
  })
  const signupTransferMutation = useMutation({
    mutationFn: () => {
      signupTransferKeyRef.current ??= createAffiliateIdempotencyKey()
      return transferAffiliateInviteRewards(
        String(signupTransferQuota),
        signupTransferKeyRef.current
      )
    },
    onSuccess: (result) => {
      if (result.success) {
        signupTransferKeyRef.current = null
        toast.success(t('Transfer successful'))
        setSignupTransferAmount('')
        refresh()
        return
      }
      signupTransferKeyRef.current = null
      toast.error(result.message || t('Transfer failed'))
    },
    onError: () => toast.error(t('Transfer failed')),
  })
  const commissionTransferMutation = useMutation({
    mutationFn: () => {
      commissionTransferKeyRef.current ??= createAffiliateIdempotencyKey()
      return transferAffiliateCommissions(commissionTransferKeyRef.current)
    },
    onSuccess: (result) => {
      if (result.success) {
        commissionTransferKeyRef.current = null
        toast.success(t('Transfer successful'))
        refresh()
        return
      }
      commissionTransferKeyRef.current = null
      toast.error(result.message || t('Transfer failed'))
    },
    onError: () => toast.error(t('Transfer failed')),
  })

  const overview = overviewQuery.data?.data
  const commissions = commissionsQuery.data?.data
  const commissionLoadFailed =
    commissionsQuery.isError ||
    (commissionsQuery.data && !commissionsQuery.data.success)
  const signupTransferNumericAmount = Number(signupTransferAmount)
  const signupTransferQuota = Number.isFinite(signupTransferNumericAmount)
    ? parseQuotaFromDollars(signupTransferNumericAmount)
    : 0
  const signupAvailableQuota = Number(
    overview?.signup_rewards.available_quota ?? 0
  )
  const canSubmitSignupTransfer =
    signupTransferQuota > 0 && signupTransferQuota <= signupAvailableQuota

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Affiliate Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='icon-sm'
          onClick={refresh}
          aria-label={t('Refresh')}
        >
          <RefreshCw aria-hidden='true' />
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-7xl flex-col gap-4'>
          {overviewQuery.isLoading ? (
            <Skeleton className='h-52 w-full' />
          ) : null}
          {overviewQuery.isError ||
          (overviewQuery.data && !overviewQuery.data.success) ? (
            <Card>
              <CardContent className='flex min-h-40 flex-col items-center justify-center gap-3'>
                <p className='text-muted-foreground text-sm'>
                  {t('Failed to load affiliate overview')}
                </p>
                <Button
                  variant='outline'
                  onClick={() => void overviewQuery.refetch()}
                >
                  {t('Retry')}
                </Button>
              </CardContent>
            </Card>
          ) : null}
          {overview ? (
            <>
              <Card>
                <CardHeader>
                  <CardTitle className='flex items-center gap-2'>
                    <Share2 className='size-4' />
                    {t('Invite friends')}
                  </CardTitle>
                  <CardDescription>
                    {t(
                      'Share your invitation link and earn registration rewards when available.'
                    )}
                  </CardDescription>
                </CardHeader>
                <CardContent className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end'>
                  <div className='grid gap-2'>
                    <label
                      className='text-sm font-medium'
                      htmlFor='affiliate-link'
                    >
                      {t('Invitation Link')}
                    </label>
                    <div className='flex gap-2'>
                      <Input
                        id='affiliate-link'
                        readOnly
                        value={absoluteAffiliateLink(overview.invite.link)}
                        className='font-mono text-xs'
                      />
                      <Button
                        variant='outline'
                        size='icon-sm'
                        aria-label={t('Copy invitation link')}
                        onClick={() => {
                          const link = absoluteAffiliateLink(
                            overview.invite.link
                          )
                          void copyToClipboard(link)
                        }}
                      >
                        <Copy aria-hidden='true' />
                      </Button>
                    </div>
                  </div>
                  <AffiliateStat
                    label={t('Invited Users')}
                    value={String(overview.invite.count)}
                    icon={Users}
                  />
                </CardContent>
              </Card>

              <div className='grid gap-4 xl:grid-cols-2'>
                <Card className='h-full'>
                  <CardHeader>
                    <CardTitle className='flex items-center gap-2'>
                      <Gift className='size-4' />
                      {t('Registration rewards')}
                    </CardTitle>
                    <CardDescription>
                      {overview.signup_rewards.enabled
                        ? t(
                            'Rewards are available after referred users register.'
                          )
                        : t('Registration rewards are currently disabled.')}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='grid gap-4'>
                    <div className='grid grid-cols-2 gap-3'>
                      <AffiliateStat
                        label={t('Inviter Reward')}
                        value={formatAffiliateQuota(
                          overview.signup_rewards.inviter_reward_quota
                        )}
                        icon={Gift}
                      />
                      <AffiliateStat
                        label={t('Invitee Reward')}
                        value={formatAffiliateQuota(
                          overview.signup_rewards.invitee_reward_quota
                        )}
                        icon={Gift}
                      />
                      <AffiliateStat
                        label={t('Available')}
                        value={formatAffiliateQuota(
                          overview.signup_rewards.available_quota
                        )}
                        icon={Gift}
                      />
                      <AffiliateStat
                        label={t('Lifetime')}
                        value={formatAffiliateQuota(
                          overview.signup_rewards.lifetime_quota
                        )}
                        icon={CheckCircle2}
                      />
                    </div>
                    <div className='grid gap-2 sm:grid-cols-[1fr_auto]'>
                      <Input
                        type='number'
                        min='0'
                        step='any'
                        value={signupTransferAmount}
                        onChange={(event) =>
                          setSignupTransferAmount(event.currentTarget.value)
                        }
                        placeholder={`${t('Transfer Amount')} (${getCurrencyLabel()})`}
                        aria-label={`${t('Transfer Amount')} (${getCurrencyLabel()})`}
                      />
                      <Button
                        disabled={
                          !canSubmitSignupTransfer ||
                          signupTransferMutation.isPending
                        }
                        onClick={() => signupTransferMutation.mutate()}
                      >
                        <SendToBack aria-hidden='true' />
                        {t('Transfer to Balance')}
                      </Button>
                    </div>
                  </CardContent>
                </Card>

                <Card className='h-full'>
                  <CardHeader>
                    <CardTitle className='flex items-center gap-2'>
                      <BadgeDollarSign className='size-4' />
                      {t('Top-up commission')}
                    </CardTitle>
                    <CardDescription>
                      {t(
                        'Cash withdrawal uses the verified payment amount; transfer to balance uses the purchased balance.'
                      )}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='grid gap-4'>
                    {!overview.program.enabled ? (
                      <p className='text-muted-foreground text-sm'>
                        {t(
                          'The affiliate commission program is currently disabled.'
                        )}
                      </p>
                    ) : null}
                    <div className='grid grid-cols-2 gap-3'>
                      <AffiliateStat
                        label={t('Verified online payments')}
                        value={formatAffiliateMinor(
                          overview.program.verified_amount_minor,
                          overview.program.currency
                        )}
                        icon={BadgeDollarSign}
                      />
                      <AffiliateStat
                        label={t('Remaining qualification amount')}
                        value={formatAffiliateMinor(
                          overview.program.remaining_amount_minor,
                          overview.program.currency
                        )}
                        icon={BadgeDollarSign}
                      />
                      <AffiliateStat
                        label={t('Commission Rate')}
                        value={formatAffiliateRate(
                          overview.program.commission_rate_bps
                        )}
                        icon={BadgeDollarSign}
                      />
                      <AffiliateStat
                        label={t('Commission wait days')}
                        value={String(overview.program.commission_wait_days)}
                        icon={BadgeDollarSign}
                      />
                    </div>
                    <AffiliateDetailsDisclosure>
                      <div className='grid grid-cols-2 gap-3'>
                        {overview.program.enabled &&
                        overview.program.status === 'active' &&
                        overview.program.access !== 'deny' ? (
                          <>
                            <AffiliateStat
                              label={t('Balance rewards available')}
                              value={formatAffiliateQuota(
                                overview.commissions.available_quota
                              )}
                              icon={BadgeDollarSign}
                            />
                            <AffiliateStat
                              label={t('Balance rewards pending')}
                              value={formatAffiliateQuota(
                                overview.commissions.pending_quota
                              )}
                              icon={BadgeDollarSign}
                            />
                            <AffiliateStat
                              label={t('Lifetime balance rewards')}
                              value={formatAffiliateQuota(
                                overview.commissions.lifetime_quota
                              )}
                              icon={BadgeDollarSign}
                            />
                            <AffiliateStat
                              label={t('Transferred balance rewards')}
                              value={formatAffiliateQuota(
                                overview.commissions.transferred_quota
                              )}
                              icon={BadgeDollarSign}
                            />
                            <AffiliateStat
                              label={t('Referred verified payments')}
                              value={formatAffiliateMinor(
                                overview.commissions.referred_paid_amount_minor,
                                overview.program.currency
                              )}
                              icon={BadgeDollarSign}
                            />
                            <AffiliateStat
                              label={t('Balance reward debt')}
                              value={formatAffiliateQuota(
                                overview.program.commission_debt_quota
                              )}
                              icon={BadgeDollarSign}
                            />
                          </>
                        ) : null}
                      </div>
                      {overview.program.enabled &&
                      overview.program.status === 'active' &&
                      overview.program.access !== 'deny' ? (
                        <Button
                          disabled={
                            !hasPositiveAffiliateInteger(
                              overview.commissions.available_quota
                            ) || commissionTransferMutation.isPending
                          }
                          onClick={() => commissionTransferMutation.mutate()}
                        >
                          <SendToBack aria-hidden='true' />
                          {t('Transfer commissions to balance')}
                        </Button>
                      ) : null}
                    </AffiliateDetailsDisclosure>
                    <CashWithdrawalComingSoon />
                    {overview.program.access === 'deny' ||
                    overview.program.status === 'suspended' ? (
                      <p className='text-muted-foreground text-sm'>
                        {t('Affiliate commission access is suspended.')}
                      </p>
                    ) : null}
                    {overview.program.enabled &&
                    overview.program.status === 'inactive' ? (
                      <div className='grid gap-3'>
                        <p className='text-sm'>
                          {t(
                            'Verified top-up progress: {{current}} / {{threshold}}',
                            {
                              current: formatAffiliateMinor(
                                overview.program.verified_amount_minor,
                                overview.program.currency
                              ),
                              threshold: formatAffiliateMinor(
                                overview.program.qualification_threshold_minor,
                                overview.program.currency
                              ),
                            }
                          )}
                        </p>
                        <Button
                          disabled={
                            !overview.program.eligible ||
                            activateMutation.isPending
                          }
                          onClick={() => activateMutation.mutate()}
                        >
                          {t('Activate affiliate program')}
                        </Button>
                      </div>
                    ) : null}
                  </CardContent>
                </Card>
              </div>

              {overview.program.status !== 'inactive' ||
              (commissions?.total ?? 0) > 0 ||
              commissionLoadFailed ? (
                <Card>
                  <CardHeader>
                    <CardTitle>{t('Commission details')}</CardTitle>
                    <CardDescription>
                      {t(
                        'Commission availability is subject to the configured waiting period.'
                      )}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {commissionLoadFailed ? (
                      <div className='flex justify-center gap-2 py-6'>
                        <span className='text-muted-foreground text-sm'>
                          {t('Failed to load commissions')}
                        </span>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => void commissionsQuery.refetch()}
                        >
                          {t('Retry')}
                        </Button>
                      </div>
                    ) : (
                      <AffiliateCommissionList
                        items={commissions?.items ?? []}
                        loading={commissionsQuery.isLoading}
                      />
                    )}
                    {commissions && commissions.total > COMMISSION_PAGE_SIZE ? (
                      <div className='mt-4 flex items-center justify-end gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={page <= 1}
                          onClick={() => setPage((value) => value - 1)}
                        >
                          {t('Previous')}
                        </Button>
                        <span className='text-muted-foreground text-sm'>
                          {page}
                        </span>
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={
                            page * COMMISSION_PAGE_SIZE >= commissions.total
                          }
                          onClick={() => setPage((value) => value + 1)}
                        >
                          {t('Next')}
                        </Button>
                      </div>
                    ) : null}
                  </CardContent>
                </Card>
              ) : null}
            </>
          ) : null}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
