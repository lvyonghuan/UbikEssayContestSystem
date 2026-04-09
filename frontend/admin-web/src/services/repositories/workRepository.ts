import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, Work, WorkQueryParams } from '@/types/api'

function normalizeWorkList(msg: unknown): Work[] {
  return Array.isArray(msg) ? (msg as Work[]) : []
}

function buildWorkQueryParams(params: WorkQueryParams) {
  const query: Record<string, string | number> = {}

  if (typeof params.trackID === 'number') {
    query.track_id = params.trackID
  }
  if (params.workTitle && params.workTitle.trim()) {
    query.work_title = params.workTitle.trim()
  }
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

export async function fetchWorks(params: WorkQueryParams = {}) {
  const { data } = await adminClient.get<ApiResponse<Work[]>>('/admin/works', {
    params: buildWorkQueryParams(params),
  })
  return normalizeWorkList(unwrapResponse(data))
}

export async function fetchWorksByTrack(trackId: number) {
  return fetchWorks({ trackID: trackId, limit: 100 })
}

export async function fetchWorksByAuthorName(authorName: string) {
  return fetchWorks({ authorName, limit: 100 })
}

export async function fetchWorkByID(workId: number) {
  const { data } = await adminClient.get<ApiResponse<Work>>(`/admin/works/${workId}`)
  return unwrapResponse(data)
}

export async function removeWork(workId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/works/${workId}`)
  unwrapResponse(data)
}

export async function downloadWorkFile(workId: number) {
  const { data } = await adminClient.get<Blob>(`/admin/works/${workId}/file`, {
    responseType: 'blob',
  })
  return data
}
