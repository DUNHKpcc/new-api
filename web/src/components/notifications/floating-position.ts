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
export type FloatingVerticalSpan = {
  top: number
  bottom: number
}

type ResolveNotificationBottomOffsetOptions = {
  preferredOffset: number
  widgetHeight: number
  viewportHeight: number
  blockers: FloatingVerticalSpan[]
}

export const floatingNotificationPosition = {
  viewportMargin: 12,
  collisionGap: 12,
  keyboardStep: 80,
  maxCollisionPasses: 8,
  storageKey: 'global-notification-bottom-offset',
} as const

export function clampNotificationBottomOffset(
  offset: number,
  widgetHeight: number,
  viewportHeight: number
): number {
  const minimum = floatingNotificationPosition.viewportMargin
  const maximum = Math.max(
    minimum,
    viewportHeight - widgetHeight - floatingNotificationPosition.viewportMargin
  )

  return Math.min(Math.max(offset, minimum), maximum)
}

export function resolveNotificationBottomOffset(
  options: ResolveNotificationBottomOffsetOptions
): number {
  let offset = clampNotificationBottomOffset(
    options.preferredOffset,
    options.widgetHeight,
    options.viewportHeight
  )
  const blockers = options.blockers
    .filter(
      (blocker) =>
        Number.isFinite(blocker.top) &&
        Number.isFinite(blocker.bottom) &&
        blocker.bottom > blocker.top
    )
    .sort((left, right) => right.bottom - left.bottom)

  for (
    let pass = 0;
    pass < floatingNotificationPosition.maxCollisionPasses;
    pass += 1
  ) {
    const widgetBottom = options.viewportHeight - offset
    const widgetTop = widgetBottom - options.widgetHeight
    const blocker = blockers.find(
      (candidate) =>
        widgetTop <
          candidate.bottom + floatingNotificationPosition.collisionGap &&
        widgetBottom > candidate.top - floatingNotificationPosition.collisionGap
    )

    if (!blocker) break

    const nextOffset = clampNotificationBottomOffset(
      options.viewportHeight -
        blocker.top +
        floatingNotificationPosition.collisionGap,
      options.widgetHeight,
      options.viewportHeight
    )
    if (nextOffset === offset) break
    offset = nextOffset
  }

  return offset
}
