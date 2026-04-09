import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  FlowMount,
  FlowStep,
  ScriptFlow,
  ScriptFlowStatusPayload,
} from '@/types/api'

function normalizeFlowList(msg: unknown): ScriptFlow[] {
  return Array.isArray(msg) ? (msg as ScriptFlow[]) : []
}

function normalizeStepList(msg: unknown): FlowStep[] {
  return Array.isArray(msg) ? (msg as FlowStep[]) : []
}

function normalizeMountList(msg: unknown): FlowMount[] {
  return Array.isArray(msg) ? (msg as FlowMount[]) : []
}

export async function fetchScriptFlows() {
  const { data } = await adminClient.get<ApiResponse<ScriptFlow[]>>('/admin/script-flows')
  return normalizeFlowList(unwrapResponse(data))
}

export async function fetchScriptFlowByID(flowId: number) {
  const { data } = await adminClient.get<ApiResponse<ScriptFlow>>(`/admin/script-flows/${flowId}`)
  return unwrapResponse(data)
}

export async function createScriptFlow(payload: ScriptFlow) {
  const { data } = await adminClient.post<ApiResponse<ScriptFlow>>('/admin/script-flows', payload)
  return unwrapResponse(data)
}

export async function updateScriptFlow(flowId: number, payload: ScriptFlow) {
  const { data } = await adminClient.put<ApiResponse<ScriptFlow>>(`/admin/script-flows/${flowId}`, payload)
  return unwrapResponse(data)
}

export async function updateScriptFlowStatus(flowId: number, payload: ScriptFlowStatusPayload) {
  const { data } = await adminClient.post<ApiResponse<null>>(`/admin/script-flows/${flowId}/status`, payload)
  unwrapResponse(data)
}

export async function replaceFlowSteps(flowId: number, payload: FlowStep[]) {
  const { data } = await adminClient.put<ApiResponse<null>>(`/admin/script-flows/${flowId}/steps`, payload)
  unwrapResponse(data)
}

export async function fetchFlowSteps(flowId: number) {
  const { data } = await adminClient.get<ApiResponse<FlowStep[]>>(`/admin/script-flows/${flowId}/steps`)
  return normalizeStepList(unwrapResponse(data))
}

export async function createFlowMount(payload: FlowMount) {
  const { data } = await adminClient.post<ApiResponse<FlowMount>>('/admin/script-flows/mounts', payload)
  return unwrapResponse(data)
}

export async function removeFlowMount(mountId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/script-flows/mounts/${mountId}`)
  unwrapResponse(data)
}

export async function fetchFlowMounts(flowId: number) {
  const { data } = await adminClient.get<ApiResponse<FlowMount[]>>(`/admin/script-flows/${flowId}/mounts`)
  return normalizeMountList(unwrapResponse(data))
}
