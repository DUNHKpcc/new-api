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

import { renderToStaticMarkup } from 'react-dom/server'

import { HeroModelSnowfall } from '../hero-model-snowfall'

describe('home hero model snowfall', () => {
  test('renders a decorative loop populated by the requested model families', () => {
    const markup = renderToStaticMarkup(<HeroModelSnowfall />)

    assert.match(markup, /data-hero-model-snowfall="true"/)
    assert.match(markup, /aria-hidden="true"/)
    assert.equal((markup.match(/hero-model-flake/g) || []).length, 48)

    for (const model of [
      'OpenAI',
      'Claude',
      'DeepSeek',
      'Kimi',
      'Grok',
      'Gemini',
    ]) {
      assert.match(markup, new RegExp(`data-model-icon="${model}"`))
    }

    assert.match(markup, /aria-label="Kimi"[^>]*style="[^"]*background:#000/)
  })

  test('seeds every model logo with deterministic motion variables', () => {
    const markup = renderToStaticMarkup(<HeroModelSnowfall />)

    assert.equal((markup.match(/--hero-model-duration:/g) || []).length, 48)
    assert.equal((markup.match(/--hero-model-delay:-/g) || []).length, 48)
    assert.equal((markup.match(/--hero-model-left:/g) || []).length, 48)
    assert.equal((markup.match(/--hero-model-size:48px/g) || []).length, 48)
  })

  test('keeps every responsive snowfall lane on a unique vertical phase', () => {
    const markup = renderToStaticMarkup(<HeroModelSnowfall />)
    const flakeStyles = [
      ...markup.matchAll(/hero-model-flake[^>]+style="([^"]+)"/g),
    ].map((match) => match[1])

    assert.equal(flakeStyles.length, 48)
    assert.equal((markup.match(/--hero-model-duration:52s/g) || []).length, 48)
    assert.equal((markup.match(/--hero-model-drift:0px/g) || []).length, 48)
    assert.equal((markup.match(/--hero-model-rotate:0deg/g) || []).length, 48)

    for (const [suffix, laneCount, visibleCount] of [
      ['', 14, 48],
      ['-tablet', 9, 48],
      ['-mobile', 5, 36],
    ] as const) {
      const lanes = new Map<string, Set<string>>()

      for (const style of flakeStyles.slice(0, visibleCount)) {
        const left = style.match(
          new RegExp(`--hero-model-left${suffix}:([^;]+)`)
        )?.[1]
        const delay = style.match(
          new RegExp(`--hero-model-delay${suffix}:([^;]+)`)
        )?.[1]

        assert.ok(left)
        assert.ok(delay)

        const phases = lanes.get(left) ?? new Set<string>()
        assert.equal(phases.has(delay), false)
        phases.add(delay)
        lanes.set(left, phases)
      }

      assert.equal(lanes.size, laneCount)
    }
  })
})
