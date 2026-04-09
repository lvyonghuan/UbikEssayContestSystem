import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Author, AuthorQueryParams } from '@/types/api'

function normalizeAuthorList(msg: unknown): Author[] {
  return Array.isArray(msg) ? (msg as Author[]) : []
}

function buildAuthorQueryParams(params: AuthorQueryParams) {
  const query: Record<string, string | number> = {}
  if (params.authorName && params.authorName.trim()) {
    query.author_name = params.authorName.trim()
  }
  if (typeof params.offset === 'number') {
    query.offset = params.offset
  }
  if (typeof params.limit === 'number') {
    query.limit = params.limit
  }
  return query
}

export async function fetchAuthors(params: AuthorQueryParams = {}) {
  const { data } = await adminClient.get<ApiResponse<Author[]>>('/admin/authors', {
    params: buildAuthorQueryParams(params),
  })
  return normalizeAuthorList(unwrapResponse(data))
}
