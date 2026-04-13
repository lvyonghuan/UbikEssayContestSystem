import type { ApiResponse } from '@/types/api'
import { ApiError } from './errors'

export async function assertBlobNotApiError(blob: Blob, contentType?: string) {
  const normalizedType = (contentType || blob.type || '').toLowerCase()
  if (!normalizedType.includes('application/json')) {
    return
  }

  const body = await blob.text()
  try {
    const parsed = JSON.parse(body) as ApiResponse<unknown>
    if (typeof parsed?.code === 'number' && parsed.code !== 200) {
      const detail = typeof parsed.msg === 'string' ? parsed.msg : '业务请求失败'
      throw new ApiError(detail, { code: parsed.code, detail: parsed.msg })
    }
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new ApiError('文件下载失败', { detail: body })
  }
}

export function parseAttachmentFilename(contentDisposition?: string) {
  if (!contentDisposition) {
    return null
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1])
    } catch {
      return utf8Match[1]
    }
  }

  const plainMatch = contentDisposition.match(/filename="?([^\";]+)"?/i)
  return plainMatch?.[1] || null
}
