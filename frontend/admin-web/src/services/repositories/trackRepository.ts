import { adminClient, systemClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Track } from '@/types/api'

export async function fetchTracks(contestId: number) {
  const { data } = await systemClient.get<ApiResponse<Track[]>>(`/tracks/${contestId}`)
  const msg = unwrapResponse(data) || []
  return Array.isArray(msg) ? msg : []
}

export async function createTrack(payload: Track) {
  const { data } = await adminClient.post<ApiResponse<Track>>('/admin/track', payload)
  return unwrapResponse(data)
}

export async function updateTrack(trackId: number, payload: Track) {
  const { data } = await adminClient.put<ApiResponse<Track>>(`/admin/track/${trackId}`, payload)
  return unwrapResponse(data)
}

export async function removeTrack(trackId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/track/${trackId}`)
  unwrapResponse(data)
}
