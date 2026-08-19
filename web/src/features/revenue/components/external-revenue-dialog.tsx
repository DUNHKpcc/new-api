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
import { Loader2 } from 'lucide-react'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

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
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

import { createExternalRevenue, updateExternalRevenue } from '../api'
import { decimalAmountToMinor, minorAmountToDecimal } from '../lib/analytics'
import type { ExternalRevenueRecord, RevenueSource } from '../types'

type ExternalRevenueDialogProps = {
  open: boolean
  record: ExternalRevenueRecord | null
  onOpenChange: (open: boolean) => void
  onSaved: () => void
}

type FormValues = {
  source: RevenueSource
  sourceLabel: string
  externalOrderNo: string
  amount: string
  currency: string
  occurredAt: string
  note: string
}

function localDatetimeValue(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
    .toISOString()
    .slice(0, 16)
}

function currentDatetimeValue(): string {
  return localDatetimeValue(Math.floor(Date.now() / 1000))
}

export function ExternalRevenueDialog(props: ExternalRevenueDialogProps) {
  const { t } = useTranslation()
  const schema = z
    .object({
      source: z.enum(['xianyu', 'wechat', 'alipay', 'other']),
      sourceLabel: z.string().trim().max(64, t('Source name is too long')),
      externalOrderNo: z
        .string()
        .trim()
        .max(128, t('Order number is too long')),
      amount: z
        .string()
        .refine(
          (value) => decimalAmountToMinor(value) !== null,
          t('Enter a valid positive amount with up to two decimals')
        ),
      currency: z
        .string()
        .trim()
        .regex(/^[A-Za-z]{3}$/, t('Enter a three-letter currency code')),
      occurredAt: z.string().min(1, t('Received time is required')),
      note: z.string().trim().max(500, t('Note is too long')),
    })
    .superRefine((values, context) => {
      if (values.source === 'other' && !values.sourceLabel.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['sourceLabel'],
          message: t('Source name is required'),
        })
      }
    })

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      source: 'xianyu',
      sourceLabel: '',
      externalOrderNo: '',
      amount: '',
      currency: 'CNY',
      occurredAt: currentDatetimeValue(),
      note: '',
    },
  })
  const source = form.watch('source')

  useEffect(() => {
    if (!props.open) return
    if (props.record) {
      form.reset({
        source: props.record.source,
        sourceLabel: props.record.source_label,
        externalOrderNo: props.record.external_order_no ?? '',
        amount: minorAmountToDecimal(props.record.amount_minor),
        currency: props.record.currency,
        occurredAt: localDatetimeValue(props.record.occurred_at),
        note: props.record.note,
      })
      return
    }
    form.reset({
      source: 'xianyu',
      sourceLabel: '',
      externalOrderNo: '',
      amount: '',
      currency: 'CNY',
      occurredAt: currentDatetimeValue(),
      note: '',
    })
  }, [form, props.open, props.record])

  const onSubmit = async (values: FormValues) => {
    const amountMinor = decimalAmountToMinor(values.amount)
    if (!amountMinor) return
    const payload = {
      source: values.source,
      source_label: values.sourceLabel.trim(),
      external_order_no: values.externalOrderNo.trim(),
      amount_minor: amountMinor,
      currency: values.currency.trim().toUpperCase(),
      occurred_at: Math.floor(new Date(values.occurredAt).getTime() / 1000),
      note: values.note.trim(),
      version: props.record?.version,
    }
    try {
      if (props.record) {
        await updateExternalRevenue(props.record.id, payload)
        toast.success(t('External revenue updated'))
      } else {
        await createExternalRevenue(payload)
        toast.success(t('External revenue added'))
      }
      props.onSaved()
      props.onOpenChange(false)
    } catch {
      // The shared API client presents server and network errors.
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {props.record
              ? t('Edit external revenue')
              : t('Add external revenue')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'External revenue is used for reporting only and does not change platform orders or user balances.'
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            id='external-revenue-form'
            className='grid gap-4 sm:grid-cols-2'
            onSubmit={form.handleSubmit(onSubmit)}
          >
            <FormField
              control={form.control}
              name='source'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Source')}</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={(value) =>
                      value && field.onChange(value as RevenueSource)
                    }
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value='xianyu'>{t('Xianyu')}</SelectItem>
                      <SelectItem value='wechat'>{t('WeChat')}</SelectItem>
                      <SelectItem value='alipay'>{t('Alipay')}</SelectItem>
                      <SelectItem value='other'>{t('Other')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            {source === 'other' ? (
              <FormField
                control={form.control}
                name='sourceLabel'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Source name')}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : (
              <FormField
                control={form.control}
                name='externalOrderNo'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('External order number')}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            {source === 'other' ? (
              <FormField
                control={form.control}
                name='externalOrderNo'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('External order number')}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : null}
            <FormField
              control={form.control}
              name='amount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Received amount')}</FormLabel>
                  <FormControl>
                    <Input inputMode='decimal' placeholder='0.00' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='currency'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Currency')}</FormLabel>
                  <FormControl>
                    <Input maxLength={3} className='uppercase' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='occurredAt'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Received time')}</FormLabel>
                  <FormControl>
                    <Input type='datetime-local' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='note'
              render={({ field }) => (
                <FormItem className='sm:col-span-2'>
                  <FormLabel>{t('Note')}</FormLabel>
                  <FormControl>
                    <Textarea rows={3} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>
        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={form.formState.isSubmitting}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='submit'
            form='external-revenue-form'
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? (
              <Loader2 className='animate-spin' />
            ) : null}
            {props.record ? t('Update') : t('Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
