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
import {
  enUS,
  fr,
  ja,
  ru,
  vi,
  zhCN,
  zhTW,
  type DayPickerLocale,
} from 'react-day-picker/locale'

const calendarLocales: Record<string, DayPickerLocale> = {
  en: enUS,
  fr,
  ja,
  ru,
  vi,
}

export function getDatePickerLocale(language: string): DayPickerLocale {
  const normalizedLanguage = language.toLowerCase()
  if (normalizedLanguage === 'zh-tw') return zhTW
  if (normalizedLanguage.startsWith('zh')) return zhCN

  const baseLanguage = normalizedLanguage.split('-')[0]
  return calendarLocales[baseLanguage] ?? enUS
}
