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

import { WalletPromotionNotice } from '../wallet-promotion-notice'

describe('wallet promotion notice', () => {
  test('shows configured promotion details with multiline wrapping', () => {
    const markup = renderToStaticMarkup(
      <WalletPromotionNotice
        content={'  Save 20% today.\nEnds at midnight.  '}
      />
    )

    assert.match(markup, /role="alert"/)
    assert.match(markup, /whitespace-pre-wrap/)
    assert.match(markup, /break-words/)
    assert.match(markup, /max-h-\[min\(30svh,16rem\)\]/)
    assert.match(markup, /overflow-y-auto/)
    assert.match(markup, /overscroll-contain/)
    assert.match(markup, /scrollbar-gutter:stable/)
    assert.match(markup, /tabindex="0"/)
    assert.match(markup, /Save 20% today\.\nEnds at midnight\./)
    assert.doesNotMatch(markup, />\s{2}Save 20%/)
  })

  test('stays hidden when the administrator leaves the notice blank', () => {
    const markup = renderToStaticMarkup(
      <WalletPromotionNotice content={'   \n  '} />
    )

    assert.equal(markup, '')
  })
})
