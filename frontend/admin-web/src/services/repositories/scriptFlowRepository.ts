import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  FlowMount,
  FlowMountScope,
  FlowStep,
  JsonObject,
  ScriptFlow,
  ScriptFlowStatusPayload,
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

function asMountScope(value: unknown): FlowMountScope | undefined {
  if (value === 'global' || value === 'contest' || value === 'track') {
    return value
  }
  return undefined
}

function normalizeFlow(item: unknown): ScriptFlow | null {
  const record = asRecord(item)
  if (!record) {
    return null
  }

  const description = asString(record.description) ?? asString(record.flowDescription)
  const meta = asJsonObject(record.meta) ?? asJsonObject(record.extensionData)

  return {
    flowID: asNumber(record.flowID),
    flowKey: asString(record.flowKey),
    flowName: asString(record.flowName) ?? '',
    description,
    flowDescription: description,
    isEnabled: asBoolean(record.isEnabled),
    meta,
    extensionData: meta,
    createdAt: asString(record.createdAt),
    updatedAt: asString(record.updatedAt),
  }
}

function normalizeStep(item: unknown): FlowStep | null {
  const record = asRecord(item)
  if (!record) {
    return null
  }

  const inputTemplate = asJsonObject(record.inputTemplate) ?? asJsonObject(record.stepConfig)

  return {
    stepID: asNumber(record.stepID),
    flowID: asNumber(record.flowID),
    stepOrder: asNumber(record.stepOrder) ?? 0,
    stepName: asString(record.stepName),
    scriptID: asNumber(record.scriptID),
    scriptVersionID: asNumber(record.scriptVersionID),
    isEnabled: asBoolean(record.isEnabled),
    failureStrategy: asString(record.failureStrategy),
    inputTemplate,
    timeoutMs: asNumber(record.timeoutMs),
    stepConfig: inputTemplate,
    extensionData: asJsonObject(record.extensionData),
  }
}

function inferScopeFromTarget(targetType: string | undefined): FlowMountScope {
  if (targetType === 'contest') {
    return 'contest'
  }
  if (targetType === 'track') {
    return 'track'
  }
  return 'global'
}

function normalizeMount(item: unknown): FlowMount | null {
  const record = asRecord(item)
  if (!record) {
    return null
  }

  const targetType = asString(record.targetType) ?? asString(record.containerType)
  const scope = asMountScope(record.scope) ?? inferScopeFromTarget(targetType)
  const targetID = asNumber(record.targetID) ?? asNumber(record.containerID)

  return {
    mountID: asNumber(record.mountID),
    flowID: asNumber(record.flowID),
    scope,
    targetType: targetType ?? (scope === 'global' ? 'global' : scope),
    targetID,
    eventKey: asString(record.eventKey),
    isEnabled: asBoolean(record.isEnabled),
    containerType: targetType ?? (scope === 'global' ? 'global' : scope),
    containerID: targetID,
    mountConfig: asJsonObject(record.mountConfig),
    extensionData: asJsonObject(record.extensionData),
    createdAt: asString(record.createdAt),
  }
}

function toFlowPayload(payload: ScriptFlow) {
  const description = payload.description ?? payload.flowDescription
  const meta = payload.meta ?? payload.extensionData

  return {
    flowName: payload.flowName,
    flowKey: payload.flowKey,
    description,
    isEnabled: payload.isEnabled,
    meta,
  }
}

function toStepPayload(payload: FlowStep[]) {
  return payload.map((item) => ({
    stepID: item.stepID,
    flowID: item.flowID,
    stepOrder: item.stepOrder,
    stepName: item.stepName,
    scriptID: item.scriptID,
    scriptVersionID: item.scriptVersionID,
    isEnabled: item.isEnabled,
    failureStrategy: item.failureStrategy,
    inputTemplate: item.inputTemplate ?? item.stepConfig,
    timeoutMs: item.timeoutMs,
  }))
}

