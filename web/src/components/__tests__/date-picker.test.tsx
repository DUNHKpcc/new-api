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

import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { DatePicker } from '../date-picker'

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

describe('date picker', () => {
  test('renders a custom-formatted month selection with an accessible trigger', () => {
    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DatePicker
          selected={new Date(2026, 7, 1)}
          onSelect={() => undefined}
          granularity='month'
          formatSelected={() => 'August 2026'}
          ariaLabel='Select month'
        />
      </I18nextProvider>
    )

    assert.match(markup, /data-granularity="month"/)
    assert.match(markup, /aria-label="Select month"/)
    assert.match(markup, />August 2026</)
  })
})
