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

/**
 * Compatibility export for the pre-security-page profile tab.
 *
 * Account binding now lives in the security surface so both routes share the
 * same proof-backed bind/unbind flow. Keeping this alias avoids breaking any
 * extension that still imports the legacy profile component path.
 */
export { AccountBindings as AccountBindingsTab } from '@/features/security/components/account-bindings'
