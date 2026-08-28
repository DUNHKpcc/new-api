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
import type {
  DashboardFlowGraph,
  DashboardFlowLink,
  DashboardFlowNode,
  FlowNodeKind,
} from '@/features/dashboard/types'

import type {
  CombinedRevenueTotal,
  ExternalRevenueSummary,
  PlatformRevenueSummary,
  RevenueChartMetric,
  RevenueMonthRange,
  RevenueProfitSummary,
  RevenueTimelineDatum,
} from '../types'

const COLORS = ['#2563eb', '#0f766e', '#d97706', '#9333ea', '#dc2626']
export const MAX_MANUAL_COST_MINOR = 1_000_000_000_000_000n

function bigintValue(value: string | undefined): bigint {
  if (!value || !/^\d+$/.test(value)) return 0n
  try {
    return BigInt(value)
  } catch {
    return 0n
  }
}

export function revenueMonthRange(month: Date): RevenueMonthRange {
  const start = new Date(month.getFullYear(), month.getMonth(), 1)
  const end = new Date(month.getFullYear(), month.getMonth() + 1, 1)
  return {
    startTime: Math.floor(start.getTime() / 1000),
    endTime: Math.floor(end.getTime() / 1000),
    label: new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'long',
    }).format(start),
  }
}

export function revenueMonthKey(month: Date): string {
  return `${month.getFullYear()}-${String(month.getMonth() + 1).padStart(2, '0')}`
}

/** Parse a non-negative money input while keeping all arithmetic in minor units. */
export function nonNegativeDecimalAmountToMinor(value: string): string | null {
  const normalized = value.trim()
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null
  const [whole, fraction = ''] = normalized.split('.')
  let minor: bigint
  try {
    minor = BigInt(whole) * 100n + BigInt(fraction.padEnd(2, '0'))
  } catch {
    return null
  }
  if (minor < 0n || minor > MAX_MANUAL_COST_MINOR) return null
  return minor.toString()
}

export function calculateRevenueProfit(
  revenueAmountMinor: string,
  costAmountMinor: string
): RevenueProfitSummary {
  const revenue = bigintValue(revenueAmountMinor)
  const cost = bigintValue(costAmountMinor)
  const net = revenue - cost
  const marginPercent =
    revenue > 0n ? (Number(net) / Number(revenue)) * 100 : null
  return {
    revenueAmountMinor: revenue.toString(),
    costAmountMinor: cost.toString(),
    netAmountMinor: net.toString(),
    marginPercent:
      marginPercent !== null && Number.isFinite(marginPercent)
        ? marginPercent
        : null,
  }
}

export function sumMinorAmounts(values: string[]): string {
  return values
    .reduce((total, value) => total + bigintValue(value), 0n)
    .toString()
}

export function combineRevenueTotals(
  platform: PlatformRevenueSummary | undefined,
  external: ExternalRevenueSummary | undefined
): CombinedRevenueTotal[] {
  const currencies = new Set<string>()
  const platformTotals = platform?.totals ?? []
  const externalTotals = external?.totals ?? []
  platformTotals.forEach((item) => currencies.add(item.currency))
  externalTotals.forEach((item) => currencies.add(item.currency))

  return [...currencies].sort().map((currency) => {
    const platformTotal = platformTotals.find(
      (item) => item.currency === currency
    )
    const externalTotal = externalTotals.find(
      (item) => item.currency === currency
    )
    const platformAmount = bigintValue(platformTotal?.amount_minor)
    const externalAmount = bigintValue(externalTotal?.amount_minor)
    return {
      currency,
      platformAmountMinor: platformAmount.toString(),
      externalAmountMinor: externalAmount.toString(),
      totalAmountMinor: (platformAmount + externalAmount).toString(),
      platformCount: platformTotal?.count ?? 0,
      externalCount: externalTotal?.count ?? 0,
    }
  })
}

type TimelineAccumulator = {
  amount: bigint
  count: number
}

