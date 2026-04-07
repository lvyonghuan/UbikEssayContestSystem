import { describe, expect, it } from 'vitest'
import { clearTokenPair, getAccessToken, getRefreshToken, hasTokenPair, setTokenPair } from '@/services/auth/token'

describe('token service', () => {
  it('stores and reads token pair', () => {
    setTokenPair({ access_token: 'a', refresh_token: 'b' })

    expect(getAccessToken()).toBe('a')
    expect(getRefreshToken()).toBe('b')
    expect(hasTokenPair()).toBe(true)
  })

  it('clears token pair', () => {
    setTokenPair({ access_token: 'a', refresh_token: 'b' })
    clearTokenPair()

    expect(getAccessToken()).toBeNull()
    expect(getRefreshToken()).toBeNull()
    expect(hasTokenPair()).toBe(false)
  })
})
