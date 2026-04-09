import { systemClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Contest } from '@/types/api'

function normalizeContestList(msg: unknown) {
  return Array.isArray(msg) ? (msg as Contest[]) : []
}

export async function fetchContests() {
  const { data } = await systemClient.get<ApiResponse<Contest[]>>('/contests')
  return normalizeContestList(unwrapResponse(data))
}

export async function fetchContestByID(contestID: number) {
  try {
    const { data } = await systemClient.get<ApiResponse<Contest>>(`/contests/${contestID}`)
    return unwrapResponse(data)
  } catch {
    const contests = await fetchContests()
    return contests.find((contest) => contest.contestID === contestID) || null
  }
}