function dateKeyFromTimestamp(
  timestamp: number,
  timezoneOffset: number
): string | null {
  if (!Number.isFinite(timestamp) || !Number.isFinite(timezoneOffset)) {
    return null
  }
  const date = new Date((timestamp + timezoneOffset * 60) * 1000)
  if (!Number.isFinite(date.getTime())) return null
  return date.toISOString().slice(0, 10)
}

function addTimelinePoint(
  points: Map<string, TimelineAccumulator>,
  date: string,
  amountMinor: string,
  count: number
) {
  const current = points.get(date) ?? { amount: 0n, count: 0 }
  current.amount += bigintValue(amountMinor)
  const normalizedCount = Number(count)
  current.count +=
    Number.isFinite(normalizedCount) && normalizedCount > 0
      ? Math.trunc(normalizedCount)
      : 0
  points.set(date, current)
}

function timelineDates(
  platform: PlatformRevenueSummary | undefined,
  external: ExternalRevenueSummary | undefined,
  fallbackDates: string[]
): string[] {
  const summary = platform ?? external
  if (
    !summary ||
    !Number.isFinite(summary.start_time) ||
    !Number.isFinite(summary.end_time)
  ) {
    return [...new Set(fallbackDates)].sort()
  }
  const offset = Number.isFinite(summary.timezone_offset)
    ? summary.timezone_offset
    : 0
  const start = dateKeyFromTimestamp(summary.start_time, offset)
  const end = dateKeyFromTimestamp(
    Math.max(summary.start_time, summary.end_time - 1),
    offset
  )
  if (!start || !end) return [...new Set(fallbackDates)].sort()
  const startDay = Date.parse(`${start}T00:00:00Z`)
  const endDay = Date.parse(`${end}T00:00:00Z`)
  if (
    !Number.isFinite(startDay) ||
    !Number.isFinite(endDay) ||
    endDay < startDay
  ) {
    return [...new Set(fallbackDates)].sort()
  }
  const dayCount = Math.floor((endDay - startDay) / 86_400_000) + 1
  // The revenue screen is month-based. A hard cap keeps malformed server
  // ranges from making the chart allocate an unbounded number of rows.
  if (dayCount > 366) return [...new Set(fallbackDates)].sort()
  return Array.from({ length: dayCount }, (_, index) =>
    new Date(startDay + index * 86_400_000).toISOString().slice(0, 10)
  )
}

function timelineValue(
  amount: bigint,
  count: number,
  metric: RevenueChartMetric
): number {
  if (metric === 'count') return count
  const value = Number(amount) / 100
  return Number.isFinite(value) ? value : 0
}

export function buildRevenueTimeline(
  platform: PlatformRevenueSummary | undefined,
  external: ExternalRevenueSummary | undefined,
  currency: string,
  metric: RevenueChartMetric = 'amount'
): RevenueTimelineDatum[] {
  const platformPoints = new Map<string, TimelineAccumulator>()
  const externalPoints = new Map<string, TimelineAccumulator>()
  ;(platform?.timeline ?? [])
    .filter((item) => item.currency === currency)
    .forEach((item) =>
      addTimelinePoint(platformPoints, item.date, item.amount_minor, item.count)
    )
  ;(external?.timeline ?? [])
    .filter((item) => item.currency === currency)
    .forEach((item) =>
      addTimelinePoint(externalPoints, item.date, item.amount_minor, item.count)
    )

  const dates = timelineDates(platform, external, [
    ...platformPoints.keys(),
    ...externalPoints.keys(),
  ])
  const sources: Array<
    ['platform' | 'external', Map<string, TimelineAccumulator>]
  > = []
  if (platform) sources.push(['platform', platformPoints])
  if (external) sources.push(['external', externalPoints])

  return dates.flatMap((date) =>
    sources.map(([source, points]) => {
      const point = points.get(date) ?? { amount: 0n, count: 0 }
      return {
        date,
        source,
        amountMinor: point.amount.toString(),
        count: point.count,
        value: timelineValue(point.amount, point.count, metric),
      }
    })
  )
}

