import { submissionClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Author, AuthorLoginPayload, TokenPair } from '@/types/api'

function normalizeLoginPayload(payload: AuthorLoginPayload) {
  const body: Partial<Author> & { password: string } = {
    password: payload.password,
  }

  if (payload.authorName?.trim()) {
    body.authorName = payload.authorName.trim()
  }
  if (payload.authorEmail?.trim()) {
    body.authorEmail = payload.authorEmail.trim()
  }

  return body
}

export async function login(payload: AuthorLoginPayload) {
  const { data } = await submissionClient.post<ApiResponse<TokenPair>>('/author/login', normalizeLoginPayload(payload))
  return unwrapResponse(data)
}

export async function register(payload: Author) {
  const { data } = await submissionClient.post<ApiResponse<unknown>>('/author/register', payload)
  return unwrapResponse(data)
}

export async function refreshToken(refreshTokenValue: string) {
  const { data } = await submissionClient.get<ApiResponse<TokenPair>>('/author/refresh', {
    headers: {
      Authorization: `Bearer ${refreshTokenValue}`,
    },
  })
  return unwrapResponse(data)
}

export async function updateAuthorProfile(payload: Author) {
  const { data } = await submissionClient.put<ApiResponse<Author>>('/author', payload)
  return unwrapResponse(data)
}
