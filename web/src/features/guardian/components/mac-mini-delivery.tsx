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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { FaqList } from './faq-section'
import { ServiceProcessSection } from './service-process-section'

export function MacMiniDelivery() {
  const { t } = useTranslation()
  return (
    <>
      <ServiceProcessSection
        title={t('Your Mac, from order to handover.')}
        steps={[
          {
            title: t('Choose, then pay'),
            body: t(
              'Check the hardware, region, rental period and included services in the plan. Pay through this site and follow progress in My orders.'
            ),
          },
          {
            title: t('Prepare your device'),
            body: t(
              'We contact you to arrange the device and connection. Once equipment and payment are ready, setup can take as little as 30 minutes; availability and the queue determine the actual delivery time.'
            ),
          },
          {
            title: t('Connect and inspect'),
            body: t(
              'Install UU Remote from its official website. Use the privately supplied device ID and verification code. Check the Mac configuration, network, keyboard, display and file transfer before accepting delivery.'
            ),
          },
          {
            title: t('Confirm the rental period'),
            body: t(
              'After you confirm the device works, support records the rental start and expiry. Payment alone does not start the rental. Confirm dates with support; Claude subscriptions follow their own activation dates.'
            ),
          },
        ]}
      />
      <nav
        aria-label={t('Delivery and rental guide')}
        className='mx-auto flex max-w-6xl flex-wrap justify-center gap-3 px-4 py-8'
      >
        <Button
          variant='outline'
          render={
            <a
              href='https://uuyc.163.com/'
              target='_blank'
              rel='noopener noreferrer'
            />
          }
        >
          UU Remote ↗
        </Button>
        <Button variant='outline' render={<Link to='/guardian/orders' />}>
          {t('My orders')}
        </Button>
        <Button variant='outline' render={<Link to='/support' />}>
          {t('Customer Service')}
        </Button>
      </nav>
      <FaqList
        id='mac-mini-delivery'
        title={t('Delivery and rental guide')}
        items={[
          {
            q: t('Hardware and network'),
            a: t(
              'An overseas physical Mac mini is reserved for your use, with native macOS and a residential IP in Korea or Australia. Confirm the model, memory, storage and network with your selected plan. An IP is not a guarantee against account restrictions.'
            ),
          },
          {
            q: t('Claude accounts and subscriptions'),
            a: t(
              'Device rental does not include Claude unless the plan says so. Use your own account, or arrange registration and payment assistance. Keep control of your email, passwords and recovery details. Verify the activated plan and renewal cost yourself.'
            ),
          },
          {
            q: t('Everyday use and renewal'),
            a: t(
              'Disconnect UU when finished; this does not pause the rental. Contact support before shutting down or changing remote access settings. Arrange renewal with support before expiry, checking device and subscription periods separately.'
            ),
          },
          {
            q: t('Connection failures and pauses'),
            a: t(
              'Send support your order number, incident time and a redacted error screenshot. A confirmed device interruption can be recorded and the rental extended for the approved pause. Subscription time and remote access permissions are handled separately.'
            ),
          },
          {
            q: t('Refund assessment'),
            a: t(
              'Support reviews the cause and the affected service. For an eligible refund, calculate each item separately: amount actually paid \u00d7 unused time \u00f7 paid period, less prior refunds. Account restrictions alone do not establish that the device service failed; confirmed commitments and statutory rights still apply.'
            ),
          },
          {
            q: t('Backups and returning the Mac'),
            a: t(
              'Keep your own backups. Before returning the device, download and check your files and sign out of personal accounts. Ask support to finish the handover and revoke access; do not erase or reinstall the Mac yourself.'
            ),
          },
          {
            q: t('Usage and account risks'),
            a: t(
              'Follow applicable rules. Mining, distributed training, resale of compute, credential sharing and abusive access are prohibited. Remote hardware does not guarantee account approval or uninterrupted Claude access. Support must assess evidence when handling restrictions or disputes.'
            ),
          },
        ]}
      />
    </>
  )
}