type RevenuePath = {
  ids: [string, string, string]
  labels: [string, string, string]
  value: number
  count: number
  color: string
}

function addGraphNode(
  nodes: Map<string, DashboardFlowNode>,
  id: string,
  label: string,
  kind: FlowNodeKind,
  value: number,
  count: number,
  color: string
) {
  const existing = nodes.get(id)
  if (existing) {
    existing.value += value
    existing.quota += value
    existing.requests += count
    return
  }
  nodes.set(id, {
    id,
    label,
    kind,
    value,
    quota: value,
    requests: count,
    tokens: 0,
    color,
    colorKey: id,
  })
}

export function buildRevenueFlowGraph(
  platform: PlatformRevenueSummary | undefined,
  external: ExternalRevenueSummary | undefined,
  currency: string,
  labels: { platform: string; external: string },
  metric: RevenueChartMetric = 'amount'
): DashboardFlowGraph {
  const paths: RevenuePath[] = []
  const platformMethods = platform?.by_payment_method ?? []
  const externalSources = external?.by_source ?? []
  platformMethods
    .filter((item) => item.currency === currency)
    .forEach((item, index) => {
      paths.push({
        ids: [
          'root:platform',
          `platform:${item.payment_method}`,
          `currency:${currency}`,
        ],
        labels: [labels.platform, item.payment_method, currency],
        value:
          metric === 'count'
            ? item.count
            : Number(bigintValue(item.amount_minor)),
        count: item.count,
        color: COLORS[index % COLORS.length],
      })
    })
  externalSources
    .filter((item) => item.currency === currency)
    .forEach((item, index) => {
      const sourceLabel = item.source_label || item.source
      paths.push({
        ids: [
          'root:external',
          `external:${item.source}:${sourceLabel}`,
          `currency:${currency}`,
        ],
        labels: [labels.external, sourceLabel, currency],
        value:
          metric === 'count'
            ? item.count
            : Number(bigintValue(item.amount_minor)),
        count: item.count,
        color: COLORS[(index + 2) % COLORS.length],
      })
    })

  const nodes = new Map<string, DashboardFlowNode>()
  const links = new Map<string, DashboardFlowLink>()
  const total = paths.reduce((sum, path) => sum + path.value, 0)
  paths.forEach((path) => {
    path.ids.forEach((id, index) => {
      const kinds: FlowNodeKind[] = ['user', 'model', 'channel']
      addGraphNode(
        nodes,
        id,
        path.labels[index],
        kinds[index],
        path.value,
        path.count,
        path.color
      )
    })
    for (let index = 0; index < path.ids.length - 1; index += 1) {
      const source = path.ids[index]
      const target = path.ids[index + 1]
      const key = `${source}\u0000${target}`
      const existing = links.get(key)
      if (existing) {
        existing.value += path.value
        existing.quota += path.value
        existing.requests += path.count
        continue
      }
      links.set(key, {
        source,
        target,
        value: path.value,
        quota: path.value,
        requests: path.count,
        tokens: 0,
        sourceLabel: path.labels[index],
        targetLabel: path.labels[index + 1],
        color: path.color,
        linkColor: path.color,
        hoverColor: path.color,
        linkAlpha: 0.45,
        colorKey: key,
        share: total > 0 ? path.value / total : 0,
      })
    }
  })
  return { nodes: [...nodes.values()], links: [...links.values()] }
}

export function decimalAmountToMinor(value: string): string | null {
  const minor = nonNegativeDecimalAmountToMinor(value)
  if (minor === null || minor === '0') return null
  return minor
}

export function minorAmountToDecimal(value: string): string {
  const amount = bigintValue(value)
  return `${amount / 100n}.${String(amount % 100n).padStart(2, '0')}`
}
