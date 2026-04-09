import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  ScriptDefinition,
  ScriptDefinitionStatusPayload,
  ScriptVersion,
} from '@/types/api'

function normalizeScriptList(msg: unknown): ScriptDefinition[] {
  return Array.isArray(msg) ? (msg as ScriptDefinition[]) : []
}

function normalizeVersionList(msg: unknown): ScriptVersion[] {
  return Array.isArray(msg) ? (msg as ScriptVersion[]) : []
}

export async function fetchScriptDefinitions() {
  const { data } = await adminClient.get<ApiResponse<ScriptDefinition[]>>('/admin/scripts')
  return normalizeScriptList(unwrapResponse(data))
}

export async function fetchScriptDefinitionByID(scriptId: number) {
  const { data } = await adminClient.get<ApiResponse<ScriptDefinition>>(`/admin/scripts/${scriptId}`)
  return unwrapResponse(data)
}

export async function createScriptDefinition(payload: ScriptDefinition) {
  const { data } = await adminClient.post<ApiResponse<ScriptDefinition>>('/admin/scripts', payload)
  return unwrapResponse(data)
}

export async function updateScriptDefinition(scriptId: number, payload: ScriptDefinition) {
  const { data } = await adminClient.put<ApiResponse<ScriptDefinition>>(`/admin/scripts/${scriptId}`, payload)
  return unwrapResponse(data)
}

export async function updateScriptDefinitionStatus(scriptId: number, payload: ScriptDefinitionStatusPayload) {
  const { data } = await adminClient.post<ApiResponse<null>>(`/admin/scripts/${scriptId}/status`, payload)
  unwrapResponse(data)
}

export async function uploadScriptVersion(scriptId: number, file: File) {
  const formData = new FormData()
  formData.append('script_file', file)

  const { data } = await adminClient.post<ApiResponse<ScriptVersion>>(
    `/admin/scripts/${scriptId}/versions/upload`,
    formData,
  )
  return unwrapResponse(data)
}

export async function fetchScriptVersions(scriptId: number) {
  const { data } = await adminClient.get<ApiResponse<ScriptVersion[]>>(`/admin/scripts/${scriptId}/versions`)
  return normalizeVersionList(unwrapResponse(data))
}

export async function activateScriptVersion(scriptId: number, versionId: number) {
  const { data } = await adminClient.post<ApiResponse<null>>(
    `/admin/scripts/${scriptId}/versions/${versionId}/activate`,
    null,
  )
  unwrapResponse(data)
}