function toMountPayload(payload: FlowMount) {
  const targetType = payload.targetType ?? payload.containerType
  const scope = payload.scope ?? inferScopeFromTarget(targetType)
  const fallbackTargetType = targetType ?? (scope === 'global' ? 'global' : scope)
  const rawTargetID = payload.targetID ?? payload.containerID
  const targetID = scope === 'global' ? rawTargetID ?? 0 : rawTargetID

  return {
    flowID: payload.flowID,
    scope,
    targetType: fallbackTargetType,
    targetID,
    eventKey: payload.eventKey,
    isEnabled: payload.isEnabled,
  }
}

function normalizeFlowList(msg: unknown): ScriptFlow[] {
  if (!Array.isArray(msg)) {
    return []
  }
  return msg.map((item) => normalizeFlow(item)).filter((item): item is ScriptFlow => Boolean(item))
}

function normalizeStepList(msg: unknown): FlowStep[] {
  if (!Array.isArray(msg)) {
    return []
  }
  return msg.map((item) => normalizeStep(item)).filter((item): item is FlowStep => Boolean(item))
}

function normalizeMountList(msg: unknown): FlowMount[] {
  if (!Array.isArray(msg)) {
    return []
  }
  return msg.map((item) => normalizeMount(item)).filter((item): item is FlowMount => Boolean(item))
}

export async function fetchScriptFlows() {
  const { data } = await adminClient.get<ApiResponse<ScriptFlow[]>>('/admin/script-flows')
  return normalizeFlowList(unwrapResponse(data))
}

export async function fetchScriptFlowByID(flowId: number) {
  const { data } = await adminClient.get<ApiResponse<ScriptFlow>>(`/admin/script-flows/${flowId}`)
  const normalized = normalizeFlow(unwrapResponse(data))
  if (!normalized) {
    throw new Error('流程详情响应格式异常')
  }
  return normalized
}

export async function createScriptFlow(payload: ScriptFlow) {
  const { data } = await adminClient.post<ApiResponse<ScriptFlow>>('/admin/script-flows', toFlowPayload(payload))
  const normalized = normalizeFlow(unwrapResponse(data))
  if (!normalized) {
    throw new Error('流程创建响应格式异常')
  }
  return normalized
}

export async function updateScriptFlow(flowId: number, payload: ScriptFlow) {
  const { data } = await adminClient.put<ApiResponse<ScriptFlow>>(
    `/admin/script-flows/${flowId}`,
    toFlowPayload(payload),
  )
  const normalized = normalizeFlow(unwrapResponse(data))
  if (!normalized) {
    throw new Error('流程更新响应格式异常')
  }
  return normalized
}

export async function updateScriptFlowStatus(flowId: number, payload: ScriptFlowStatusPayload) {
  const { data } = await adminClient.post<ApiResponse<null>>(`/admin/script-flows/${flowId}/status`, payload)
  unwrapResponse(data)
}

export async function replaceFlowSteps(flowId: number, payload: FlowStep[]) {
  const { data } = await adminClient.put<ApiResponse<null>>(`/admin/script-flows/${flowId}/steps`, toStepPayload(payload))
  unwrapResponse(data)
}

export async function fetchFlowSteps(flowId: number) {
  const { data } = await adminClient.get<ApiResponse<FlowStep[]>>(`/admin/script-flows/${flowId}/steps`)
  return normalizeStepList(unwrapResponse(data))
}

export async function createFlowMount(payload: FlowMount) {
  const { data } = await adminClient.post<ApiResponse<FlowMount>>('/admin/script-flows/mounts', toMountPayload(payload))
  const normalized = normalizeMount(unwrapResponse(data))
  if (!normalized) {
    throw new Error('流程挂载响应格式异常')
  }
  return normalized
}

export async function removeFlowMount(mountId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/script-flows/mounts/${mountId}`)
  unwrapResponse(data)
}

export async function fetchFlowMounts(flowId: number) {
  const { data } = await adminClient.get<ApiResponse<FlowMount[]>>(`/admin/script-flows/${flowId}/mounts`)
  return normalizeMountList(unwrapResponse(data))
}
