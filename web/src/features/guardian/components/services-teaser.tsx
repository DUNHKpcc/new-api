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
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { GuardianSection, SectionHeading } from './section-heading'

/** Points the landing page at the dedicated setup services page. */
export function ServicesTeaser() {
  const { t } = useTranslation()

  return (
    <GuardianSection id='services'>
      <div className='flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between'>
        <SectionHeading
          id='services'
          eyebrow={t('Hands-on services')}
          title={t('Rather have someone do it with you?')}
          lede={t(
            'The Skill is free and always will be. If you would rather not configure things yourself, book a session and we set it up together on your machine, following the same red lines and authorization tiers.'
          )}
        />
        <Button
          className='g-pill h-11 shrink-0 gap-2 px-5 text-[0.9375rem]'
          render={<Link to='/guardian/services' />}
        >
          {t('See setup services')}
          <ArrowRight aria-hidden='true' />
        </Button>
      </div>
    </GuardianSection>
  )
}
