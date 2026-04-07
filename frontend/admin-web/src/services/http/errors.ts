import axios from 'axios'

export class ApiError extends Error {
  readonly status?: number
  readonly code?: number
  readonly detail?: unknown

  constructor(message: string, options: { status?: number; code?: number; detail?: unknown } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = options.status
    this.code = options.code
    this.detail = options.detail
  }
}

export function normalizeError(error: unknown): ApiError {
  if (axios.isAxiosError(error)) {
    const status = error.response?.status
    const code = error.response?.data?.code
    const message = error.response?.data?.msg || error.message || '请求失败'
    return new ApiError(message, { status, code, detail: error.response?.data })
  }

  if (error instanceof Error) {
    return new ApiError(error.message)
  }

  return new ApiError('未知错误')
}
