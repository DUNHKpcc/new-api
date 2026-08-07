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

const versionUpdateSchema = z.object({
  VersionUpdateDetails: z.string().optional(),
})

type VersionUpdateFormValues = z.infer<typeof versionUpdateSchema>

type VersionUpdateSectionProps = {
  defaultValue: string
}

export function VersionUpdateSection(props: VersionUpdateSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<VersionUpdateFormValues>({
    resolver: zodResolver(versionUpdateSchema),
    defaultValues: {
      VersionUpdateDetails: props.defaultValue ?? '',
    },
  })

  useEffect(() => {
    form.reset({ VersionUpdateDetails: props.defaultValue ?? '' })
  }, [form, props.defaultValue])

  const onSubmit = async (values: VersionUpdateFormValues) => {
    const normalized = values.VersionUpdateDetails ?? ''
    if (normalized === (props.defaultValue ?? '')) return

    await updateOption.mutateAsync({
      key: 'VersionUpdateDetails',
      value: normalized,
    })
  }

  return (
    <SettingsSection title={t('Version update details')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save changes'
          />
          <FormField
            control={form.control}
            name='VersionUpdateDetails'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Content')}</FormLabel>
                <FormControl>
                  <Textarea rows={14} {...field} />
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
