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
import type { TFunction } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'

import type { GuardianReceipt } from '../types'
import { formatCny } from './price'

// A5 landscape: screen preview scales the document; printing retains physical dimensions.
const receiptStyle = `
@page { size: A5 landscape; margin: 0; }
* { box-sizing: border-box; }
html { font-size: calc(100vw / 49.6063); background: white; color: #18202a; }
body { margin: 0; padding: 2rem 2.5rem; font: .875rem/1.5 Arial, "PingFang SC", "Microsoft YaHei", sans-serif; }
header { display: flex; justify-content: space-between; gap: 1rem; border-bottom: .15rem solid #18202a; padding-bottom: 1rem; }
h1 { margin: .25rem 0 0; font-size: 1.6rem; }
.brand { font-size: .9rem; font-weight: bold; letter-spacing: .08em; }
.number { text-align: right; max-width: 24rem; overflow-wrap: anywhere; }
dl { display: grid; grid-template-columns: 1fr 1fr; gap: .75rem 2rem; margin: 1rem 0; }
dt { font-size: .7rem; color: #637080; }
dd { margin: .15rem 0 0; overflow-wrap: anywhere; }
table { width: 100%; border-collapse: collapse; table-layout: fixed; }
td { border-bottom: 1px solid #dce1e7; padding: .6rem 0; vertical-align: top; overflow-wrap: anywhere; }
td:last-child { text-align: right; width: 9rem; white-space: nowrap; }
.total td { font-weight: bold; font-size: 1.2rem; border: 0; padding-top: .9rem; }
footer { margin-top: 1rem; border-top: 1px solid #dce1e7; padding-top: .65rem; font-size: .65rem; color: #637080; }
p { margin: .5rem 0; }
@media print { html { font-size: 16px; } body { width: 210mm; } }
`

export function buildReceiptDocument(props: {
  data: GuardianReceipt
  locale: string | undefined
  t: TFunction
  billingPeriod: string
}) {
  const { data, locale, t } = props
  return `<!doctype html>${renderToStaticMarkup(
    <html lang={locale}>
      <head>
        <meta charSet='utf-8' />
        <title>{data.number}</title>
        <style>{receiptStyle}</style>
      </head>
      <body>
        <header>
          <div>
            <div className='brand'>CLAUDE GUARDIAN</div>
            <h1>{t('Payment receipt')}</h1>
          </div>
          <div className='number'>
            {data.number}
            <p>CNY</p>
          </div>
        </header>
        <dl>
          <div>
            <dt>{t('Order number')}</dt>
            <dd>{data.trade_no}</dd>
          </div>
          <div>
            <dt>{t('Payment method')}</dt>
            <dd>
              {data.payment_method === 'alipay'
                ? t('Alipay')
                : data.payment_method}
            </dd>
          </div>
          <div>
            <dt>{t('Paid at')}</dt>
            <dd>{new Date(data.paid_at * 1000).toLocaleString(locale)}</dd>
          </div>
          <div>
            <dt>{t('Receipt created at')}</dt>
            <dd>{new Date(data.created_at * 1000).toLocaleString(locale)}</dd>
          </div>
        </dl>
        <table>
          <tbody>
            <tr>
              <td>
                {data.service_title}
                {props.billingPeriod && ` · ${props.billingPeriod}`}
              </td>
              <td>
                {formatCny(
                  data.amount_cents - data.max_cents - data.ultra_cents,
                  locale
                )}
              </td>
            </tr>
            {data.max_cents > 0 && (
              <tr>
                <td>{t('Claude MAX first month')}</td>
                <td>{formatCny(data.max_cents, locale)}</td>
              </tr>
            )}
            {data.ultra_cents > 0 && (
              <tr>
                <td>{t('Claude Ultra first month')}</td>
                <td>{formatCny(data.ultra_cents, locale)}</td>
              </tr>
            )}
            <tr className='total'>
              <td>{t('Total paid')} · CNY</td>
              <td>{formatCny(data.amount_cents, locale)}</td>
            </tr>
          </tbody>
        </table>
        {data.refund_state && (
          <p>
            {data.refund_state === 'processing'
              ? t('Refund processing')
              : t('Refunded')}{' '}
            ·{' '}
            {data.refund_cents === null
              ? t('Refund details not recorded')
              : formatCny(data.refund_cents, locale)}
            {data.refunded_at > 0 &&
              ` · ${new Date(data.refunded_at * 1000).toLocaleString(locale)}`}
          </p>
        )}
        <footer>
          {t('This receipt records your payment and is not a tax invoice.')}
        </footer>
      </body>
    </html>
  )}`
}
