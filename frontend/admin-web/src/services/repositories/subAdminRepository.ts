import { adminClient } from '@/services/http/client'
import { unwrapResponse } from '@/services/http/response'
import type {
  ApiResponse,
  CreateSubAdminRequest,
  SubAdminCreateResult,
  SubAdminInfo,
  UpdateSubAdminPermissionsRequest,
} from '@/types/api'

function normalizeSubAdminList(msg: unknown): SubAdminInfo[] {
  return Array.isArray(msg) ? (msg as SubAdminInfo[]) : []
}

export async function fetchSubAdmins() {
  const { data } = await adminClient.get<ApiResponse<SubAdminInfo[]>>('/admin/sub-admins')
  return normalizeSubAdminList(unwrapResponse(data))
}

export async function createSubAdmin(payload: CreateSubAdminRequest) {
  const { data } = await adminClient.post<ApiResponse<SubAdminCreateResult>>('/admin/sub-admins', payload)
  return unwrapResponse(data)
}

export async function updateSubAdminPermissions(adminId: number, payload: UpdateSubAdminPermissionsRequest) {
  const { data } = await adminClient.put<ApiResponse<null>>(`/admin/sub-admins/${adminId}/permissions`, payload)
  unwrapResponse(data)
}

export async function disableSubAdmin(adminId: number) {
  const { data } = await adminClient.post<ApiResponse<null>>(`/admin/sub-admins/${adminId}/disable`, null)
  unwrapResponse(data)
}

export async function removeSubAdmin(adminId: number) {
  const { data } = await adminClient.delete<ApiResponse<null>>(`/admin/sub-admins/${adminId}`)
  unwrapResponse(data)
}
