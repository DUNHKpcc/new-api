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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getNextConsoleOnboardingStep,
  type ConsoleOnboardingStep,
} from '../console-onboarding-context'

const allSteps: ConsoleOnboardingStep[] = [
  'affiliate',
  'resource-downloads',
  'lottery',
]

describe('console onboarding sequence', () => {
  test('starts with the first registered step', () => {
    assert.equal(getNextConsoleOnboardingStep(allSteps, []), 'affiliate')
  })

  test('advances in order after a step is completed', () => {
    assert.equal(
      getNextConsoleOnboardingStep(allSteps, ['affiliate']),
      'resource-downloads'
    )
    assert.equal(
      getNextConsoleOnboardingStep(allSteps, [
        'affiliate',
        'resource-downloads',
      ]),
      'lottery'
    )
  })

  test('does not activate an unregistered step', () => {
    assert.equal(
      getNextConsoleOnboardingStep(['resource-downloads', 'lottery'], []),
      'resource-downloads'
    )
    assert.equal(
      getNextConsoleOnboardingStep(
        ['resource-downloads'],
        ['resource-downloads']
      ),
      null
    )
  })
})
