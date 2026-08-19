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
  RevenueMonthRange,
} from '../types'

const COLORS = ['#2563eb', '#0f766e', '#d97706', '#9333ea', '#dc2626']

function bigintValue(value: string | undefined): bigint {
  if (!value || !/^\d+$/.test(value)) return 0n
  return BigInt(value)
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
  metric: 'amount' | 'count' = 'amount'
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
  const normalized = value.trim()
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null
  const [whole, fraction = ''] = normalized.split('.')
  const minor = BigInt(whole) * 100n + BigInt(fraction.padEnd(2, '0'))
  if (minor <= 0n || minor > 1_000_000_000_000_000n) return null
  return minor.toString()
}

export function minorAmountToDecimal(value: string): string {
  const amount = bigintValue(value)
  return `${amount / 100n}.${String(amount % 100n).padStart(2, '0')}`
}
