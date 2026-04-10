import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  JsonObject,
  ScriptDefinition,
  ScriptDefinitionStatusPayload,
  ScriptVersion,
} from '@/types/api'

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function asJsonObject(value: unknown): JsonObject | undefined {
  return asRecord(value) as JsonObject | undefined
}

function normalizeScriptDefinition(item: unknown): ScriptDefinition | null {
  const record = asRecord(item)
  if (!record) {
    return null
  }

  const description = asString(record.description) ?? asString(record.scriptDescription)
  const meta = asJsonObject(record.meta) ?? asJsonObject(record.extensionData)

  return {
    scriptID: asNumber(record.scriptID),
    scriptKey: asString(record.scriptKey),
    scriptName: asString(record.scriptName) ?? '',
    description,
    scriptDescription: description,
    interpreter: asString(record.interpreter),
    isEnabled: asBoolean(record.isEnabled),
    activeVersionID: asNumber(record.activeVersionID),
    meta,
    extensionData: meta,
    createdAt: asString(record.createdAt),
    updatedAt: asString(record.updatedAt),
  }
}

function normalizeScriptVersion(item: unknown): ScriptVersion | null {
  const record = asRecord(item)
  if (!record) {
    return null
  }

  const versionNum = asNumber(record.versionNum)
  const checksum = asString(record.checksum) ?? asString(record.fileHash)
  return {
    versionID: asNumber(record.versionID),
    scriptID: asNumber(record.scriptID),
    versionNum,
    versionName: asString(record.versionName) ?? (versionNum ? `v${versionNum}` : undefined),
    fileName: asString(record.fileName),
    relativePath: asString(record.relativePath),
    fileURL: asString(record.fileURL),
    checksum,
    fileHash: checksum,
    createdBy: asNumber(record.createdBy),
    isActive: asBoolean(record.isActive),
    extensionData: asJsonObject(record.extensionData),
    createdAt: asString(record.createdAt),
  }
}

function normalizeScriptList(msg: unknown): ScriptDefinition[] {
  if (!Array.isArray(msg)) {
    return []
  }
  return msg.map((item) => normalizeScriptDefinition(item)).filter((item): item is ScriptDefinition => Boolean(item))
}

function normalizeVersionList(msg: unknown): ScriptVersion[] {
  if (!Array.isArray(msg)) {
    return []
  }
  return msg.map((item) => normalizeScriptVersion(item)).filter((item): item is ScriptVersion => Boolean(item))
}

function toScriptDefinitionPayload(payload: ScriptDefinition) {
  const description = payload.description ?? payload.scriptDescription
  const meta = payload.meta ?? payload.extensionData

  return {
    scriptName: payload.scriptName,
    scriptKey: payload.scriptKey,
    description,
    interpreter: payload.interpreter,
    isEnabled: payload.isEnabled,
    meta,
  }
}

export async function fetchScriptDefinitions() {
  const { data } = await adminClient.get<ApiResponse<ScriptDefinition[]>>('/admin/scripts')
  return normalizeScriptList(unwrapResponse(data))
}

export async function fetchScriptDefinitionByID(scriptId: number) {
  const { data } = await adminClient.get<ApiResponse<ScriptDefinition>>(`/admin/scripts/${scriptId}`)
  const normalized = normalizeScriptDefinition(unwrapResponse(data))
  if (!normalized) {
    throw new Error('脚本详情响应格式异常')
  }
  return normalized
}

export async function createScriptDefinition(payload: ScriptDefinition) {
  const { data } = await adminClient.post<ApiResponse<ScriptDefinition>>('/admin/scripts', toScriptDefinitionPayload(payload))
  const normalized = normalizeScriptDefinition(unwrapResponse(data))
  if (!normalized) {
    throw new Error('脚本创建响应格式异常')
  }
  return normalized
}

export async function updateScriptDefinition(scriptId: number, payload: ScriptDefinition) {
  const { data } = await adminClient.put<ApiResponse<ScriptDefinition>>(
    `/admin/scripts/${scriptId}`,
    toScriptDefinitionPayload(payload),
  )
  const normalized = normalizeScriptDefinition(unwrapResponse(data))
  if (!normalized) {
    throw new Error('脚本更新响应格式异常')
  }
  return normalized
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
