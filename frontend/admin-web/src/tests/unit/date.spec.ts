import { describe, expect, it } from 'vitest'
import { toRfc3339 } from '@/utils/date'

describe('toRfc3339', () => {
  it('converts date to RFC3339', () => {
    const result = toRfc3339('2026-04-05')
    expect(result).toMatch(/^2026-04-05T00:00:00[+-]\d{2}:\d{2}$/)
  })

  it('converts datetime with blank separator', () => {
    const result = toRfc3339('2026-04-05 10:11')
    expect(result).toMatch(/^2026-04-05T10:11:00[+-]\d{2}:\d{2}$/)
  })

  it('keeps valid RFC3339 string', () => {
    const value = '2026-04-05T10:11:12+08:00'
    expect(toRfc3339(value)).toBe(value)
  })
})
