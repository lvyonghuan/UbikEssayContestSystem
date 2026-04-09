import axios from 'axios'
import { getAccessToken } from '@/services/auth/token'
import { submissionClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  JsonObject,
  SubmissionPayload,
  SubmissionUpdatePayload,
  Work,
} from '@/types/api'
import { extractAuthorIdentityFromToken } from '@/utils/jwt'

function normalizeWorkList(msg: unknown) {
  return Array.isArray(msg) ? (msg as Work[]) : []
}

function toWorkPayload(payload: SubmissionPayload | SubmissionUpdatePayload) {
  const base: Work = {
    workTitle: payload.workTitle,
    trackID: payload.trackID,
    workInfos: payload.workInfos,
  }

  if ('workID' in payload) {
    base.workID = payload.workID
  }

  return base
}

function shouldFallbackToLegacyPath(error: unknown) {
  if (!axios.isAxiosError(error)) {
    return false
  }
  const status = error.response?.status
  return status === 404 || status === 405
}

async function submitWithJson(method: 'post' | 'put', work: Work) {
  if (method === 'post') {
    const { data } = await submissionClient.post<ApiResponse<Work>>('/author/submission', work)
    return unwrapResponse(data)
  }

  const { data } = await submissionClient.put<ApiResponse<Work>>('/author/submission', work)
  return unwrapResponse(data)
}

async function submitWithFormData(method: 'post' | 'put', work: Work, file: File) {
  const formData = new FormData()
  formData.append('work', JSON.stringify(work))
  formData.append('article_file', file)

  if (method === 'post') {
    const { data } = await submissionClient.post<ApiResponse<Work>>('/author/submission', formData)
    return unwrapResponse(data)
  }

  const { data } = await submissionClient.put<ApiResponse<Work>>('/author/submission', formData)
  return unwrapResponse(data)
}

export async function uploadSubmissionFile(workID: number, file: File) {
  const formData = new FormData()
  formData.append('work_id', String(workID))
  formData.append('article_file', file)

  const { data } = await submissionClient.post<ApiResponse<unknown>>('/author/submission/file', formData)
  return unwrapResponse(data)
}

export async function fetchMySubmissions() {
  try {
    const { data } = await submissionClient.get<ApiResponse<Work[]>>('/author/submission')
    return normalizeWorkList(unwrapResponse(data))
  } catch (error) {
    if (!shouldFallbackToLegacyPath(error)) {
      throw error
    }

    const token = getAccessToken()
    const { authorID } = extractAuthorIdentityFromToken(token)
    if (!authorID) {
      throw error
    }

    const { data } = await submissionClient.get<ApiResponse<Work[]>>(`/author/submission/${authorID}`)
    return normalizeWorkList(unwrapResponse(data))
  }
}

export async function fetchSubmissionByID(workID: number) {
  const workList = await fetchMySubmissions()
  return workList.find((work) => work.workID === workID) || null
}

export async function createSubmission(payload: SubmissionPayload, file?: File) {
  const workPayload = toWorkPayload(payload)

  if (!file) {
    return submitWithJson('post', workPayload)
  }

  try {
    return await submitWithFormData('post', workPayload, file)
  } catch (error) {
    const saved = await submitWithJson('post', workPayload)
    if (saved.workID) {
      await uploadSubmissionFile(saved.workID, file)
    }
    return saved
  }
}

export async function updateSubmission(payload: SubmissionUpdatePayload, file?: File) {
  const workPayload = toWorkPayload(payload)

  if (!file) {
    return submitWithJson('put', workPayload)
  }

  try {
    return await submitWithFormData('put', workPayload, file)
  } catch (error) {
    const updated = await submitWithJson('put', workPayload)
    if (updated.workID) {
      await uploadSubmissionFile(updated.workID, file)
    }
    return updated
  }
}

export async function removeSubmission(workID: number) {
  const payload: Work = { workID }
  const { data } = await submissionClient.delete<ApiResponse<string | JsonObject>>('/author/submission', {
    data: payload,
  })
  return unwrapResponse(data)
}
