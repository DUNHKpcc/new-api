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
import { ExternalLink, Headphones } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SiBilibili, SiTiktok } from 'react-icons/si'

import douyinOfficialQr from '@/assets/contact/douyin-official.jpg'
import qqIcon from '@/assets/contact/qq-icon.png'
import qqSupportQr from '@/assets/contact/qq-support.png'
import telegramDevelopersQr from '@/assets/contact/telegram-developers.png'
import telegramIcon from '@/assets/contact/tg-icon.png'
import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'

const BILIBILI_ACCOUNT_URL =
  'https://space.bilibili.com/397853169?spm_id_from=333.1007.0.0'
const DOUYIN_ACCOUNT_ID = '933648547'

export function Support() {
  const { t } = useTranslation()
  const channels = [
    {
      id: 'qq',
      platform: 'QQ',
      title: t('QQ Support Group'),
      description: t(
        'Scan with QQ to join the official group for product support and after-sales service.'
      ),
      qrAlt: t('QQ support group QR code'),
      icon: qqIcon,
      qrCode: qqSupportQr,
    },
    {
      id: 'telegram',
      platform: 'Telegram',
      title: t('Telegram Group'),
      description: t(
        'Scan with Telegram to join the group for technical support and community discussion.'
      ),
      qrAlt: t('Telegram group QR code'),
      icon: telegramIcon,
      qrCode: telegramDevelopersQr,
    },
  ]

  return (
    <PublicLayout>
      <div className='mx-auto w-full max-w-5xl py-8 sm:py-12'>
        <header className='max-w-2xl'>
          <div className='flex items-start gap-3'>
            <Headphones
              className='text-primary mt-1 size-7 shrink-0'
              aria-hidden='true'
            />
            <div>
              <p className='text-muted-foreground text-sm font-medium'>
                {t('Customer Service')}
              </p>
              <h1 className='mt-1 text-2xl font-bold sm:text-3xl'>
                {t('Customer Support')}
              </h1>
            </div>
          </div>
          <p className='text-muted-foreground mt-4 text-sm leading-6 sm:text-base'>
            {t(
              'Get product help and technical support through our official communities.'
            )}
          </p>
        </header>

        <section
          className='mt-8 grid grid-cols-1 gap-4 md:grid-cols-2 md:gap-5'
          aria-label={t('Official support channels')}
        >
          {channels.map((channel) => (
            <figure
              key={channel.id}
              data-support-channel={channel.id}
              className='border-border bg-card flex min-w-0 flex-col rounded-md border p-5 shadow-xs sm:p-6'
            >
              <div className='flex items-center gap-2.5'>
                <img
                  src={channel.icon}
                  alt=''
                  className='size-7 shrink-0 object-contain'
                  aria-hidden='true'
                />
                <div className='min-w-0'>
                  <p className='text-muted-foreground text-xs font-medium'>
                    {channel.platform}
                  </p>
                  <h2 className='text-base font-semibold sm:text-lg'>
                    {channel.title}
                  </h2>
                </div>
              </div>

              <div className='mx-auto mt-5 aspect-square w-full max-w-[280px] overflow-hidden rounded-sm bg-white p-2 ring-1 ring-black/10'>
                <img
                  src={channel.qrCode}
                  alt={channel.qrAlt}
                  className='size-full object-contain'
                  draggable={false}
                />
              </div>

              <figcaption className='text-muted-foreground mx-auto mt-5 max-w-sm text-center text-sm leading-6'>
                {channel.description}
              </figcaption>
            </figure>
          ))}
        </section>

        <section className='mt-12' aria-labelledby='official-accounts-title'>
          <div className='max-w-2xl'>
            <h2
              id='official-accounts-title'
              className='text-xl font-bold sm:text-2xl'
            >
              {t('DPCC API Official Accounts')}
            </h2>
            <p className='text-muted-foreground mt-2 text-sm leading-6'>
              {t(
                'Follow DPCC API on official video platforms for updates and tutorials.'
              )}
            </p>
          </div>

          <div className='mt-5 grid grid-cols-1 gap-4 md:grid-cols-2 md:gap-5'>
            <article
              data-official-account='bilibili'
              className='border-border bg-card flex min-w-0 flex-col rounded-md border p-5 shadow-xs sm:p-6'
            >
              <div className='flex items-center gap-2.5'>
                <SiBilibili
                  className='size-7 shrink-0 text-[#00aeec]'
                  aria-hidden='true'
                />
                <div className='min-w-0'>
                  <p className='text-muted-foreground text-xs font-medium'>
                    Bilibili
                  </p>
                  <h3 className='text-base font-semibold sm:text-lg'>
                    DPCC API
                  </h3>
                </div>
              </div>

              <div className='flex flex-1 flex-col items-center justify-center py-12 text-center sm:py-16'>
                <SiBilibili
                  className='size-20 text-[#00aeec]'
                  aria-hidden='true'
                />
                <p className='mt-4 text-lg font-semibold'>
                  {t('Bilibili Official Account')}
                </p>
              </div>

              <Button
                variant='outline'
                className='w-full'
                render={
                  <a
                    href={BILIBILI_ACCOUNT_URL}
                    target='_blank'
                    rel='noopener noreferrer'
                  />
                }
              >
                {t('Visit Bilibili')}
                <ExternalLink data-icon='inline-end' aria-hidden='true' />
              </Button>
            </article>

            <figure
              data-official-account='douyin'
              className='border-border bg-card flex min-w-0 flex-col rounded-md border p-5 shadow-xs sm:p-6'
            >
              <div className='flex items-center gap-2.5'>
                <SiTiktok
                  className='text-foreground size-7 shrink-0'
                  aria-hidden='true'
                />
                <div className='min-w-0'>
                  <p className='text-muted-foreground text-xs font-medium'>
                    Douyin
                  </p>
                  <h3 className='text-base font-semibold sm:text-lg'>
                    DPCC API
                  </h3>
                </div>
              </div>

              <div className='mx-auto mt-5 w-full max-w-[240px] overflow-hidden rounded-sm bg-white ring-1 ring-black/10'>
                <img
                  src={douyinOfficialQr}
                  alt={t('Douyin official account QR code')}
                  className='h-auto w-full object-contain'
                  draggable={false}
                />
              </div>

              <figcaption className='mt-5 text-center'>
                <p className='font-medium'>
                  {t('Douyin ID: {{id}}', { id: DOUYIN_ACCOUNT_ID })}
                </p>
                <p className='text-muted-foreground mt-1 text-sm leading-6'>
                  {t('Scan with Douyin to follow the official account.')}
                </p>
                <p className='text-muted-foreground mt-2 text-sm leading-6'>
                  {t(
                    'Occasional live streams with free Agent tool setup, enterprise Agent customization, and AI knowledge analysis.'
                  )}
                </p>
              </figcaption>
            </figure>
          </div>
        </section>
      </div>
    </PublicLayout>
  )
}
