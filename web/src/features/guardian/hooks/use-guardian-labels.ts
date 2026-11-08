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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import type {
  GuardianBillingPeriod,
  GuardianContactType,
  GuardianMacRegion,
  GuardianMacConfig,
  GuardianOrderRoute,
  GuardianOrderStatus,
  GuardianOs,
  GuardianServiceRoute,
} from '../types'

/** Display labels for every Guardian enum, shared by the page and the admin. */
export function useGuardianLabels() {
  const { t } = useTranslation()

  return useMemo(() => {
    const orderRoute: Record<GuardianOrderRoute, string> = {
      local: t('Local machine'),
      vm: t('Virtual machine'),
      vps: t('Remote server'),
      mac_mini: t('Overseas Mac mini'),
      unsure: t('Not sure yet'),
    }
    const serviceRoute: Record<GuardianServiceRoute, string> = {
      any: t('Any route'),
      local: t('Local machine'),
      vm: t('Virtual machine'),
      vps: t('Remote server'),
      mac_mini: t('Overseas Mac mini'),
    }
    const os: Record<GuardianOs, string> = {
      macos: 'macOS',
      windows: 'Windows',
      linux: 'Linux',
      other: t('Other'),
    }
    const contactType: Record<GuardianContactType, string> = {
      telegram: 'Telegram',
      qq: 'QQ',
      wechat: t('WeChat'),
      email: t('Email'),
    }
    const status: Record<GuardianOrderStatus, string> = {
      pending_payment: t('Awaiting payment'),
      paid: t('Paid'),
      scheduled: t('Scheduled'),
      in_progress: t('In progress'),
      delivered: t('Delivered'),
      expired: t('Expired'),
      cancelled: t('Cancelled'),
      refunded: t('Refunded'),
    }
    const macRegion: Record<GuardianMacRegion, string> = {
      kr_seoul: t('South Korea · Seoul'),
      au_sydney: t('Australia · Sydney'),
    }
    const macConfig: Record<GuardianMacConfig, string> = {
      '16_256': t('16GB memory + 256GB storage'),
    }
    const billingPeriod: Record<GuardianBillingPeriod, string> = {
      monthly: t('Monthly payment'),
      quarterly: t('Quarterly'),
      yearly: t('Yearly'),
    }
    return {
      billingPeriod,
      orderRoute,
      serviceRoute,
      os,
      contactType,
      status,
      macRegion,
      macConfig,
    }
  }, [t])
}
