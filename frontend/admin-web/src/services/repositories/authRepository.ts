import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { AdminLoginPayload, ApiResponse, TokenPair } from '@/types/api'

export async function login(payload: AdminLoginPayload) {
  const { data } = await adminClient.post<ApiResponse<TokenPair>>('/admin/login', payload)
  return unwrapResponse(data)
}

export async function refreshToken(refreshTokenValue: string) {
  const { data } = await adminClient.post<ApiResponse<TokenPair>>('/admin/refresh', null, {
    headers: {
      Authorization: `Bearer ${refreshTokenValue}`,
    },
  })
  return unwrapResponse(data)
}
