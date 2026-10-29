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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'

import { updateGuardianContact } from '../api'
import { GUARDIAN_QUERY_KEYS } from '../constants'
import { guardianContactSchema } from '../lib/schema'
import type { GuardianOrder, GuardianContactUpdate } from '../types'
import { GuardianContactFields } from './contact-fields'

export function EditGuardianContact(props: { order: GuardianOrder }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button
        className='mt-4'
        variant='outline'
        size='sm'
        onClick={() => setOpen(true)}
      >
        {t('Edit contact and note')}
      </Button>
      {open && (
        <ContactDialog order={props.order} onClose={() => setOpen(false)} />
      )}
    </>
  )
}

function ContactDialog(props: { order: GuardianOrder; onClose: () => void }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const form = useForm<GuardianContactUpdate>({
    resolver: zodResolver(guardianContactSchema),
    defaultValues: {
      contact_type: props.order.contact_type,
      contact: props.order.contact,
      note: props.order.note,
    },
  })
  const save = useMutation({
    mutationFn: (value: GuardianContactUpdate) =>
      updateGuardianContact(props.order.id, value),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: GUARDIAN_QUERY_KEYS.myOrders })
      props.onClose()
    },
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.onClose()
      }}
      title={t('Edit contact and note')}
      footer={
        <>
          <Button
            variant='outline'
            disabled={save.isPending}
            onClick={props.onClose}
          >
            {t('Cancel')}
          </Button>
          <Button
            disabled={save.isPending}
            onClick={form.handleSubmit((value) => save.mutate(value))}
          >
            {save.isPending ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <FormProvider {...form}>
        <form
          className='space-y-4'
          onSubmit={form.handleSubmit((value) => save.mutate(value))}
        >
          <GuardianContactFields />
        </form>
      </FormProvider>
    </Dialog>
  )
}
