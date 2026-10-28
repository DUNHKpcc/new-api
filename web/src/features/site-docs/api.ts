import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export interface SiteDocument {
  slug: string
  title: string
  category: string
  content?: string
  published: boolean
  sort_order: number
  version: number
  updated_at: number
}
interface Response<T> {
  success: boolean
  message: string
  data: T
}
export async function listDocuments(admin = false) {
  return requireServerSuccess(
    (
      await api.get<Response<SiteDocument[]>>(
        admin ? '/api/site-docs-admin' : '/api/site-docs'
      )
    ).data
  ).data
}
export async function getDocument(slug: string, admin = false) {
  return requireServerSuccess(
    (
      await api.get<Response<SiteDocument>>(
        `${admin ? '/api/site-docs-admin' : '/api/site-docs'}/${encodeURIComponent(slug)}`
      )
    ).data
  ).data
}
export async function saveDocument(document: SiteDocument, create: boolean) {
  const response = create
    ? await api.post<Response<SiteDocument>>('/api/site-docs-admin', document)
    : await api.put<Response<SiteDocument>>(
        `/api/site-docs-admin/${encodeURIComponent(document.slug)}`,
        document
      )
  return requireServerSuccess(response.data).data
}

export async function uploadDocumentImage(image: string) {
  return requireServerSuccess(
    (
      await api.post<Response<{ url: string; size: number }>>(
        '/api/site-docs-admin/images',
        { image }
      )
    ).data
  ).data
}
