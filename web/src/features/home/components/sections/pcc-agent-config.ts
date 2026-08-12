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
export const PCC_AGENT_STORE_URL =
  'https://apps.microsoft.com/detail/9pf5ff13cbhp?hl=zh-CN&gl=CN'

export const PCC_AGENT_AUTOPLAY_OPTIONS = {
  delay: 3000,
  playOnInit: true,
  stopOnFocusIn: true,
  stopOnInteraction: false,
  stopOnMouseEnter: true,
} as const

export const PCC_AGENT_SECTION_CLASSES = {
  container: 'mx-auto max-w-6xl',
  layout:
    'grid items-center gap-12 lg:grid-cols-[minmax(0,0.78fr)_minmax(0,1.5fr)] lg:gap-16',
  media: 'aspect-[3/2] w-full overflow-hidden rounded-lg bg-sidebar',
  image: 'block h-full w-full select-none rounded-[inherit] object-contain',
} as const

export const PCC_AGENT_SLIDE_SOURCES = [
  '/landing/pcc-agent-marketplace-generated.webp',
  '/landing/pcc-agent-workspace-generated.webp',
  '/landing/pcc-agent-analytics-generated.webp',
] as const

export const PCC_AGENT_SLIDE_DIMENSIONS = {
  width: 1536,
  height: 1024,
} as const
