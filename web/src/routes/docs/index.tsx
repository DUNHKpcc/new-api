import { createFileRoute } from '@tanstack/react-router'

import { SiteDocs } from '@/features/site-docs'

export const Route = createFileRoute('/docs/')({
  validateSearch: (search: Record<string, unknown>): { article?: string } => ({
    article: typeof search.article === 'string' ? search.article : undefined,
  }),
  component: DocsPage,
})
function DocsPage() {
  const search = Route.useSearch()
  return <SiteDocs article={search.article} />
}
