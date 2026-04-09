import { systemClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Track } from '@/types/api'

function normalizeTrackList(msg: unknown) {
  return Array.isArray(msg) ? (msg as Track[]) : []
}

export async function fetchTracksByContest(contestID: number) {
  const { data } = await systemClient.get<ApiResponse<Track[]>>(`/tracks/${contestID}`)
  return normalizeTrackList(unwrapResponse(data))
}

export async function fetchTrackByID(trackID: number) {
  const { data } = await systemClient.get<ApiResponse<Track>>(`/tracks/detail/${trackID}`)
  return unwrapResponse(data)
}
