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

import { ProfileHeader } from '../profile-header'

describe('profile header', () => {
  test('does not draw the circular avatar overlay inside the square profile avatar', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      resources: { en: { translation: {} } },
    })

    const markup = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <ProfileHeader
          loading={false}
          profile={{
            id: 1,
            username: 'root',
            display_name: 'Root User',
            role: 100,
            group: 'default',
            quota: 200,
            used_quota: 0,
            request_count: 0,
            status: 1,
            aff_count: 0,
            aff_quota: 0,
            aff_history_quota: 0,
            created_time: 0,
          }}
        />
      </I18nextProvider>
    )

    assert.match(markup, /data-slot="avatar"[^>]*class="[^"]*after:hidden/)
  })
})
