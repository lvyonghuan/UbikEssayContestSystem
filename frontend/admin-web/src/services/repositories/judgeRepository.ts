import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type { ApiResponse, ContestJudgeBinding, JudgeProfile } from '@/types/api'

// 评委模块后端接口尚未发布到 Admin Swagger。
// 先按比赛维度预留数据访问层，待后端文档就绪后直接接入 UI。
export async function fetchContestJudges(contestId: number) {
  const { data } = await adminClient.get<ApiResponse<JudgeProfile[]>>(`/admin/contests/${contestId}/judges`)
  const msg = unwrapResponse(data)
  return Array.isArray(msg) ? (msg as JudgeProfile[]) : []
}

export async function bindJudgeToContest(payload: ContestJudgeBinding) {
  const { data } = await adminClient.post<ApiResponse<null>>(
    `/admin/contests/${payload.contestID}/judges/${payload.judgeID}`,
    { role: payload.role },
  )
  unwrapResponse(data)
}

export async function unbindJudgeFromContest(contestId: number, judgeId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/contests/${contestId}/judges/${judgeId}`)
  unwrapResponse(data)
}
