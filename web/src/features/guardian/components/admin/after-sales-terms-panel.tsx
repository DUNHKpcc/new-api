import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel } from '@/components/ui/field'
import { Textarea } from '@/components/ui/textarea'
import { TitledCard } from '@/components/ui/titled-card'
import { handleServerError } from '@/lib/handle-server-error'

import {
  getAdminGuardianAfterSalesTerms,
  saveGuardianAfterSalesTerms,
} from '../../api'

const schema = z.object({
  deployment: z.string().max(20000),
  mac: z.string().max(20000),
})

export function GuardianAfterSalesTermsPanel() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const query = useQuery({
    queryKey: ['guardian', 'admin-after-sales-terms'],
    queryFn: getAdminGuardianAfterSalesTerms,
  })
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { deployment: '', mac: '' },
  })
  const dirty = form.formState.isDirty
  useEffect(() => {
    if (query.data && !dirty) form.reset(query.data)
  }, [query.data, form, dirty])
  const save = useMutation({
    mutationFn: saveGuardianAfterSalesTerms,
    onSuccess: (terms) => {
      client.setQueryData(['guardian', 'admin-after-sales-terms'], terms)
      form.reset(terms)
      void client.invalidateQueries({ queryKey: ['guardian'] })
      toast.success(t('Saved successfully'))
    },
    onError: (error) => handleServerError(error),
  })
  return (
    <TitledCard
      title={t('After-sales terms')}
      description={t(
        'Configure terms separately for environment deployment and Mac services. Empty terms disable new orders in that category.'
      )}
      disableHoverEffect
    >
      {query.isPending && <LoadingState />}
      {query.isError && <ErrorState onRetry={() => void query.refetch()} />}
      {query.isSuccess && (
        <form
          className='space-y-4'
          onSubmit={form.handleSubmit((values) => save.mutate(values))}
        >
          <Field>
            <FieldLabel htmlFor='guardian-deployment-terms'>
              {t('Environment deployment')}
            </FieldLabel>
            <Textarea
              id='guardian-deployment-terms'
              rows={10}
              maxLength={20000}
              {...form.register('deployment')}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='guardian-mac-terms'>
              {t('Mac services')}
            </FieldLabel>
            <Textarea
              id='guardian-mac-terms'
              rows={10}
              maxLength={20000}
              {...form.register('mac')}
            />
          </Field>
          <Button
            type='submit'
            disabled={save.isPending || !form.formState.isDirty}
          >
            {t('Save')}
          </Button>
        </form>
      )}
    </TitledCard>
  )
}
