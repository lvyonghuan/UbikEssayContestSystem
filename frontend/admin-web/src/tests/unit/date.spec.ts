import { describe, expect, it } from 'vitest'
import { formatChinaDateTime, toChinaPickerValue, toRfc3339 } from '@/utils/date'

describe('toRfc3339', () => {
  it('converts date to RFC3339', () => {
    const result = toRfc3339('2026-04-05')
    expect(result).toBe('2026-04-05T00:00:00+08:00')
  })

  it('converts datetime with blank separator', () => {
    const result = toRfc3339('2026-04-05 10:11')
    expect(result).toBe('2026-04-05T10:11:00+08:00')
  })

  it('keeps valid RFC3339 string', () => {
    const value = '2026-04-05T10:11:12+08:00'
    expect(toRfc3339(value)).toBe(value)
  })

  it('converts UTC RFC3339 to China offset', () => {
    const value = '2026-04-05T02:11:12Z'
    expect(toRfc3339(value)).toBe('2026-04-05T10:11:12+08:00')
  })
})

describe('china datetime helpers', () => {
  it('formats UTC time as China local time', () => {
    expect(formatChinaDateTime('2026-04-05T02:11:00Z')).toBe('2026-04-05 10:11')
  })

  it('keeps picker value in YYYY-MM-DD HH:mm', () => {
    expect(toChinaPickerValue('2026-04-05T02:11:12Z')).toBe('2026-04-05 10:11')
  })
})
