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
export const globalNotificationCenterLayout = {
  root: 'pointer-events-none fixed right-3 bottom-[calc(env(safe-area-inset-bottom,0px)+0.75rem)] z-[60] sm:right-4',
  panel:
    'bg-popover/95 text-popover-foreground pointer-events-auto w-[min(22rem,calc(100vw-5rem))] overflow-hidden rounded-lg border shadow-xl backdrop-blur-xl',
  preview:
    'border-destructive/30 bg-popover/95 text-popover-foreground pointer-events-auto w-[min(18rem,calc(100vw-5rem))] rounded-lg border px-3 py-2 text-left shadow-lg backdrop-blur-xl transition-colors hover:bg-muted focus-visible:ring-ring/50 focus-visible:ring-2 focus-visible:outline-none',
  trigger:
    'bg-popover/95 pointer-events-auto relative size-16 flex-col gap-1 rounded-lg p-0 shadow-lg backdrop-blur-xl',
  triggerIcon: 'size-7',
  feed: 'max-h-[min(52svh,24rem)] overflow-y-auto overscroll-contain',
  presenceMode: 'wait',
  motionOrigin: 'bottom right',
  motion: {
    initial: { opacity: 0, scale: 0.86 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 0.86 },
    transition: { duration: 0.18, ease: [0.16, 1, 0.3, 1] },
  },
} as const
