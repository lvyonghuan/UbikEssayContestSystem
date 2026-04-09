import type { ApiResponse } from '@/types/api'
import { ApiError } from './errors'

export function unwrapResponse<T>(response: ApiResponse<T>) {
  if (response.code !== 200) {
    const detail = typeof response.msg === 'string' ? response.msg : '业务请求失败'
    throw new ApiError(detail, { code: response.code, detail: response.msg })
  }
  return response.msg
}
