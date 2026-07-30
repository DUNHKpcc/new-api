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
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Textarea } from '@/components/ui/textarea'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const discountNoticeSchema = z.object({
  DiscountNotice: z.string().optional(),
})

type DiscountNoticeFormValues = z.infer<typeof discountNoticeSchema>

type DiscountNoticeSectionProps = {
  defaultValue: string
}

export function DiscountNoticeSection(props: DiscountNoticeSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<DiscountNoticeFormValues>({
    resolver: zodResolver(discountNoticeSchema),
    defaultValues: {
      DiscountNotice: props.defaultValue ?? '',
    },
  })

  useEffect(() => {
    form.reset({ DiscountNotice: props.defaultValue ?? '' })
  }, [form, props.defaultValue])

  const onSubmit = async (values: DiscountNoticeFormValues) => {
    const normalized = values.DiscountNotice ?? ''
    if (normalized === (props.defaultValue ?? '')) {
      return
    }
    await updateOption.mutateAsync({
      key: 'DiscountNotice',
      value: normalized,
    })
  }

  return (
    <SettingsSection title={t('Discount Notice')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save discount notice'
          />
          <FormField
            control={form.control}
            name='DiscountNotice'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Discount notice content')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={6}
                    placeholder={t(
                      'Example: GPT-5 is 20% off through August 31.'
                    )}
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
