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
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'

import { useAuthStore } from '@/stores/auth-store'

export type ConsoleOnboardingStep =
  | 'affiliate'
  | 'resource-downloads'
  | 'lottery'

const CONSOLE_ONBOARDING_STEPS: ConsoleOnboardingStep[] = [
  'affiliate',
  'resource-downloads',
  'lottery',
]
const CONSOLE_ONBOARDING_STORAGE_PREFIX = 'console_onboarding_seen'

// eslint-disable-next-line react-refresh/only-export-components
export function getNextConsoleOnboardingStep(
  registeredSteps: readonly ConsoleOnboardingStep[],
  seenSteps: readonly ConsoleOnboardingStep[]
): ConsoleOnboardingStep | null {
  return (
    CONSOLE_ONBOARDING_STEPS.find(
      (step) => registeredSteps.includes(step) && !seenSteps.includes(step)
    ) ?? null
  )
}

type ConsoleOnboardingContextValue = {
  activeStep: ConsoleOnboardingStep | null
  completeStep: (step: ConsoleOnboardingStep) => void
  registerStep: (step: ConsoleOnboardingStep) => void
  unregisterStep: (step: ConsoleOnboardingStep) => void
}

const ConsoleOnboardingContext =
  createContext<ConsoleOnboardingContextValue | null>(null)

const noopRegister = (_step: ConsoleOnboardingStep) => {}

function getStorageKey(userId: number): string {
  return `${CONSOLE_ONBOARDING_STORAGE_PREFIX}:${userId}`
}

function readSeenSteps(userId: number): ConsoleOnboardingStep[] {
  try {
    const raw = window.localStorage.getItem(getStorageKey(userId))
    const parsed: unknown = raw ? JSON.parse(raw) : []
    const seen = Array.isArray(parsed) ? parsed : []
    const steps = CONSOLE_ONBOARDING_STEPS.filter((step) => seen.includes(step))

    if (
      window.localStorage.getItem(`affiliate_onboarding_seen:${userId}`) ===
        'true' &&
      !steps.includes('affiliate')
    ) {
      steps.unshift('affiliate')
    }

    return steps
  } catch {
    return []
  }
}

function persistSeenSteps(storageKey: string, steps: ConsoleOnboardingStep[]) {
  try {
    window.localStorage.setItem(storageKey, JSON.stringify(steps))
  } catch {
    return
  }
}

export function ConsoleOnboardingProvider(props: { children: ReactNode }) {
  const userId = useAuthStore((state) => state.auth.user?.id)
  const storageKey = userId === undefined ? null : getStorageKey(userId)
  const [seenState, setSeenState] = useState<{
    key: string | null
    steps: ConsoleOnboardingStep[]
  }>({ key: null, steps: [] })
  const [registeredSteps, setRegisteredSteps] = useState<
    ConsoleOnboardingStep[]
  >([])

  useEffect(() => {
    if (userId === undefined || !storageKey) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSeenState({ key: null, steps: [] })
      return
    }

    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSeenState({ key: storageKey, steps: readSeenSteps(userId) })
  }, [storageKey, userId])

  const registerStep = useCallback((step: ConsoleOnboardingStep) => {
    setRegisteredSteps((current) =>
      current.includes(step) ? current : [...current, step]
    )
  }, [])

  const unregisterStep = useCallback((step: ConsoleOnboardingStep) => {
    setRegisteredSteps((current) => current.filter((item) => item !== step))
  }, [])

  const completeStep = useCallback(
    (step: ConsoleOnboardingStep) => {
      if (!storageKey) return

      setSeenState((current) => {
        const previous = current.key === storageKey ? current.steps : []
        if (previous.includes(step)) return current

        const steps = [...previous, step]
        persistSeenSteps(storageKey, steps)
        return { key: storageKey, steps }
      })
    },
    [storageKey]
  )

  const activeStep = useMemo(() => {
    if (!storageKey || seenState.key !== storageKey) return null

    return getNextConsoleOnboardingStep(registeredSteps, seenState.steps)
  }, [registeredSteps, seenState, storageKey])

  const value = useMemo<ConsoleOnboardingContextValue>(
    () => ({
      activeStep,
      completeStep,
      registerStep,
      unregisterStep,
    }),
    [activeStep, completeStep, registerStep, unregisterStep]
  )

  return (
    <ConsoleOnboardingContext.Provider value={value}>
      {props.children}
    </ConsoleOnboardingContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useConsoleOnboarding(step: ConsoleOnboardingStep) {
  const context = useContext(ConsoleOnboardingContext)
  const registerStep = context?.registerStep ?? noopRegister
  const unregisterStep = context?.unregisterStep ?? noopRegister
  const completeStep = context?.completeStep
  const hasContext = context !== null

  useEffect(() => {
    if (!hasContext) return
    registerStep(step)
    return () => unregisterStep(step)
  }, [hasContext, registerStep, step, unregisterStep])

  const complete = useCallback(() => {
    completeStep?.(step)
  }, [completeStep, step])

  return {
    isActive: context?.activeStep === step,
    complete,
  }
}
