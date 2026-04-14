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

export async function fetchAuthorByID(authorId: number) {
  const { data } = await adminClient.get<ApiResponse<Author>>(`/admin/authors/${authorId}`)
  return unwrapResponse(data)
}

export async function updateAuthor(authorId: number, payload: Author) {
  const { data } = await adminClient.put<ApiResponse<Author>>(`/admin/authors/${authorId}`, payload)
  return unwrapResponse(data)
}

export async function deleteAuthor(authorId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/authors/${authorId}`)
  unwrapResponse(data)
}
