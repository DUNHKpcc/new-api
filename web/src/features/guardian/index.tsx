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
import { FaqSection } from './components/faq-section'
import { GuardianShell } from './components/guardian-shell'
import { GuardianHero } from './components/hero'
import { HowItWorksSection } from './components/how-it-works-section'
import { PrinciplesSection } from './components/principles-section'
import { QuickStartSection } from './components/quick-start-section'
import { RoutesSection } from './components/routes-section'
import { ServicesTeaser } from './components/services-teaser'
import { TiersSection } from './components/tiers-section'

export function Guardian() {
  return (
    <GuardianShell>
      <GuardianHero />
      <PrinciplesSection />
      <HowItWorksSection />
      <RoutesSection />
      <TiersSection />
      <QuickStartSection />
      <ServicesTeaser />
      <FaqSection />
    </GuardianShell>
  )
}
