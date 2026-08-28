/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

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
import { handleServerError } from '@/lib/handle-server-error'

import {
  minorAmountToDecimal,
  nonNegativeDecimalAmountToMinor,
} from '../lib/analytics'
import type {
  RevenueCostCategory,
  RevenueCostMutation,
  RevenueCostRecord,
} from '../types'

type RevenueCostDialogProps = {
  open: boolean
  month: string
  currency: string
  record: RevenueCostRecord | null
  onOpenChange: (open: boolean) => void
  onSaved: (payload: RevenueCostMutation) => Promise<void>
}

type FormValues = {
  category: RevenueCostCategory
  amount: string
  description: string
}

export function RevenueCostDialog(props: RevenueCostDialogProps) {
  const { t } = useTranslation()
  const schema = z.object({
    category: z.enum(['server', 'upstream', 'account', 'other']),
    amount: z
      .string()
      .refine(
        (value) => nonNegativeDecimalAmountToMinor(value) !== null,
        t('Enter a valid non-negative amount with up to two decimals')
      ),
    description: z.string().trim().max(500, t('Details are too long')),
  })
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      category: 'server',
      amount: '',
      description: '',
    },
  })

  useEffect(() => {
    if (!props.open) return
    form.reset(
      props.record
        ? {
            category: props.record.category,
            amount: minorAmountToDecimal(props.record.amount_minor),
            description: props.record.description,
          }
        : {
            category: 'server',
            amount: '',
            description: '',
          }
    )
  }, [form, props.open, props.record])

  const onSubmit = async (values: FormValues) => {
    const amountMinor = nonNegativeDecimalAmountToMinor(values.amount)
    if (amountMinor === null) return
    const payload: RevenueCostMutation = {
      month: props.month,
      category: values.category,
      description: values.description.trim(),
      amount_minor: amountMinor,
      currency: props.currency,
      version: props.record?.version,
    }
    try {
      await props.onSaved(payload)
      toast.success(t(props.record ? 'Cost updated' : 'Cost added'))
      props.onOpenChange(false)
    } catch (error) {
      handleServerError(error)
      return
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>
            {props.record ? t('Edit cost') : t('Add cost')}
          </DialogTitle>
          <DialogDescription>
            {t('Record an operating cost for the selected month and currency.')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            id='revenue-cost-form'
            className='grid gap-4 sm:grid-cols-2'
            onSubmit={form.handleSubmit(onSubmit)}
          >
            <div className='space-y-1.5'>
              <p className='text-sm font-medium'>{t('Month')}</p>
              <Input value={props.month} readOnly aria-label={t('Month')} />
            </div>
            <div className='space-y-1.5'>
              <p className='text-sm font-medium'>{t('Currency')}</p>
              <Input
                value={props.currency}
                readOnly
                aria-label={t('Currency')}
              />
            </div>
            <FormField
              control={form.control}
              name='category'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Cost category')}</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={(value) =>
                      value && field.onChange(value as RevenueCostCategory)
                    }
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value='server'>{t('Server')}</SelectItem>
                      <SelectItem value='upstream'>
                        {t('Upstream resources')}
                      </SelectItem>
                      <SelectItem value='account'>{t('Account')}</SelectItem>
                      <SelectItem value='other'>{t('Other')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='amount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Cost amount')}</FormLabel>
                  <FormControl>
                    <Input inputMode='decimal' placeholder='0.00' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='description'
              render={({ field }) => (
                <FormItem className='sm:col-span-2'>
                  <FormLabel>{t('Details')}</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      placeholder={t('Describe what this cost covers')}
                      {...field}
                    />
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
            form='revenue-cost-form'
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
