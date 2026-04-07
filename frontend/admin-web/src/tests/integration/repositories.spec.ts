import { describe, expect, it } from 'vitest'
import { login } from '@/services/repositories/authRepository'
import { fetchContests } from '@/services/repositories/contestRepository'
import { fetchTracks } from '@/services/repositories/trackRepository'

describe('repositories with mock api', () => {
  it('can login and receive token pair', async () => {
    const token = await login({ adminName: 'superadmin', password: 'password' })
    expect(token.access_token).toBeTruthy()
    expect(token.refresh_token).toBeTruthy()
  })

  it('loads contests and tracks', async () => {
    const contests = await fetchContests()
    expect(contests.length).toBeGreaterThan(0)

    const contestId = contests[0].contestID || 1
    const tracks = await fetchTracks(contestId)
    expect(Array.isArray(tracks)).toBe(true)
  })
})
