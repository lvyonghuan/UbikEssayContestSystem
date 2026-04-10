import { describe, expect, it } from 'vitest'
import { calculateFileSHA256, calculateSHA256FromArrayBuffer } from '@/utils/hash'

describe('hash utils', () => {
  it('calculates SHA-256 from array buffer', async () => {
    const source = new TextEncoder().encode('hello').buffer
    const hash = await calculateSHA256FromArrayBuffer(source)
    expect(hash).toBe('2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824')
  })

  it('calculates SHA-256 from file', async () => {
    const file = new File(['hello'], 'hello.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    const hash = await calculateFileSHA256(file)
    expect(hash).toBe('2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824')
  })
})
