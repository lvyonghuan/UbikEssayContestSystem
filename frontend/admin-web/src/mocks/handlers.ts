import { http, HttpResponse } from 'msw'
import { mockContests, mockTokens, mockTracksByContest } from './data'

let nextContestId = 3
let nextTrackId = 300

export const handlers = [
  http.post('/api/admin/admin/login', async ({ request }) => {
    const body = await request.json() as { adminName?: string; password?: string }
    if (!body.adminName || !body.password) {
      return HttpResponse.json({ code: 400, msg: '用户名或密码不能为空' }, { status: 400 })
    }
    return HttpResponse.json({ code: 200, msg: mockTokens })
  }),

  http.post('/api/admin/admin/refresh', () => {
    return HttpResponse.json({ code: 200, msg: mockTokens })
  }),

  http.get('/api/system/contests/', () => {
    return HttpResponse.json({ code: 200, msg: mockContests })
  }),

  http.post('/api/admin/admin/contest/', async ({ request }) => {
    const payload = await request.json() as Record<string, unknown>
    const nextContest = { ...payload, contestID: nextContestId++ }
    mockContests.push(nextContest as never)
    return HttpResponse.json({ code: 200, msg: nextContest })
  }),

  http.put('/api/admin/admin/contest/:contestId/', async ({ params, request }) => {
    const contestId = Number(params.contestId)
    const payload = await request.json() as Record<string, unknown>
    const index = mockContests.findIndex((contest) => contest.contestID === contestId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '赛事不存在' }, { status: 404 })
    }
    mockContests[index] = { ...mockContests[index], ...payload, contestID: contestId }
    return HttpResponse.json({ code: 200, msg: mockContests[index] })
  }),

  http.delete('/api/admin/admin/contest/:contestId/', ({ params }) => {
    const contestId = Number(params.contestId)
    const index = mockContests.findIndex((contest) => contest.contestID === contestId)
    if (index >= 0) {
      mockContests.splice(index, 1)
      delete mockTracksByContest[contestId]
    }
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/system/tracks/:contestId', ({ params }) => {
    const contestId = Number(params.contestId)
    return HttpResponse.json({ code: 200, msg: mockTracksByContest[contestId] || [] })
  }),

  http.post('/api/admin/admin/track/', async ({ request }) => {
    const payload = await request.json() as Record<string, unknown>
    const contestId = Number(payload.contestID)
    const track = { ...payload, trackID: nextTrackId++ }
    if (!mockTracksByContest[contestId]) {
      mockTracksByContest[contestId] = []
    }
    mockTracksByContest[contestId].push(track as never)
    return HttpResponse.json({ code: 200, msg: track })
  }),

  http.put('/api/admin/admin/track/:trackId/', async ({ params, request }) => {
    const trackId = Number(params.trackId)
    const payload = await request.json() as Record<string, unknown>

    for (const contestId of Object.keys(mockTracksByContest)) {
      const tracks = mockTracksByContest[Number(contestId)]
      const targetIndex = tracks.findIndex((track) => track.trackID === trackId)
      if (targetIndex >= 0) {
        tracks[targetIndex] = { ...tracks[targetIndex], ...payload, trackID: trackId }
        return HttpResponse.json({ code: 200, msg: tracks[targetIndex] })
      }
    }

    return HttpResponse.json({ code: 404, msg: '赛道不存在' }, { status: 404 })
  }),

  http.delete('/api/admin/admin/track/:trackId/', ({ params }) => {
    const trackId = Number(params.trackId)

    for (const contestId of Object.keys(mockTracksByContest)) {
      const tracks = mockTracksByContest[Number(contestId)]
      const targetIndex = tracks.findIndex((track) => track.trackID === trackId)
      if (targetIndex >= 0) {
        tracks.splice(targetIndex, 1)
        break
      }
    }

    return HttpResponse.json({ code: 200, msg: null })
  }),
]
