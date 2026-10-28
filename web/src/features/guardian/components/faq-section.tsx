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
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

import { GuardianSection, SectionHeading } from './section-heading'

type FaqItem = { q: string; a: string }

function FaqList(props: { id: string; title: string; items: FaqItem[] }) {
  return (
    <GuardianSection id={props.id} tone='sunk'>
      <div className='grid gap-12 lg:grid-cols-[0.8fr_1.2fr]'>
        <SectionHeading id={props.id} eyebrow='FAQ' title={props.title} />
        <Accordion className='g-rule border-t'>
          {props.items.map((item) => (
            <AccordionItem
              key={item.q}
              value={item.q}
              className='g-rule border-b'
            >
              <AccordionTrigger className='g-heading py-5 text-left text-lg hover:no-underline'>
                {item.q}
              </AccordionTrigger>
              <AccordionContent className='pb-5 text-[0.9375rem] leading-7 text-[var(--g-ink-soft)]'>
                {item.a}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </GuardianSection>
  )
}

export function FaqSection() {
  const { t } = useTranslation()
  const items = [
    {
      q: t('Is the Skill really free?'),
      a: t(
        'Yes. Claude-Guardian is open source under GPL-3.0-only. The paid services are optional hands-on help; everything they do is documented in the Skill.'
      ),
    },
    {
      q: t('Can you guarantee my account will not be banned?'),
      a: t(
        'No one can. The Skill and the services lower avoidable risk and help you collect evidence, but access rules change and the final decision is never ours.'
      ),
    },
    {
      q: t('Is this affiliated with Anthropic?'),
      a: t(
        'No. Claude-Guardian is an independent community project and is not affiliated with, endorsed by or sponsored by Anthropic.'
      ),
    },
  ]
  return (
    <FaqList id='faq' title={t('Questions people ask first')} items={items} />
  )
}

export function ServicesFaqSection() {
  const { t } = useTranslation()
  const items = [
    {
      q: t('Do you need my passwords or proxy subscription?'),
      a: t(
        'No. The order form only asks how to reach you. Remote access is arranged privately, scoped to the session, and you can revoke it at any time.'
      ),
    },
    {
      q: t('What happens after I pay?'),
      a: t(
        'The order shows as paid under My orders. We contact you to schedule, then move it through scheduled, in progress and delivered. Notes from support appear on the order.'
      ),
    },
    {
      q: t('How do refunds work?'),
      a: t(
        'Refunds are handled by support. Contact us with your order number and the order will be marked as refunded once processed.'
      ),
    },
  ]
  return (
    <FaqList id='services-faq' title={t('Before you book')} items={items} />
  )
}
