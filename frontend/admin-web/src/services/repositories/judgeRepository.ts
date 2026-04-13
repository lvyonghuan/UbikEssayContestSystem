import { adminClient } from '@/services/http/client'
import { assertBlobNotApiError, parseAttachmentFilename } from '@/services/http/blob'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  ContestDailySubmissionStat,
  ContestReviewRegenerateResult,
  ContestTrackStatusStat,
  DashboardOverview,
  JudgeAccountInput,
  JudgeDeadlinePayload,
  JudgeProfile,
  JudgeProgressStat,
  ReviewEvent,
  ReviewEventInput,
  ReviewEventProgress,
  ReviewResult,
  TrackRankItem,
  WorkReviewStatus,
} from '@/types/api'

function normalizeList<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : []
}

export async function createJudgeAccount(payload: JudgeAccountInput) {
  const { data } = await adminClient.post<ApiResponse<JudgeProfile>>('/admin/judge/account', payload)
  return unwrapResponse(data)
}

export async function batchCreateJudgeAccounts(judges: JudgeAccountInput[]) {
  const { data } = await adminClient.post<ApiResponse<JudgeProfile[]>>('/admin/judge/accounts', { judges })
  return normalizeList<JudgeProfile>(unwrapResponse(data))
}

export async function updateJudgeAccount(judgeId: number, payload: JudgeAccountInput) {
  const { data } = await adminClient.put<ApiResponse<null>>(`/admin/judge/${judgeId}`, payload)
  unwrapResponse(data)
}

export async function deleteJudgeAccount(judgeId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/judge/${judgeId}`)
  unwrapResponse(data)
}

export async function createReviewEvent(payload: ReviewEventInput) {
  const { data } = await adminClient.post<ApiResponse<ReviewEvent>>('/admin/judge/review/event', payload)
  return unwrapResponse(data)
}

export async function updateReviewEvent(eventId: number, payload: ReviewEventInput) {
  const { data } = await adminClient.put<ApiResponse<null>>(`/admin/judge/review/${eventId}`, payload)
  unwrapResponse(data)
}

export async function assignReviewEventJudges(eventId: number, judgeIDs: number[]) {
  const { data } = await adminClient.put<ApiResponse<null>>(`/admin/judge/review/${eventId}/assign`, { judgeIDs })
  unwrapResponse(data)
}

export async function deleteReviewEvent(eventId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/judge/review/${eventId}`)
  unwrapResponse(data)
}

export async function fetchReviewEventProgress(eventId: number) {
  const { data } = await adminClient.get<ApiResponse<ReviewEventProgress>>(`/admin/judge/review/${eventId}`)
  return unwrapResponse(data)
}

export async function fetchTrackStatuses(trackId: number) {
  const { data } = await adminClient.get<ApiResponse<string[]>>(`/admin/judge/review/track/${trackId}/status`)
  return normalizeList<string>(unwrapResponse(data))
}

export async function fetchWorkReviewStatus(workId: number) {
  const { data } = await adminClient.get<ApiResponse<WorkReviewStatus>>(`/admin/judge/review/status/${workId}`)
  return unwrapResponse(data)
}

export async function fetchWorkReviewResults(workId: number) {
  const { data } = await adminClient.get<ApiResponse<ReviewResult[]>>(`/admin/judge/review/result/${workId}`)
  return normalizeList<ReviewResult>(unwrapResponse(data))
}

export async function regenerateWorkReviewResults(workId: number) {
  const { data } = await adminClient.post<ApiResponse<ReviewResult[]>>(`/admin/judge/review/result/${workId}/gen`)
  return normalizeList<ReviewResult>(unwrapResponse(data))
}

export async function fetchTrackReviewRanking(trackId: number) {
  const { data } = await adminClient.get<ApiResponse<TrackRankItem[]>>(`/admin/judge/review/rank/${trackId}`)
  return normalizeList<TrackRankItem>(unwrapResponse(data))
}

export async function exportTrackReviewExcel(trackId: number) {
  const response = await adminClient.get<Blob>(`/admin/judge/review/export/${trackId}`, {
    params: { format: 'xlsx' },
    responseType: 'blob',
  })

  await assertBlobNotApiError(response.data, response.headers['content-type'])
  const fallbackName = `track-${trackId}-review.xlsx`
  const filename = parseAttachmentFilename(response.headers['content-disposition']) || fallbackName
  return { blob: response.data, filename }
}

export async function fetchDashboardOverview() {
  const { data } = await adminClient.get<ApiResponse<DashboardOverview>>('/admin/dashboard/overview')
  return unwrapResponse(data)
}

export async function fetchContestTrackStatusStats(contestId: number) {
  const { data } = await adminClient.get<ApiResponse<ContestTrackStatusStat[]>>(
    `/admin/contests/${contestId}/stats/tracks-status`,
  )
  return normalizeList<ContestTrackStatusStat>(unwrapResponse(data))
}

export async function fetchContestDailySubmissionStats(contestId: number) {
  const { data } = await adminClient.get<ApiResponse<ContestDailySubmissionStat[]>>(
    `/admin/contests/${contestId}/stats/daily-submissions`,
  )
  return normalizeList<ContestDailySubmissionStat>(unwrapResponse(data))
}

export async function fetchContestJudgeProgressStats(contestId: number) {
  const { data } = await adminClient.get<ApiResponse<JudgeProgressStat[]>>(
    `/admin/contests/${contestId}/stats/judges-progress`,
  )
  return normalizeList<JudgeProgressStat>(unwrapResponse(data))
}

export async function regenerateContestReviewResults(contestId: number) {
  const { data } = await adminClient.post<ApiResponse<ContestReviewRegenerateResult>>(
    `/admin/review-results/generate/${contestId}`,
  )
  return unwrapResponse(data)
}

export async function updateReviewEventJudgeDeadline(eventId: number, judgeId: number, payload: JudgeDeadlinePayload) {
  const { data } = await adminClient.post<ApiResponse<null>>(
    `/admin/review-events/${eventId}/judges/${judgeId}/deadline`,
    payload,
  )
  unwrapResponse(data)
}
