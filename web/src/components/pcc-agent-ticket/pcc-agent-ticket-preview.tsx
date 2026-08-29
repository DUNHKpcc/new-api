/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PccAgentLogo } from '@/features/desktop-authorization/pcc-agent-logo'

type PccAgentTicketPreviewProps = {
  title?: string
  content?: string
}

export function PccAgentTicketPreview(props: PccAgentTicketPreviewProps) {
  const { t } = useTranslation()
  const title =
    props.title?.trim() ||
    t('Claim {{amount}} in AI API credits', { amount: '$450' })
  const content =
    props.content?.trim() || t('Use PccAgent to unlock your credit')

  return (
    <article
      className='pcc-agent-ticket pcc-agent-ticket--embedded'
      aria-label={t('PccAgent $450 credit offer')}
    >
      <div className='pcc-agent-ticket__body'>
        <div className='pcc-agent-ticket__topline'>
          <span className='pcc-agent-ticket__eyebrow'>
            {t('Limited offer')}
          </span>
          <span className='pcc-agent-ticket__serial'>PCC / 450</span>
        </div>

        <div className='pcc-agent-ticket__brand'>
          <span className='pcc-agent-ticket__brand-mark'>
            <PccAgentLogo className='pcc-agent-ticket__logo' />
          </span>
          <span className='pcc-agent-ticket__brand-name'>PccAgent</span>
        </div>

        <p className='pcc-agent-ticket__headline'>{title}</p>

        <div className='pcc-agent-ticket__amount' aria-hidden='true'>
          <span className='pcc-agent-ticket__currency'>$</span>
          <strong>450</strong>
          <span className='pcc-agent-ticket__unit'>{t('AI API credits')}</span>
        </div>

        <p className='pcc-agent-ticket__subline'>{content}</p>
      </div>

      <div className='pcc-agent-ticket__stub-shell' aria-hidden='true'>
        <div className='pcc-agent-ticket__stub'>
          <span className='pcc-agent-ticket__stub-topline'>
            <span>PccAgent</span>
            <span>01 / 01</span>
          </span>
          <span className='pcc-agent-ticket__stub-title'>$450</span>
          <span className='pcc-agent-ticket__stub-guide'>
            <ArrowRight aria-hidden='true' />
            <span>{t('Official PccAgent offer')}</span>
          </span>
        </div>
      </div>

      <span className='pcc-agent-ticket__seam' aria-hidden='true' />
      <span
        className='pcc-agent-ticket__notch pcc-agent-ticket__notch--top'
        aria-hidden='true'
      />
      <span
        className='pcc-agent-ticket__notch pcc-agent-ticket__notch--bottom'
        aria-hidden='true'
      />
    </article>
  )
}
