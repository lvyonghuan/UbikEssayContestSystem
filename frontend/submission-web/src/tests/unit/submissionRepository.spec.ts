import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/services/http/client', () => ({
  submissionClient: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

import { submissionClient } from '@/services/http/client'
import {
  downloadSubmissionFile,
  uploadSubmissionFile,
} from '@/services/repositories/submissionRepository'
import { calculateFileSHA256, calculateSHA256FromArrayBuffer } from '@/utils/hash'

describe('submission repository', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('appends file_hash when uploading submission file', async () => {
    vi.mocked(submissionClient.post).mockResolvedValue({
      data: { code: 200, msg: null },
    } as never)

    const file = new File(['upload-content'], 'paper.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    const expectedHash = await calculateFileSHA256(file)

    await uploadSubmissionFile(7, file)

    expect(submissionClient.post).toHaveBeenCalledTimes(1)
    const postArgs = vi.mocked(submissionClient.post).mock.calls[0]
    const formData = postArgs[1] as FormData

    expect(formData.get('work_id')).toBe('7')
    expect(formData.get('article_file')).toBe(file)
    expect(formData.get('file_hash')).toBe(expectedHash)
  })

  it('downloads file and parses response headers', async () => {
    const fileBlob = new Blob(['download-content'], {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    const fileHash = await calculateSHA256FromArrayBuffer(await fileBlob.arrayBuffer())

    vi.mocked(submissionClient.get).mockResolvedValue({
      data: fileBlob,
      headers: {
        'content-type': 'application/octet-stream',
        'content-disposition': 'attachment; filename="paper.docx"',
        'x-file-sha256': fileHash,
      },
    } as never)

    const result = await downloadSubmissionFile(9)

    expect(result.fileBlob).toBe(fileBlob)
    expect(result.fileName).toBe('paper.docx')
    expect(result.fileHashSHA256).toBe(fileHash)
  })

  it('throws business error when download returns json blob', async () => {
    const errorBlob = new Blob([JSON.stringify({ code: 403, msg: 'forbidden' })], {
      type: 'application/json',
    })

    vi.mocked(submissionClient.get).mockResolvedValue({
      data: errorBlob,
      headers: {
        'content-type': 'application/json',
      },
    } as never)

    await expect(downloadSubmissionFile(10)).rejects.toThrow('forbidden')
  })

  it('rejects download result when hash header is missing', async () => {
    const fileBlob = new Blob(['download-content'], {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })

    vi.mocked(submissionClient.get).mockResolvedValue({
      data: fileBlob,
      headers: {
        'content-type': 'application/octet-stream',
      },
    } as never)

    await expect(downloadSubmissionFile(11)).rejects.toThrow('完整性标识')
  })
})
