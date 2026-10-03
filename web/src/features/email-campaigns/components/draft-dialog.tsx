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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import { saveEmailDraft } from '../api'
import { emailDraftSchema } from '../lib/schema'
import type { EmailCampaign, EmailDraft } from '../types'

export function EmailDraftDialog(props: {
  campaign?: EmailCampaign
  onClose: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const form = useForm<EmailDraft>({
    resolver: zodResolver(emailDraftSchema),
    defaultValues: props.campaign ?? {
      subject: '',
      body: '',
      category: 'promotion',
      group: '',
    },
  })
  const save = useMutation({
    mutationFn: (draft: EmailDraft) =>
      saveEmailDraft(draft, props.campaign?.id),
    onSuccess: () => {
      toast.success(t('Email draft saved'))
      props.onSaved()
      props.onClose()
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save email draft')),
  })
  const errors = form.formState.errors
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.onClose()
      }}
      title={
        props.campaign ? t('Edit email draft') : t('Create email campaign')
      }
      description={t(
        'Drafts are not sent until you preview recipients and confirm queueing.'
      )}
      footer={
        <>
          <Button
            variant='outline'
            onClick={props.onClose}
            disabled={save.isPending}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form='email-draft' disabled={save.isPending}>
            {save.isPending ? t('Saving...') : t('Save draft')}
          </Button>
        </>
      }
    >
      <form
        id='email-draft'
        className='space-y-4'
        noValidate
        onSubmit={form.handleSubmit((draft) => save.mutate(draft))}
      >
        <div className='space-y-2'>
          <Label htmlFor='email-category'>{t('Category')}</Label>
          <NativeSelect id='email-category' {...form.register('category')}>
            <NativeSelectOption value='promotion'>
              {t('Promotional emails')}
            </NativeSelectOption>
            <NativeSelectOption value='platform'>
              {t('Platform updates')}
            </NativeSelectOption>
          </NativeSelect>
        </div>
        <div className='space-y-2'>
          <Label htmlFor='email-group'>{t('User group filter')}</Label>
          <Input
            id='email-group'
            maxLength={64}
            {...form.register('group')}
            aria-describedby='email-group-help'
          />
          <p id='email-group-help' className='text-muted-foreground text-xs'>
            {t(
              'Leave empty for all enabled users with an email address. Subscription preferences are always respected.'
            )}
          </p>
        </div>
        <div className='space-y-2'>
          <Label htmlFor='email-subject'>{t('Email subject')}</Label>
          <Input
            id='email-subject'
            maxLength={200}
            {...form.register('subject')}
            aria-invalid={!!errors.subject}
            aria-describedby={
              errors.subject ? 'email-subject-error' : undefined
            }
          />
          {errors.subject && (
            <p
              id='email-subject-error'
              role='alert'
              className='text-destructive text-sm'
            >
              {t(errors.subject.message ?? '')}
            </p>
          )}
        </div>
        <div className='space-y-2'>
          <Label htmlFor='email-body'>{t('Email message')}</Label>
          <Textarea
            id='email-body'
            rows={10}
            maxLength={20000}
            {...form.register('body')}
            aria-invalid={!!errors.body}
            aria-describedby={errors.body ? 'email-body-error' : undefined}
          />
          {errors.body && (
            <p
              id='email-body-error'
              role='alert'
              className='text-destructive text-sm'
            >
              {t(errors.body.message ?? '')}
            </p>
          )}
          <p className='text-muted-foreground text-xs'>
            {t('Plain text only. An unsubscribe link is added automatically.')}
          </p>
        </div>
        {save.isError && (
          <p role='alert' className='text-destructive text-sm'>
            {t('Failed to save email draft')}
          </p>
        )}
      </form>
    </Dialog>
  )
}
