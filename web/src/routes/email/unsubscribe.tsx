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
import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { z } from 'zod'

import { EmailUnsubscribe } from '@/features/email-campaigns/unsubscribe'

export const Route = createFileRoute('/email/unsubscribe')({
  validateSearch: z.object({
    token: z.string().max(512).optional().catch(undefined),
  }),
  component: UnsubscribeRoute,
})

function UnsubscribeRoute() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const [token] = useState(
    () =>
      search.token ||
      new URLSearchParams(window.location.hash.slice(1)).get('token') ||
      ''
  )
  useEffect(() => {
    void navigate({ search: {}, hash: '', replace: true })
  }, [navigate])
  return <EmailUnsubscribe token={token.length <= 512 ? token : ''} />
}
