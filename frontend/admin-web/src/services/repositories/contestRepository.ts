import { adminClient, systemClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Contest } from '@/types/api'
import { toRfc3339 } from '@/utils/date'

function normalizeContestPayload(payload: Contest): Contest {
  return {
    ...payload,
    contestStartDate: toRfc3339(payload.contestStartDate),
    contestEndDate: toRfc3339(payload.contestEndDate),
  }
}

export async function fetchContests() {
  const { data } = await systemClient.get<ApiResponse<Contest[]>>('/contests/')
  const msg = unwrapResponse(data) || []
  return Array.isArray(msg) ? msg : []
}

export async function createContest(payload: Contest) {
  const { data } = await adminClient.post<ApiResponse<Contest>>('/admin/contest/', normalizeContestPayload(payload))
  return unwrapResponse(data)
}

export async function updateContest(contestId: number, payload: Contest) {
  const { data } = await adminClient.put<ApiResponse<Contest>>(`/admin/contest/${contestId}/`, normalizeContestPayload(payload))
  return unwrapResponse(data)
}

export async function removeContest(contestId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/contest/${contestId}/`)
  unwrapResponse(data)
}
