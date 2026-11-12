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
import { useMutation } from '@tanstack/react-query'
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Collapsible,
  CollapsibleTrigger,
  CollapsibleContent,
} from '@/components/ui/collapsible'
import { Field, FieldLabel } from '@/components/ui/field'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { handleServerError } from '@/lib/handle-server-error'

import { saveGuardianAccountStatus } from '../../api'
import { useAccountStatusLabels } from '../../hooks/use-account-status-labels'
import type { GuardianAccountFlags, GuardianAccountStatus } from '../../types'

export function GuardianAccountStatusDialog(props: {
  user: GuardianAccountStatus
  onClose: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const labels = useAccountStatusLabels()
  const [flags, setFlags] = useState<GuardianAccountFlags>({
    banned: props.user.banned,
    phone_required: props.user.phone_required,
    kyc_required: props.user.kyc_required,
  })
  const [historicalCategory, setHistoricalCategory] = useState(
    props.user.historical_category || ''
  )
  const save = useMutation({
    mutationFn: () =>
      saveGuardianAccountStatus(props.user.user_id, {
        ...flags,
        historical_category: historicalCategory,
      }),
    onSuccess: () => {
      toast.success(t('Account status saved'))
      props.onSaved()
      props.onClose()
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save account status')),
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.onClose()
      }}
      title={t('Record account status')}
      description={`${props.user.username} (#${props.user.user_id})`}
      footer={
        <>
          <Button
            variant='outline'
            disabled={save.isPending}
            onClick={props.onClose}
          >
            {t('Cancel')}
          </Button>
          <Button disabled={save.isPending} onClick={() => save.mutate()}>
            {save.isPending ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        <p className='text-muted-foreground text-sm'>
          {t('Select current issues. Leave all unchecked if none.')}
        </p>
        <div className='space-y-3'>
          {labels.map((item) => (
            <Field key={item.key} orientation='horizontal'>
              <Checkbox
                id={`guardian-account-${item.key}`}
                checked={flags[item.key]}
                disabled={save.isPending}
                onCheckedChange={(checked) =>
                  setFlags((previous) => ({
                    ...previous,
                    [item.key]: !!checked,
                  }))
                }
              />
              <FieldLabel htmlFor={`guardian-account-${item.key}`}>
                {item.label}
              </FieldLabel>
            </Field>
          ))}
        </div>
        <Collapsible className='border-t pt-3'>
          <CollapsibleTrigger className='group text-muted-foreground flex w-full items-center justify-between text-sm'>
            {t('Historical user link (optional)')}
            <ChevronDown className='size-4 transition-transform group-aria-expanded:rotate-180' />
          </CollapsibleTrigger>
          <CollapsibleContent className='pt-3'>
            <Field>
              <FieldLabel htmlFor='guardian-history-link'>
                {t('Historical user link')}
              </FieldLabel>
              <NativeSelect
                id='guardian-history-link'
                value={historicalCategory}
                disabled={save.isPending}
                onChange={(event) => setHistoricalCategory(event.target.value)}
              >
                <NativeSelectOption value=''>
                  {t('Not linked to historical users')}
                </NativeSelectOption>
                <NativeSelectOption value='healthy'>
                  {t('No ban, no KYC, no phone verification')}
                </NativeSelectOption>
                <NativeSelectOption value='banned_only'>
                  {t('Ban only')}
                </NativeSelectOption>
                <NativeSelectOption value='kyc_only'>
                  {t('KYC only')}
                </NativeSelectOption>
                <NativeSelectOption value='phone_only'>
                  {t('Phone verification only')}
                </NativeSelectOption>
                <NativeSelectOption value='banned_kyc'>
                  {t('Ban and KYC only')}
                </NativeSelectOption>
                <NativeSelectOption value='banned_phone'>
                  {t('Ban and phone verification only')}
                </NativeSelectOption>
                <NativeSelectOption value='kyc_phone'>
                  {t('KYC and phone verification only')}
                </NativeSelectOption>
                <NativeSelectOption value='all_three'>
                  {t('All three conditions flagged')}
                </NativeSelectOption>
              </NativeSelect>
            </Field>
          </CollapsibleContent>
        </Collapsible>
      </div>
    </Dialog>
  )
}
