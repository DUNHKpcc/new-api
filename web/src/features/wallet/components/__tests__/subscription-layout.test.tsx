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
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import type { PricingModel } from '@/features/pricing/types'

const { mock } = await import(`bun:${'test'}`)

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'IntersectionObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const now = Math.floor(Date.now() / 1000)
let publicPlanCount = 6

const estimateModel: PricingModel = {
  id: 10,
  model_name: 'estimate-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  cache_ratio: 0.5,
  enable_groups: ['default'],
  group_ratio: { default: 1 },
}

mock.module('@/features/subscriptions/api', () => ({
  getPublicPlans: async () => ({
    success: true,
    data: Array.from({ length: publicPlanCount }, (_, index) => index + 1).map(
      (id) => ({
        plan: {
          id,
          title: `Plan ${id}`,
          price_amount: id * 10,
          currency: 'USD',
          duration_unit: 'month',
          duration_value: 1,
          quota_reset_period: 'monthly',
          enabled: true,
          sort_order: id,
          allow_balance_pay: true,
          allow_wallet_overflow: true,
          max_purchase_per_user: id === 1 ? 2 : 0,
          total_amount: id * 100,
        },
      })
    ),
  }),
  getSelfSubscriptionFull: async () => ({
    success: true,
    data: {
      billing_preference: 'subscription_first',
      subscriptions: [
        {
          subscription: {
            id: 42,
            user_id: 1,
            plan_id: 4,
            status: 'active',
            start_time: now - 100,
            end_time: now + 86_400,
            amount_total: 100,
            amount_used: 10,
          },
        },
      ],
      all_subscriptions: [
        {
          subscription: {
            id: 48,
            user_id: 1,
            plan_id: 1,
            status: 'cancelled',
            start_time: now - 200,
            end_time: now - 100,
            amount_total: 5,
            amount_used: 1,
          },
        },
        {
          subscription: {
            id: 42,
            user_id: 1,
            plan_id: 4,
            status: 'active',
            start_time: now - 100,
            end_time: now + 86_400,
            amount_total: 100,
            amount_used: 10,
          },
        },
      ],
    },
  }),
  updateBillingPreference: async (preference: string) => ({
    success: true,
    data: { billing_preference: preference },
  }),
}))

mock.module(
  '@/features/subscriptions/components/dialogs/subscription-purchase-dialog',
  () => ({ SubscriptionPurchaseDialog: () => null })
)

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { SubscriptionPlansCard } = await import('../subscription-plans-card')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: false,
  resources: { en: { translation: {} } },
  returnNull: false,
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

async function renderSubscriptionPlansCard(options?: {
  models?: PricingModel[]
  quotaPerUnit?: number
  subscriptionDisplayModels?: string
}) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <SubscriptionPlansCard topupInfo={null} {...options} />
      </I18nextProvider>
    )
    await Promise.resolve()
    await Promise.resolve()
  })

  return { container, root }
}

async function cleanupRenderedCard(
  container: HTMLElement,
  root: ReturnType<typeof createRoot>
) {
  await act(async () => root.unmount())
  container.remove()
}

