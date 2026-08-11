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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { formatAffiliateQuota } from '@/features/affiliate/lib'
import { formatMinorCurrency, formatTimestamp } from '@/lib/format'

import { updateAffiliateAccess } from '../../api'
import type { User } from '../../types'

export function AffiliateAccessDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [access, setAccess] = useState<'inherit' | 'allow' | 'deny'>(
    (props.user.affiliate_summary?.access as 'inherit' | 'allow' | 'deny') ||
      'inherit'
  )
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    if (!props.open) return
    setAccess(
      (props.user.affiliate_summary?.access as 'inherit' | 'allow' | 'deny') ||
        'inherit'
    )
    setReason('')
  }, [props.open, props.user.affiliate_summary?.access])
  const save = async () => {
    try {
      setSaving(true)
      const result = await updateAffiliateAccess(props.user.id, access, reason)
      if (result.success) {
        toast.success(t('Affiliate access updated'))
        props.onSuccess()
        props.onOpenChange(false)
        return
      }
      toast.error(result.message || t('Failed to update affiliate access'))
    } catch {
      toast.error(t('Failed to update affiliate access'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Affiliate access')}</DialogTitle>
          <DialogDescription>
            {t('Set affiliate access for {{username}}.', {
              username: props.user.username,
            })}
          </DialogDescription>
        </DialogHeader>
        <div className='grid gap-4'>
          {props.user.affiliate_summary ? (
            <div className='grid grid-cols-2 gap-3 border-y py-3 text-sm sm:grid-cols-3'>
              <div className='min-w-0'>
                <p className='text-muted-foreground text-xs'>
                  {t('Verified online payments')}
                </p>
                <p className='font-medium break-all tabular-nums'>
                  {formatMinorCurrency(
                    props.user.affiliate_summary.verified_amount_minor,
                    props.user.affiliate_summary.currency
                  )}
                </p>
              </div>
              <div className='min-w-0'>
                <p className='text-muted-foreground text-xs'>
                  {t('Activated At')}
                </p>
                <p className='font-medium'>
                  {formatTimestamp(props.user.affiliate_summary.activated_at)}
                </p>
              </div>
              <div className='min-w-0'>
                <p className='text-muted-foreground text-xs'>
                  {t('Lifetime commissions')}
                </p>
                <p className='font-medium break-all tabular-nums'>
                  {formatAffiliateQuota(
                    props.user.affiliate_summary.lifetime_commission_quota
                  )}
                </p>
              </div>
            </div>
          ) : null}
          <Select
            value={access}
            onValueChange={(value) =>
              value && setAccess(value as 'inherit' | 'allow' | 'deny')
            }
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='inherit'>
                {t('Follow system threshold')}
              </SelectItem>
              <SelectItem value='allow'>
                {t('Allow without threshold')}
              </SelectItem>
              <SelectItem value='deny'>{t('Deny affiliate access')}</SelectItem>
            </SelectContent>
          </Select>
          {props.user.affiliate_summary?.access !== 'inherit' &&
          (props.user.affiliate_summary?.activated_at ?? 0) > 0 &&
          access === 'inherit' ? (
            <p role='alert' className='text-destructive text-sm'>
              {t(
                'If the user has not met the current recharge threshold, this will lock the affiliate program and pause commission transfers.'
              )}
            </p>
          ) : null}
          <Textarea
            value={reason}
            onChange={(event) => setReason(event.currentTarget.value)}
            minLength={1}
            maxLength={255}
            aria-label={t('Reason')}
            placeholder={t('Reason (1-255 characters)')}
          />
        </div>
        <DialogFooter>
          <Button disabled={saving || reason.trim().length < 1} onClick={save}>
            {t('Save Changes')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
