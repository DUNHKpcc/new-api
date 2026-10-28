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
/** Radius in CSS pixels of the bright patch that trails the cursor. */
export const SPOTLIGHT_RADIUS = 240
/** Ripple expansion speed in CSS pixels per millisecond. */
const RIPPLE_SPEED = 1.1
/** Distance after which a ripple has fully faded. */
const RIPPLE_REACH = 1400
/** Thickness of the ripple ring. */
const RIPPLE_WIDTH = 70

export type PixelRipple = { x: number; y: number; start: number }

function hash3(x: number, y: number, z: number): number {
  let h =
    Math.imul(x, 73856093) ^ Math.imul(y, 19349663) ^ Math.imul(z, 83492791)
  h = Math.imul(h ^ (h >>> 13), 1274126177)
  return ((h ^ (h >>> 16)) >>> 0) / 4294967295
}

function smooth(t: number): number {
  return t * t * (3 - 2 * t)
}

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t
}

/** Smooth, deterministic 3D value noise in the range [-1, 1]. */
export function valueNoise3D(x: number, y: number, z: number): number {
  const xi = Math.floor(x)
  const yi = Math.floor(y)
  const zi = Math.floor(z)
  const u = smooth(x - xi)
  const v = smooth(y - yi)
  const w = smooth(z - zi)
  const c = (dx: number, dy: number, dz: number) =>
    hash3(xi + dx, yi + dy, zi + dz)
  const near = lerp(
    lerp(c(0, 0, 0), c(1, 0, 0), u),
    lerp(c(0, 1, 0), c(1, 1, 0), u),
    v
  )
  const far = lerp(
    lerp(c(0, 0, 1), c(1, 0, 1), u),
    lerp(c(0, 1, 1), c(1, 1, 1), u),
    v
  )
  return lerp(near, far, w) * 2 - 1
}

/** Falloff of the cursor patch: 1 under the cursor, 0 at the radius. */
export function spotlight(distance: number, radius = SPOTLIGHT_RADIUS): number {
  if (distance >= radius) return 0
  const t = 1 - distance / radius
  return t * t
}

/** Brightness a ripple adds at `distance` from its origin, `age` ms after the click. */
export function rippleRing(distance: number, age: number): number {
  if (age < 0) return 0
  const radius = age * RIPPLE_SPEED
  const fade = 1 - radius / RIPPLE_REACH
  if (fade <= 0) return 0
  return Math.exp(-(((distance - radius) / RIPPLE_WIDTH) ** 2)) * fade
}

export function isRippleAlive(ripple: PixelRipple, now: number): boolean {
  return (now - ripple.start) * RIPPLE_SPEED < RIPPLE_REACH
}

/**
 * Opacity of one matrix cell. Noise sets a quiet breathing base, the cursor
 * patch lifts both the base and a flat glow, and ripples add a moving ring.
 */
export function cellAlpha(noise: number, spot: number, ripple: number): number {
  const base = Math.max(0, noise - 0.1) * 0.8
  return Math.min(1, base * (1 + spot * 1.5) + spot * 0.5 + ripple * 0.85)
}

export function parseHexColor(value: string): string | null {
  const hex = value.trim()
  return /^#[0-9a-f]{6}$/i.test(hex) ? hex : null
}