describe('wallet subscription layout', () => {
  afterEach(() => {
    publicPlanCount = 6
  })

  after(() => {
    mock.restore()
    domWindow.close()
  })

  test('puts active subscriptions first in an inner vertical scroll region', async () => {
    const { container, root } = await renderSubscriptionPlansCard()

    const subscriptionList = container.querySelector<HTMLElement>(
      '[data-subscription-record-list="true"]'
    )
    assert.ok(subscriptionList)
    assert.ok(subscriptionList.classList.contains('overflow-y-auto'))
    assert.ok(subscriptionList.classList.contains('overscroll-contain'))
    assert.equal(subscriptionList.getAttribute('role'), 'region')
    assert.equal(subscriptionList.hasAttribute('aria-roledescription'), false)

    const subscriptionRecords = subscriptionList.querySelectorAll(
      '[data-subscription-record-id]'
    )
    assert.equal(subscriptionRecords.length, 2)
    assert.equal(
      subscriptionRecords[0].getAttribute('data-subscription-record-id'),
      '42'
    )
    assert.equal(
      subscriptionRecords[1].getAttribute('data-subscription-record-id'),
      '48'
    )
    assert.match(subscriptionRecords[0].textContent ?? '', /Subscription #42/)
    assert.equal(
      subscriptionList.querySelector('[data-subscription-deck="true"]'),
      null
    )
    assert.equal(
      subscriptionList.querySelector('button[aria-label="Previous"]'),
      null
    )
    assert.equal(
      subscriptionList.querySelector('button[aria-label="Next"]'),
      null
    )

    await cleanupRenderedCard(container, root)
  })

  test('shows four bordered plans in a two-column vertical scroll region', async () => {
    const { container, root } = await renderSubscriptionPlansCard()

    const planGrid = container.querySelector<HTMLElement>(
      '[data-subscription-plan-grid="true"]'
    )
    assert.ok(planGrid)
    assert.equal(
      planGrid.getAttribute('data-visible-plan-count'),
      '2-mobile-4-wide'
    )
    assert.ok(planGrid.classList.contains('overflow-y-auto'))
    assert.ok(planGrid.classList.contains('xl:flex-1'))

    const planCards = planGrid.querySelectorAll('[data-subscription-plan-id]')
    assert.equal(planCards.length, 6)
    for (const card of planCards) {
      assert.ok(card.classList.contains('border'))
      assert.ok(card.classList.contains('gap-0'))
      assert.ok(card.classList.contains('py-0'))
      assert.ok(card.classList.contains('min-h-[17.5rem]'))
      const benefits = card.querySelector('[data-plan-benefits="true"]')
      assert.ok(benefits)
      assert.equal(benefits.classList.contains('overflow-hidden'), false)
    }

    const grid = planGrid.firstElementChild
    assert.ok(grid?.classList.contains('sm:grid-cols-2'))
    assert.ok(grid?.classList.contains('pb-3'))
    assert.ok(container.querySelector('[data-plan-scroll-hint="true"]'))

    const limitedPlan = planGrid.querySelector(
      '[data-subscription-plan-id="1"]'
    )
    assert.match(limitedPlan?.textContent ?? '', /¥10\.00/)
    assert.doesNotMatch(limitedPlan?.textContent ?? '', /\$10\.00/)
    assert.match(limitedPlan?.textContent ?? '', /Purchase Limit: 2/)
    assert.ok(limitedPlan?.querySelector('.break-words'))

    await cleanupRenderedCard(container, root)
  })

  test('shows the shared token estimate in wallet plan cards', async () => {
    const { container, root } = await renderSubscriptionPlansCard({
      models: [estimateModel],
      quotaPerUnit: 500_000,
      subscriptionDisplayModels: '["estimate-model"]',
    })

    const firstPlan = container.querySelector('[data-subscription-plan-id="1"]')
    assert.ok(firstPlan)
    assert.match(firstPlan.textContent ?? '', /Total quota over validity/)
    assert.match(firstPlan.textContent ?? '', /Estimated tokens/)
    assert.match(firstPlan.textContent ?? '', /estimate-model/)
    assert.equal(
      firstPlan
        .querySelector('[data-subscription-token-estimates="true"]')
        ?.getAttribute('data-subscription-token-estimates'),
      'true'
    )

    await cleanupRenderedCard(container, root)
  })

  test('shows the overflow hint for three plans on mobile only', async () => {
    publicPlanCount = 3
    const { container, root } = await renderSubscriptionPlansCard()

    const hint = container.querySelector<HTMLElement>(
      '[data-plan-scroll-hint="true"]'
    )
    assert.ok(hint)
    assert.ok(hint.classList.contains('sm:hidden'))

    await cleanupRenderedCard(container, root)
  })

  test('places subscription history below add funds and keeps plans in the right column on wide screens', async () => {
    const { container, root } = await renderSubscriptionPlansCard()

    const titles = [
      ...container.querySelectorAll<HTMLElement>('[data-slot="card-title"]'),
    ]
    const historyCard = titles
      .find((title) => title.textContent === 'My Subscriptions')
      ?.closest<HTMLElement>('[data-slot="card"]')
    const plansCard = titles
      .find((title) => title.textContent === 'Subscription Plans')
      ?.closest<HTMLElement>('[data-slot="card"]')

    assert.ok(historyCard)
    assert.ok(historyCard.classList.contains('xl:col-start-1'))
    assert.ok(historyCard.classList.contains('xl:row-start-2'))

    assert.ok(plansCard)
    assert.ok(plansCard.classList.contains('xl:col-start-2'))
    assert.equal(plansCard.classList.contains('xl:row-span-2'), true)
    assert.ok(plansCard.classList.contains('xl:row-start-1'))
    assert.ok(plansCard.classList.contains('xl:h-full'))
    assert.ok(plansCard.classList.contains('xl:self-stretch'))
    assert.ok(plansCard.classList.contains('xl:[contain:size]'))

    await cleanupRenderedCard(container, root)
  })
})
