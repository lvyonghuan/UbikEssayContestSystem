import { describe, expect, it } from 'vitest'
import { unwrapResponse } from '@/services/http/response'

describe('unwrapResponse', () => {
  it('returns msg when code is 200', () => {
    const result = unwrapResponse({ code: 200, msg: { ok: true } })
    expect(result).toEqual({ ok: true })
  })

  it('throws when business code is not 200', () => {
    expect(() => unwrapResponse({ code: 500, msg: 'backend failed' })).toThrow('backend failed')
  })
})
