import { http, HttpResponse } from 'msw'
import type { FlowMount, FlowStep, ReviewEvent, ReviewResult, ScriptVersion, Work } from '@/types/api'
import {
  getAllMockWorks,
  mockAuthors,
  mockContests,
  mockFlowMountsByFlow,
  mockFlowStepsByFlow,
  mockFlows,
  mockJudgeProfiles,
  mockReviewEvents,
  mockReviewResultsByWork,
  mockScripts,
  mockSubAdmins,
  mockScriptVersionsByScript,
  mockTokens,
  mockTracksByContest,
  mockWorksByTrack,
} from './data'

let nextContestId = 3
let nextTrackId = 300
let nextScriptId = 4
let nextVersionId = 40
let nextFlowId = 3
let nextStepId = 1000
let nextMountId = 10
let nextSubAdminId = 20
let nextJudgeId = 400
let nextReviewEventId = 800
let nextReviewResultId = 10000

function findWorkByID(workId: number): { trackId: number; index: number; work: Work } | null {
  for (const [trackIdText, works] of Object.entries(mockWorksByTrack)) {
    const trackId = Number(trackIdText)
    const index = works.findIndex((work) => work.workID === workId)
    if (index >= 0) {
      return { trackId, index, work: works[index] }
    }
  }
  return null
}

function ensureScriptVersionList(scriptId: number) {
  if (!mockScriptVersionsByScript[scriptId]) {
    mockScriptVersionsByScript[scriptId] = []
  }
  return mockScriptVersionsByScript[scriptId]
}

function ensureFlowStepList(flowId: number) {
  if (!mockFlowStepsByFlow[flowId]) {
    mockFlowStepsByFlow[flowId] = []
  }
  return mockFlowStepsByFlow[flowId]
}

function ensureFlowMountList(flowId: number) {
  if (!mockFlowMountsByFlow[flowId]) {
    mockFlowMountsByFlow[flowId] = []
  }
  return mockFlowMountsByFlow[flowId]
}

function toJsonObject(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}

function findContestIdByTrackId(trackId: number) {
  for (const [contestIdText, tracks] of Object.entries(mockTracksByContest)) {
    const matched = tracks.find((track) => track.trackID === trackId)
    if (matched) {
      return Number(contestIdText)
    }
  }
  return null
}

function getContestTrackIDs(contestId: number) {
  return (mockTracksByContest[contestId] || [])
    .map((track) => track.trackID)
    .filter((value): value is number => typeof value === 'number')
}

function getWorksByContestId(contestId: number) {
  return getContestTrackIDs(contestId).flatMap((trackId) => mockWorksByTrack[trackId] || [])
}

function listReviewEventsByTrackId(trackId: number) {
  return mockReviewEvents.filter((event) => event.trackID === trackId)
}

function listReviewEventsByContestId(contestId: number) {
  const trackIDs = new Set(getContestTrackIDs(contestId))
  return mockReviewEvents.filter((event) => trackIDs.has(event.trackID))
}

function listJudgeIDsByEventId(eventId: number) {
  return mockReviewEvents.find((event) => event.eventID === eventId)?.judgeIDs || []
}

function getEventFilteredWorks(event: ReviewEvent) {
  const works = mockWorksByTrack[event.trackID] || []
  const status = (event.workStatus || '').trim()
  if (!status) {
    return works
  }
  return works.filter((work) => (work.workStatus || '').trim() === status)
}

function hasJudgeReviewedWorkInOtherEvents(judgeID: number, workID: number, eventId: number) {
  const results = ensureReviewResultList(workID)
  for (const result of results) {
    if (result.reviewEventID === eventId) {
      continue
    }
    const judgeScores = result.reviews?.judgeScores
    if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores) && `${judgeID}` in judgeScores) {
      return true
    }
  }
  return false
}

function getAssignableJudgeIDsForWorkInEvent(eventId: number, workId: number) {
  const event = mockReviewEvents.find((item) => item.eventID === eventId)
  const found = findWorkByID(workId)
  if (!event || !found) {
    return []
  }
  if (found.trackId !== event.trackID) {
    return []
  }
  const status = (event.workStatus || '').trim()
  if (status && (found.work.workStatus || '').trim() !== status) {
    return []
  }

  const judgeIDs = event.judgeIDs || []
  return judgeIDs.filter((judgeID) => !hasJudgeReviewedWorkInOtherEvents(judgeID, workId, eventId))
}

function countAssignableWorksForJudgeInEvent(event: ReviewEvent, judgeID: number) {
  const works = getEventFilteredWorks(event)
  return works.filter((work) => !hasJudgeReviewedWorkInOtherEvents(judgeID, work.workID as number, event.eventID)).length
}

function ensureReviewResultList(workId: number) {
  if (!mockReviewResultsByWork[workId]) {
    mockReviewResultsByWork[workId] = []
  }
  return mockReviewResultsByWork[workId]
}

function toNumber(value: unknown) {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : 0
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}

function buildTrackRanking(trackId: number) {
  const works = mockWorksByTrack[trackId] || []
  return works
    .map((work) => {
      const results = ensureReviewResultList(work.workID as number)
      const latest = results[results.length - 1]
      const finalScore = latest ? toNumber(latest.reviews?.finalScore) : 0
      const reviewCount = latest ? Math.round(toNumber(latest.reviews?.reviewCount)) : 0

      return {
        workID: work.workID || 0,
        workTitle: work.workTitle || '',
        authorID: work.authorID || 0,
        authorName: work.authorName || '',
        finalScore,
        reviewCount,
      }
    })
    .sort((a, b) => {
      if (b.finalScore === a.finalScore) {
        return a.workID - b.workID
      }
      return b.finalScore - a.finalScore
    })
}

function buildWorkReviewStatus(workId: number) {
  const found = findWorkByID(workId)
  if (!found) {
    return null
  }

  const events = listReviewEventsByTrackId(found.trackId)
  const eventItems = events
    .filter((event) => {
      const status = (event.workStatus || '').trim()
      if (!status) {
        return true
      }
      return (found.work.workStatus || '').trim() === status
    })
    .map((event) => {
    const assignableJudgeIDs = getAssignableJudgeIDsForWorkInEvent(event.eventID, workId)
    const results = ensureReviewResultList(workId)
    const relatedResult = results.find((item) => item.reviewEventID === event.eventID)
    const judgeScores = relatedResult?.reviews?.judgeScores
    let submittedReviews = 0
    if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores)) {
      for (const judgeID of assignableJudgeIDs) {
        if (`${judgeID}` in judgeScores) {
          submittedReviews++
        }
      }
    }

    return {
      eventID: event.eventID,
      eventName: event.eventName,
      assignedJudges: assignableJudgeIDs.length,
      submittedReviews,
      completed: assignableJudgeIDs.length > 0 && submittedReviews >= assignableJudgeIDs.length,
    }
  })

  const completedEvents = eventItems.filter((item) => item.completed).length
  return {
    workID: workId,
    events: eventItems,
    summary: {
      eventCount: eventItems.length,
      completedEvents,
    },
  }
}

function buildDashboardOverview() {
  const trackSubmissionCount: Record<string, number> = {}
  const authorIDs = new Set<number>()
  const allJudgeIDs = new Set<number>()
  const completedJudgeIDs = new Set<number>()
  let completedReviewedWorks = 0

  for (const [contestIdText, tracks] of Object.entries(mockTracksByContest)) {
    const contestId = Number(contestIdText)
    const events = listReviewEventsByContestId(contestId)

    for (const track of tracks) {
      const trackId = track.trackID || 0
      const works = mockWorksByTrack[trackId] || []
      trackSubmissionCount[String(trackId)] = works.length
      for (const work of works) {
        if (typeof work.authorID === 'number') {
          authorIDs.add(work.authorID)
        }
      }
    }

    for (const event of events) {
      const judgeIDs = event.judgeIDs || []
      for (const judgeID of judgeIDs) {
        allJudgeIDs.add(judgeID)
      }

      const works = getEventFilteredWorks(event)
      for (const work of works) {
        const assignableJudgeIDs = getAssignableJudgeIDsForWorkInEvent(event.eventID, work.workID as number)
        const result = ensureReviewResultList(work.workID as number).find((item) => item.reviewEventID === event.eventID)
        const judgeScores = result?.reviews?.judgeScores
        let submitted = 0
        if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores)) {
          for (const judgeID of assignableJudgeIDs) {
            if (`${judgeID}` in judgeScores) {
              submitted++
            }
          }
        }
        if (assignableJudgeIDs.length > 0 && submitted >= assignableJudgeIDs.length) {
          completedReviewedWorks++
        }
      }

      for (const judgeID of judgeIDs) {
        const assignedCount = countAssignableWorksForJudgeInEvent(event, judgeID)
        let submittedCount = 0
        const assignableWorks = works.filter((work) => !hasJudgeReviewedWorkInOtherEvents(judgeID, work.workID as number, event.eventID))
        for (const work of assignableWorks) {
          const result = ensureReviewResultList(work.workID as number).find((item) => item.reviewEventID === event.eventID)
          const judgeScores = result?.reviews?.judgeScores
          if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores) && `${judgeID}` in judgeScores) {
            submittedCount++
          }
        }
        if (assignedCount > 0 && submittedCount >= assignedCount) {
          completedJudgeIDs.add(judgeID)
        }
      }
    }
  }

  return {
    trackSubmissionCount,
    participatingAuthors: authorIDs.size,
    completedJudgeTasks: completedJudgeIDs.size,
    totalTrackJudges: allJudgeIDs.size,
    completedReviewedWorks,
  }
}

function buildContestTrackStatusStats(contestId: number) {
  const tracks = mockTracksByContest[contestId] || []
  return tracks.map((track) => {
    const works = mockWorksByTrack[track.trackID || 0] || []
    const statusCounts: Record<string, number> = {}
    const authorIDs = new Set<number>()

    for (const work of works) {
      const status = (work.workStatus || '').trim() || 'unknown'
      statusCounts[status] = (statusCounts[status] || 0) + 1
      if (typeof work.authorID === 'number') {
        authorIDs.add(work.authorID)
      }
    }

    const distinctStates = Object.keys(statusCounts).sort((a, b) => a.localeCompare(b))
    return {
      trackID: track.trackID || 0,
      trackName: track.trackName || '',
      totalWorks: works.length,
      statusCounts,
      totalAuthors: authorIDs.size,
      distinctStates,
    }
  })
}

function buildContestDailyStats(contestId: number) {
  const works = getWorksByContestId(contestId)
  const daily: Record<string, number> = {}

  for (const work of works) {
    const submittedAt =
      (typeof work.workInfos?.submittedAt === 'string' && work.workInfos.submittedAt)
      || (typeof work.workInfos?.submitted_at === 'string' && work.workInfos.submitted_at)
      || ''

    if (!submittedAt) {
      continue
    }

    const day = submittedAt.slice(0, 10)
    if (!day) {
      continue
    }
    daily[day] = (daily[day] || 0) + 1
  }

  return Object.entries(daily)
    .map(([date, count]) => ({ date, count }))
    .sort((a, b) => a.date.localeCompare(b.date))
}

function buildContestJudgeProgressStats(contestId: number) {
  const events = listReviewEventsByContestId(contestId)
  const resultMap = new Map<number, { judgeName: string; assignedCount: number; submittedCount: number }>()

  for (const event of events) {
    const judgeIDs = event.judgeIDs || []
    const works = getEventFilteredWorks(event)

    for (const judgeID of judgeIDs) {
      const profile = mockJudgeProfiles.find((item) => item.judgeID === judgeID)
      const existing = resultMap.get(judgeID) || {
        judgeName: profile?.judgeName || `评委${judgeID}`,
        assignedCount: 0,
        submittedCount: 0,
      }

      existing.assignedCount += countAssignableWorksForJudgeInEvent(event, judgeID)

      let submitted = 0
      const assignableWorks = works.filter((work) => !hasJudgeReviewedWorkInOtherEvents(judgeID, work.workID as number, event.eventID))
      for (const work of assignableWorks) {
        const result = ensureReviewResultList(work.workID as number).find((item) => item.reviewEventID === event.eventID)
        const judgeScores = result?.reviews?.judgeScores
        if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores) && `${judgeID}` in judgeScores) {
          submitted++
        }
      }
      existing.submittedCount += submitted
      resultMap.set(judgeID, existing)
    }
  }

  return Array.from(resultMap.entries())
    .map(([judgeID, value]) => ({
      judgeID,
      judgeName: value.judgeName,
      assignedCount: value.assignedCount,
      submittedCount: value.submittedCount,
      completionRate: value.assignedCount > 0 ? value.submittedCount / value.assignedCount : 0,
    }))
    .sort((a, b) => {
      if (b.completionRate === a.completionRate) {
        return a.judgeID - b.judgeID
      }
      return b.completionRate - a.completionRate
    })
}

function regenerateWorkResults(workId: number): ReviewResult[] {
  const found = findWorkByID(workId)
  if (!found) {
    return []
  }

  const events = listReviewEventsByTrackId(found.trackId)
  const allResults = ensureReviewResultList(workId)
  const nextResults = events
    .filter((event) => {
      const status = (event.workStatus || '').trim()
      if (!status) {
        return true
      }
      return (found.work.workStatus || '').trim() === status
    })
    .map((event, index) => {
    const judgeIDs = getAssignableJudgeIDsForWorkInEvent(event.eventID, workId)
    const judgeScores = judgeIDs.reduce<Record<string, number>>((acc, judgeID) => {
      acc[String(judgeID)] = 80 + ((workId + judgeID + index) % 15)
      return acc
    }, {})
    const scoreValues = Object.values(judgeScores)
    const finalScore = scoreValues.length > 0
      ? scoreValues.reduce((sum, value) => sum + value, 0) / scoreValues.length
      : 0

    return {
      resultID: nextReviewResultId++,
      workID: workId,
      reviewEventID: event.eventID,
      reviews: {
        finalScore,
        reviewCount: scoreValues.length,
        assignedJudgeCount: judgeIDs.length,
        comments: '系统重算结果',
        judgeScores,
        generatedAt: new Date().toISOString(),
      },
    }
  })

  allResults.splice(0, allResults.length, ...nextResults)
  return allResults
}

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

  http.get('/api/system/contests', () => {
    return HttpResponse.json({ code: 200, msg: mockContests })
  }),

  http.post('/api/admin/admin/contest', async ({ request }) => {
    const payload = await request.json() as Record<string, unknown>
    const nextContest = { ...payload, contestID: nextContestId++ }
    mockContests.push(nextContest as never)
    return HttpResponse.json({ code: 200, msg: nextContest })
  }),

  http.put('/api/admin/admin/contest/:contestId', async ({ params, request }) => {
    const contestId = Number(params.contestId)
    const payload = await request.json() as Record<string, unknown>
    const index = mockContests.findIndex((contest) => contest.contestID === contestId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '赛事不存在' }, { status: 404 })
    }
    mockContests[index] = { ...mockContests[index], ...payload, contestID: contestId }
    return HttpResponse.json({ code: 200, msg: mockContests[index] })
  }),

  http.delete('/api/admin/admin/contest/:contestId', ({ params }) => {
    const contestId = Number(params.contestId)
    const index = mockContests.findIndex((contest) => contest.contestID === contestId)
    if (index >= 0) {
      const tracks = mockTracksByContest[contestId] || []
      for (const track of tracks) {
        if (track.trackID) {
          delete mockWorksByTrack[track.trackID]
        }
      }
      mockContests.splice(index, 1)
      delete mockTracksByContest[contestId]
    }
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/system/tracks/:contestId', ({ params }) => {
    const contestId = Number(params.contestId)
    return HttpResponse.json({ code: 200, msg: mockTracksByContest[contestId] || [] })
  }),

  http.post('/api/admin/admin/track', async ({ request }) => {
    const payload = await request.json() as Record<string, unknown>
    const contestId = Number(payload.contestID)
    const track = { ...payload, trackID: nextTrackId++ }
    if (!mockTracksByContest[contestId]) {
      mockTracksByContest[contestId] = []
    }
    mockTracksByContest[contestId].push(track as never)
    mockWorksByTrack[track.trackID as number] = []
    return HttpResponse.json({ code: 200, msg: track })
  }),

  http.put('/api/admin/admin/track/:trackId', async ({ params, request }) => {
    const trackId = Number(params.trackId)
    const payload = await request.json() as Record<string, unknown>

    for (const contestIdText of Object.keys(mockTracksByContest)) {
      const contestId = Number(contestIdText)
      const tracks = mockTracksByContest[contestId]
      const targetIndex = tracks.findIndex((track) => track.trackID === trackId)
      if (targetIndex < 0) {
        continue
      }

      const nextContestIdByPayload = Number(payload.contestID)
      const hasMovedContest = Number.isInteger(nextContestIdByPayload) && nextContestIdByPayload > 0 && nextContestIdByPayload !== contestId
      const updatedTrack = { ...tracks[targetIndex], ...payload, trackID: trackId }

      if (hasMovedContest) {
        tracks.splice(targetIndex, 1)
        if (!mockTracksByContest[nextContestIdByPayload]) {
          mockTracksByContest[nextContestIdByPayload] = []
        }
        mockTracksByContest[nextContestIdByPayload].push(updatedTrack as never)
      } else {
        tracks[targetIndex] = updatedTrack as never
      }

      return HttpResponse.json({ code: 200, msg: updatedTrack })
    }

    return HttpResponse.json({ code: 404, msg: '赛道不存在' }, { status: 404 })
  }),

  http.delete('/api/admin/admin/track/:trackId', ({ params }) => {
    const trackId = Number(params.trackId)

    for (const contestId of Object.keys(mockTracksByContest)) {
      const tracks = mockTracksByContest[Number(contestId)]
      const targetIndex = tracks.findIndex((track) => track.trackID === trackId)
      if (targetIndex >= 0) {
        tracks.splice(targetIndex, 1)
        break
      }
    }

    delete mockWorksByTrack[trackId]
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/admin/admin/scripts', () => {
    return HttpResponse.json({ code: 200, msg: mockScripts })
  }),

  http.get('/api/admin/admin/scripts/:scriptId', ({ params }) => {
    const scriptId = Number(params.scriptId)
    const script = mockScripts.find((item) => item.scriptID === scriptId)
    if (!script) {
      return HttpResponse.json({ code: 404, msg: '脚本不存在' }, { status: 404 })
    }
    return HttpResponse.json({ code: 200, msg: script })
  }),

  http.post('/api/admin/admin/scripts', async ({ request }) => {
    const payload = await request.json() as Record<string, unknown>
    const description = typeof payload.description === 'string'
      ? payload.description
      : typeof payload.scriptDescription === 'string'
        ? payload.scriptDescription
        : ''
    const meta = payload.meta && typeof payload.meta === 'object' && !Array.isArray(payload.meta)
      ? toJsonObject(payload.meta)
      : payload.extensionData && typeof payload.extensionData === 'object' && !Array.isArray(payload.extensionData)
        ? toJsonObject(payload.extensionData)
        : {}

    const nextScript = {
      ...payload,
      scriptID: nextScriptId++,
      description,
      scriptDescription: description,
      meta,
      extensionData: meta,
      isEnabled: false,
      createdAt: new Date().toISOString(),
    }
    mockScripts.push(nextScript as never)
    return HttpResponse.json({ code: 200, msg: nextScript })
  }),

  http.put('/api/admin/admin/scripts/:scriptId', async ({ params, request }) => {
    const scriptId = Number(params.scriptId)
    const payload = await request.json() as Record<string, unknown>
    const index = mockScripts.findIndex((item) => item.scriptID === scriptId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '脚本不存在' }, { status: 404 })
    }
    const description = typeof payload.description === 'string'
      ? payload.description
      : typeof payload.scriptDescription === 'string'
        ? payload.scriptDescription
        : mockScripts[index].description || mockScripts[index].scriptDescription || ''
    const meta = payload.meta && typeof payload.meta === 'object' && !Array.isArray(payload.meta)
      ? toJsonObject(payload.meta)
      : payload.extensionData && typeof payload.extensionData === 'object' && !Array.isArray(payload.extensionData)
        ? toJsonObject(payload.extensionData)
        : toJsonObject(mockScripts[index].meta || mockScripts[index].extensionData)

    mockScripts[index] = {
      ...mockScripts[index],
      ...payload,
      scriptID: scriptId,
      description,
      scriptDescription: description,
      meta,
      extensionData: meta,
      updatedAt: new Date().toISOString(),
    }
    return HttpResponse.json({ code: 200, msg: mockScripts[index] })
  }),

  http.post('/api/admin/admin/scripts/:scriptId/status', async ({ params, request }) => {
    const scriptId = Number(params.scriptId)
    const payload = await request.json() as { isEnabled?: boolean }
    const script = mockScripts.find((item) => item.scriptID === scriptId)
    if (!script) {
      return HttpResponse.json({ code: 404, msg: '脚本不存在' }, { status: 404 })
    }
    script.isEnabled = Boolean(payload.isEnabled)
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.post('/api/admin/admin/scripts/:scriptId/versions/upload', async ({ params, request }) => {
    const scriptId = Number(params.scriptId)
    const script = mockScripts.find((item) => item.scriptID === scriptId)
    if (!script) {
      return HttpResponse.json({ code: 404, msg: '脚本不存在' }, { status: 404 })
    }

    const formData = await request.formData()
    const file = formData.get('script_file')
    if (!file || typeof file !== 'object') {
      return HttpResponse.json({ code: 400, msg: 'script_file is required' }, { status: 400 })
    }

    const fileName = typeof (file as { name?: unknown }).name === 'string'
      ? (file as { name: string }).name
      : `script-${scriptId}.txt`

    const versions = ensureScriptVersionList(scriptId)
    const maxVersionNum = versions.reduce((max, item) => Math.max(max, item.versionNum || 0), 0)
    const versionNum = maxVersionNum + 1

    const nextVersion: ScriptVersion = {
      versionID: nextVersionId++,
      scriptID: scriptId,
      versionNum,
      versionName: `v${versionNum}`,
      fileName,
      relativePath: `scripts/${scriptId}/v${versionNum}/${fileName}`,
      checksum: `sha256:${scriptId}:${versionNum}`,
      createdBy: 1,
      isActive: false,
      createdAt: new Date().toISOString(),
    }

    versions.push(nextVersion)
    return HttpResponse.json({ code: 200, msg: nextVersion })
  }),

  http.get('/api/admin/admin/scripts/:scriptId/versions', ({ params }) => {
    const scriptId = Number(params.scriptId)
    const versions = ensureScriptVersionList(scriptId)
    return HttpResponse.json({ code: 200, msg: versions })
  }),

  http.post('/api/admin/admin/scripts/:scriptId/versions/:versionId/activate', ({ params }) => {
    const scriptId = Number(params.scriptId)
    const versionId = Number(params.versionId)

    const script = mockScripts.find((item) => item.scriptID === scriptId)
    if (!script) {
      return HttpResponse.json({ code: 404, msg: '脚本不存在' }, { status: 404 })
    }

    const versions = ensureScriptVersionList(scriptId)
    const targetVersion = versions.find((item) => item.versionID === versionId)
    if (!targetVersion) {
      return HttpResponse.json({ code: 404, msg: '脚本版本不存在' }, { status: 404 })
    }

    for (const version of versions) {
      version.isActive = version.versionID === versionId
    }
    script.activeVersionID = versionId
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/admin/admin/script-flows', () => {
    return HttpResponse.json({ code: 200, msg: mockFlows })
  }),

  http.get('/api/admin/admin/script-flows/:flowId', ({ params }) => {
    const flowId = Number(params.flowId)
    const flow = mockFlows.find((item) => item.flowID === flowId)
    if (!flow) {
      return HttpResponse.json({ code: 404, msg: '流程不存在' }, { status: 404 })
    }
    return HttpResponse.json({ code: 200, msg: flow })
  }),

  http.post('/api/admin/admin/script-flows', async ({ request }) => {
    const payload = await request.json() as Record<string, unknown>
    const description = typeof payload.description === 'string'
      ? payload.description
      : typeof payload.flowDescription === 'string'
        ? payload.flowDescription
        : ''
    const meta = payload.meta && typeof payload.meta === 'object' && !Array.isArray(payload.meta)
      ? toJsonObject(payload.meta)
      : payload.extensionData && typeof payload.extensionData === 'object' && !Array.isArray(payload.extensionData)
        ? toJsonObject(payload.extensionData)
        : {}

    const nextFlow = {
      ...payload,
      flowID: nextFlowId++,
      description,
      flowDescription: description,
      meta,
      extensionData: meta,
      isEnabled: false,
      createdAt: new Date().toISOString(),
    }
    mockFlows.push(nextFlow as never)
    return HttpResponse.json({ code: 200, msg: nextFlow })
  }),

  http.put('/api/admin/admin/script-flows/:flowId', async ({ params, request }) => {
    const flowId = Number(params.flowId)
    const payload = await request.json() as Record<string, unknown>
    const index = mockFlows.findIndex((item) => item.flowID === flowId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '流程不存在' }, { status: 404 })
    }

    const description = typeof payload.description === 'string'
      ? payload.description
      : typeof payload.flowDescription === 'string'
        ? payload.flowDescription
        : mockFlows[index].description || mockFlows[index].flowDescription || ''
    const meta = payload.meta && typeof payload.meta === 'object' && !Array.isArray(payload.meta)
      ? toJsonObject(payload.meta)
      : payload.extensionData && typeof payload.extensionData === 'object' && !Array.isArray(payload.extensionData)
        ? toJsonObject(payload.extensionData)
        : toJsonObject(mockFlows[index].meta || mockFlows[index].extensionData)

    mockFlows[index] = {
      ...mockFlows[index],
      ...payload,
      flowID: flowId,
      description,
      flowDescription: description,
      meta,
      extensionData: meta,
      updatedAt: new Date().toISOString(),
    }

    return HttpResponse.json({ code: 200, msg: mockFlows[index] })
  }),

  http.post('/api/admin/admin/script-flows/:flowId/status', async ({ params, request }) => {
    const flowId = Number(params.flowId)
    const payload = await request.json() as { isEnabled?: boolean }
    const flow = mockFlows.find((item) => item.flowID === flowId)
    if (!flow) {
      return HttpResponse.json({ code: 404, msg: '流程不存在' }, { status: 404 })
    }

    flow.isEnabled = Boolean(payload.isEnabled)
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.put('/api/admin/admin/script-flows/:flowId/steps', async ({ params, request }) => {
    const flowId = Number(params.flowId)
    const payload = await request.json() as FlowStep[]
    if (!Array.isArray(payload)) {
      return HttpResponse.json({ code: 400, msg: 'bad request' }, { status: 400 })
    }

    const normalized = payload.map((item) => ({
      ...item,
      flowID: flowId,
      stepID: item.stepID || nextStepId++,
      scriptVersionID: item.scriptVersionID,
      isEnabled: item.isEnabled ?? true,
      failureStrategy: item.failureStrategy || 'CONTINUE',
      inputTemplate: item.inputTemplate || item.stepConfig || {},
      timeoutMs: item.timeoutMs || 5000,
      stepConfig: item.inputTemplate || item.stepConfig || {},
    }))

    mockFlowStepsByFlow[flowId] = normalized
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/admin/admin/script-flows/:flowId/steps', ({ params }) => {
    const flowId = Number(params.flowId)
    return HttpResponse.json({ code: 200, msg: ensureFlowStepList(flowId) })
  }),

  http.post('/api/admin/admin/script-flows/mounts', async ({ request }) => {
    const payload = await request.json() as FlowMount
    if (!payload.flowID) {
      return HttpResponse.json({ code: 400, msg: 'flowID is required' }, { status: 400 })
    }

    const flow = mockFlows.find((item) => item.flowID === payload.flowID)
    if (!flow) {
      return HttpResponse.json({ code: 404, msg: '流程不存在' }, { status: 404 })
    }

    const scope = payload.scope || payload.targetType || payload.containerType || 'global'
    if (scope !== 'global' && scope !== 'contest' && scope !== 'track') {
      return HttpResponse.json({ code: 400, msg: 'scope must be global/contest/track' }, { status: 400 })
    }

    const targetType = payload.targetType || payload.containerType || scope
    const targetID = scope === 'global' ? 0 : payload.targetID ?? payload.containerID
    if (scope !== 'global' && (!Number.isInteger(targetID) || Number(targetID) <= 0)) {
      return HttpResponse.json({ code: 400, msg: 'targetID must be positive integer' }, { status: 400 })
    }

    const eventKey = payload.eventKey || (flow.meta && typeof flow.meta.trigger === 'string' ? flow.meta.trigger : 'work_created')

    const nextMount: FlowMount = {
      ...payload,
      mountID: nextMountId++,
      scope,
      targetType,
      targetID: Number(targetID),
      eventKey,
      isEnabled: payload.isEnabled ?? true,
      containerType: targetType,
      containerID: Number(targetID),
      createdAt: new Date().toISOString(),
    }
    ensureFlowMountList(payload.flowID).push(nextMount)
    return HttpResponse.json({ code: 200, msg: nextMount })
  }),

  http.delete('/api/admin/admin/script-flows/mounts/:mountId', ({ params }) => {
    const mountId = Number(params.mountId)

    for (const flowIdText of Object.keys(mockFlowMountsByFlow)) {
      const mounts = mockFlowMountsByFlow[Number(flowIdText)]
      const index = mounts.findIndex((item) => item.mountID === mountId)
      if (index >= 0) {
        mounts.splice(index, 1)
        return HttpResponse.json({ code: 200, msg: null })
      }
    }

    return HttpResponse.json({ code: 404, msg: '挂载不存在' }, { status: 404 })
  }),

  http.get('/api/admin/admin/script-flows/:flowId/mounts', ({ params }) => {
    const flowId = Number(params.flowId)
    return HttpResponse.json({ code: 200, msg: ensureFlowMountList(flowId) })
  }),

  http.get('/api/admin/admin/sub-admins', () => {
    return HttpResponse.json({ code: 200, msg: mockSubAdmins })
  }),

  http.post('/api/admin/admin/sub-admins', async ({ request }) => {
    const payload = await request.json() as {
      adminName?: string
      adminEmail?: string
      permissionNames?: string[]
    }

    if (!payload.adminName || !payload.adminEmail) {
      return HttpResponse.json({ code: 400, msg: 'adminName and adminEmail are required' }, { status: 400 })
    }

    const nextAdmin = {
      adminID: nextSubAdminId++,
      adminName: payload.adminName,
      adminEmail: payload.adminEmail,
      isActive: true,
      permissionNames: payload.permissionNames || [],
    }
    mockSubAdmins.push(nextAdmin)

    return HttpResponse.json({
      code: 200,
      msg: {
        adminID: nextAdmin.adminID,
        adminName: nextAdmin.adminName,
        adminEmail: nextAdmin.adminEmail,
        tempPassword: 'Temp@12345',
        emailSent: true,
      },
    })
  }),

  http.put('/api/admin/admin/sub-admins/:adminId/permissions', async ({ params, request }) => {
    const adminId = Number(params.adminId)
    const payload = await request.json() as { permissionNames?: string[] }
    const found = mockSubAdmins.find((item) => item.adminID === adminId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '子管理员不存在' }, { status: 404 })
    }
    found.permissionNames = payload.permissionNames || []
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.post('/api/admin/admin/sub-admins/:adminId/disable', ({ params }) => {
    const adminId = Number(params.adminId)
    const found = mockSubAdmins.find((item) => item.adminID === adminId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '子管理员不存在' }, { status: 404 })
    }
    found.isActive = false
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.delete('/api/admin/admin/sub-admins/:adminId', ({ params }) => {
    const adminId = Number(params.adminId)
    const index = mockSubAdmins.findIndex((item) => item.adminID === adminId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '子管理员不存在' }, { status: 404 })
    }
    mockSubAdmins.splice(index, 1)
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/admin/admin/authors', ({ request }) => {
    const url = new URL(request.url)
    const authorName = (url.searchParams.get('author_name') || '').trim().toLowerCase()
    const offset = Number(url.searchParams.get('offset') || '0')
    const limit = Number(url.searchParams.get('limit') || '20')

    let authors = [...mockAuthors]
    if (authorName) {
      authors = authors.filter((item) => {
        const byAuthorName = (item.authorName || '').toLowerCase().includes(authorName)
        const byPenName = (item.penName || '').toLowerCase().includes(authorName)
        return byAuthorName || byPenName
      })
    }

    const safeOffset = Number.isInteger(offset) && offset >= 0 ? offset : 0
    const safeLimit = Number.isInteger(limit) && limit > 0 ? Math.min(limit, 100) : 20
    const paged = authors.slice(safeOffset, safeOffset + safeLimit)
    return HttpResponse.json({ code: 200, msg: paged })
  }),

  http.post('/api/admin/admin/judge/account', async ({ request }) => {
    const payload = await request.json() as { judgeName?: string; password?: string }
    const judgeName = (payload.judgeName || '').trim()
    if (!judgeName || !(payload.password || '').trim()) {
      return HttpResponse.json({ code: 400, msg: 'judgeName and password are required' }, { status: 400 })
    }

    const nextJudge = {
      judgeID: nextJudgeId++,
      judgeName,
      judgeEmail: `${judgeName.toLowerCase()}@ubik.com`,
      isActive: true,
    }
    mockJudgeProfiles.push(nextJudge)
    return HttpResponse.json({ code: 200, msg: nextJudge })
  }),

  http.post('/api/admin/admin/judge/accounts', async ({ request }) => {
    const payload = await request.json() as { judges?: Array<{ judgeName?: string; password?: string }> }
    const judges = Array.isArray(payload.judges) ? payload.judges : []
    const created = judges
      .filter((item) => (item.judgeName || '').trim() && (item.password || '').trim())
      .map((item) => {
        const judgeName = (item.judgeName || '').trim()
        const nextJudge = {
          judgeID: nextJudgeId++,
          judgeName,
          judgeEmail: `${judgeName.toLowerCase()}@ubik.com`,
          isActive: true,
        }
        mockJudgeProfiles.push(nextJudge)
        return nextJudge
      })

    return HttpResponse.json({ code: 200, msg: created })
  }),

  http.put('/api/admin/admin/judge/:judgeId', async ({ params, request }) => {
    const judgeId = Number(params.judgeId)
    const payload = await request.json() as { judgeName?: string }
    const target = mockJudgeProfiles.find((item) => item.judgeID === judgeId)
    if (!target) {
      return HttpResponse.json({ code: 404, msg: '评委不存在' }, { status: 404 })
    }

    const judgeName = (payload.judgeName || '').trim()
    if (judgeName) {
      target.judgeName = judgeName
    }
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.delete('/api/admin/admin/judge/:judgeId', ({ params }) => {
    const judgeId = Number(params.judgeId)
    const index = mockJudgeProfiles.findIndex((item) => item.judgeID === judgeId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '评委不存在' }, { status: 404 })
    }

    mockJudgeProfiles.splice(index, 1)
    for (const event of mockReviewEvents) {
      event.judgeIDs = (event.judgeIDs || []).filter((id) => id !== judgeId)
    }
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.post('/api/admin/admin/judge/review/event', async ({ request }) => {
    const payload = await request.json() as {
      trackID?: number
      eventName?: string
      workStatus?: string
      startTime?: string
      endTime?: string
    }

    const trackID = Number(payload.trackID)
    const eventName = (payload.eventName || '').trim()
    if (!Number.isInteger(trackID) || trackID <= 0 || !eventName) {
      return HttpResponse.json({ code: 400, msg: 'bad request' }, { status: 400 })
    }

    const nextEvent: ReviewEvent = {
      eventID: nextReviewEventId++,
      trackID,
      eventName,
      workStatus: (payload.workStatus || '').trim(),
      startTime: payload.startTime || new Date().toISOString(),
      endTime: payload.endTime || new Date(Date.now() + 7 * 24 * 3600 * 1000).toISOString(),
      judgeIDs: [],
    }
    mockReviewEvents.push(nextEvent)
    return HttpResponse.json({ code: 200, msg: nextEvent })
  }),

  http.put('/api/admin/admin/judge/review/:eventId', async ({ params, request }) => {
    const eventId = Number(params.eventId)
    const payload = await request.json() as {
      trackID?: number
      eventName?: string
      workStatus?: string
      startTime?: string
      endTime?: string
    }
    const event = mockReviewEvents.find((item) => item.eventID === eventId)
    if (!event) {
      return HttpResponse.json({ code: 404, msg: '评审事件不存在' }, { status: 404 })
    }

    if (Number.isInteger(payload.trackID) && Number(payload.trackID) > 0) {
      event.trackID = Number(payload.trackID)
    }
    if ((payload.eventName || '').trim()) {
      event.eventName = (payload.eventName || '').trim()
    }
    if (typeof payload.workStatus === 'string') {
      event.workStatus = payload.workStatus.trim()
    }
    if (typeof payload.startTime === 'string' && payload.startTime.trim()) {
      event.startTime = payload.startTime
    }
    if (typeof payload.endTime === 'string' && payload.endTime.trim()) {
      event.endTime = payload.endTime
    }

    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.put('/api/admin/admin/judge/review/:eventId/assign', async ({ params, request }) => {
    const eventId = Number(params.eventId)
    const payload = await request.json() as { judgeIDs?: number[] }
    const event = mockReviewEvents.find((item) => item.eventID === eventId)
    if (!event) {
      return HttpResponse.json({ code: 404, msg: '评审事件不存在' }, { status: 404 })
    }

    const judgeIDs = Array.isArray(payload.judgeIDs)
      ? Array.from(new Set(payload.judgeIDs.filter((id) => Number.isInteger(id) && id > 0)))
      : []

    for (const judgeID of judgeIDs) {
      if (countAssignableWorksForJudgeInEvent(event, judgeID) <= 0) {
        return HttpResponse.json({ code: 400, msg: `judge ${judgeID} has no assignable works in this event` }, { status: 400 })
      }
    }

    event.judgeIDs = judgeIDs
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.delete('/api/admin/admin/judge/review/:eventId', ({ params }) => {
    const eventId = Number(params.eventId)
    const index = mockReviewEvents.findIndex((item) => item.eventID === eventId)
    if (index < 0) {
      return HttpResponse.json({ code: 404, msg: '评审事件不存在' }, { status: 404 })
    }

    mockReviewEvents.splice(index, 1)
    for (const resultList of Object.values(mockReviewResultsByWork)) {
      for (let i = resultList.length - 1; i >= 0; i--) {
        if (resultList[i].reviewEventID === eventId) {
          resultList.splice(i, 1)
        }
      }
    }
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/admin/admin/judge/review/track/:trackId/status', ({ params }) => {
    const trackId = Number(params.trackId)
    if (!Number.isInteger(trackId) || trackId <= 0) {
      return HttpResponse.json({ code: 400, msg: 'invalid track_id' }, { status: 400 })
    }

    const statuses = Array.from(new Set((mockWorksByTrack[trackId] || [])
      .map((work) => (work.workStatus || '').trim())
      .filter(Boolean)))
    return HttpResponse.json({ code: 200, msg: statuses })
  }),

  http.get('/api/admin/admin/judge/review/status/:workId', ({ params }) => {
    const workId = Number(params.workId)
    const status = buildWorkReviewStatus(workId)
    if (!status) {
      return HttpResponse.json({ code: 404, msg: '作品不存在' }, { status: 404 })
    }
    return HttpResponse.json({ code: 200, msg: status })
  }),

  http.get('/api/admin/admin/judge/review/result/:workId', ({ params }) => {
    const workId = Number(params.workId)
    const found = findWorkByID(workId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '作品不存在' }, { status: 404 })
    }
    return HttpResponse.json({ code: 200, msg: ensureReviewResultList(workId) })
  }),

  http.post('/api/admin/admin/judge/review/result/:workId/gen', ({ params }) => {
    const workId = Number(params.workId)
    const found = findWorkByID(workId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '作品不存在' }, { status: 404 })
    }
    return HttpResponse.json({ code: 200, msg: regenerateWorkResults(workId) })
  }),

  http.get('/api/admin/admin/judge/review/rank/:trackId', ({ params }) => {
    const trackId = Number(params.trackId)
    if (!Number.isInteger(trackId) || trackId <= 0) {
      return HttpResponse.json({ code: 400, msg: 'invalid track_id' }, { status: 400 })
    }
    return HttpResponse.json({ code: 200, msg: buildTrackRanking(trackId) })
  }),

  http.get('/api/admin/admin/judge/review/:eventId', ({ params }) => {
    const eventId = Number(params.eventId)
    const event = mockReviewEvents.find((item) => item.eventID === eventId)
    if (!event) {
      return HttpResponse.json({ code: 404, msg: '评审事件不存在' }, { status: 404 })
    }

    const judgeIDs = listJudgeIDsByEventId(event.eventID)
    const works = getEventFilteredWorks(event)
    const judgeProgress = judgeIDs.map((judgeID) => {
      const profile = mockJudgeProfiles.find((item) => item.judgeID === judgeID)
      const assignedCount = countAssignableWorksForJudgeInEvent(event, judgeID)
      let submittedCount = 0

      const assignableWorks = works.filter((work) => !hasJudgeReviewedWorkInOtherEvents(judgeID, work.workID as number, event.eventID))

      for (const work of assignableWorks) {
        const result = ensureReviewResultList(work.workID as number).find((item) => item.reviewEventID === event.eventID)
        const judgeScores = result?.reviews?.judgeScores
        if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores) && `${judgeID}` in judgeScores) {
          submittedCount++
        }
      }

      return {
        judgeID,
        judgeName: profile?.judgeName || `评委${judgeID}`,
        assignedCount,
        submittedCount,
        completionRate: assignedCount > 0 ? submittedCount / assignedCount : 0,
      }
    })

    const completedWorks = works.filter((work) => {
      const assignableJudgeIDs = getAssignableJudgeIDsForWorkInEvent(event.eventID, work.workID as number)
      const result = ensureReviewResultList(work.workID as number).find((item) => item.reviewEventID === event.eventID)
      const judgeScores = result?.reviews?.judgeScores
      let submittedCount = 0
      if (judgeScores && typeof judgeScores === 'object' && !Array.isArray(judgeScores)) {
        for (const judgeID of assignableJudgeIDs) {
          if (`${judgeID}` in judgeScores) {
            submittedCount++
          }
        }
      }
      return assignableJudgeIDs.length > 0 && submittedCount >= assignableJudgeIDs.length
    }).length

    return HttpResponse.json({
      code: 200,
      msg: {
        eventID: event.eventID,
        eventName: event.eventName,
        trackID: event.trackID,
        assignedJudgeIDs: judgeIDs,
        totalWorks: works.length,
        completedWorks,
        judgeProgress,
      },
    })
  }),

  http.get('/api/admin/admin/judge/review/export/:trackId', ({ params }) => {
    const trackId = Number(params.trackId)
    const foundTrack = Object.values(mockTracksByContest)
      .flat()
      .find((track) => track.trackID === trackId)
    if (!foundTrack) {
      return HttpResponse.json({ code: 404, msg: '赛道不存在' }, { status: 404 })
    }

    const ranking = buildTrackRanking(trackId)
    const header = 'workID,workTitle,authorName,finalScore,reviewCount\n'
    const content = ranking
      .map((item) => `${item.workID},${item.workTitle},${item.authorName},${item.finalScore.toFixed(2)},${item.reviewCount}`)
      .join('\n')
    const bytes = new TextEncoder().encode(header + content)

    return HttpResponse.arrayBuffer(bytes.buffer, {
      status: 200,
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': `attachment; filename="track-${trackId}-review.xlsx"`,
      },
    })
  }),

  http.get('/api/admin/admin/dashboard/overview', () => {
    return HttpResponse.json({ code: 200, msg: buildDashboardOverview() })
  }),

  http.get('/api/admin/admin/contests/:contestId/stats/tracks-status', ({ params }) => {
    const contestId = Number(params.contestId)
    if (!Number.isInteger(contestId) || contestId <= 0) {
      return HttpResponse.json({ code: 400, msg: 'invalid contest_id' }, { status: 400 })
    }
    return HttpResponse.json({ code: 200, msg: buildContestTrackStatusStats(contestId) })
  }),

  http.get('/api/admin/admin/contests/:contestId/stats/daily-submissions', ({ params }) => {
    const contestId = Number(params.contestId)
    if (!Number.isInteger(contestId) || contestId <= 0) {
      return HttpResponse.json({ code: 400, msg: 'invalid contest_id' }, { status: 400 })
    }
    return HttpResponse.json({ code: 200, msg: buildContestDailyStats(contestId) })
  }),

  http.get('/api/admin/admin/contests/:contestId/stats/judges-progress', ({ params }) => {
    const contestId = Number(params.contestId)
    if (!Number.isInteger(contestId) || contestId <= 0) {
      return HttpResponse.json({ code: 400, msg: 'invalid contest_id' }, { status: 400 })
    }
    return HttpResponse.json({ code: 200, msg: buildContestJudgeProgressStats(contestId) })
  }),

  http.post('/api/admin/admin/review-results/generate/:contestId', ({ params }) => {
    const contestId = Number(params.contestId)
    if (!Number.isInteger(contestId) || contestId <= 0) {
      return HttpResponse.json({ code: 400, msg: 'invalid contest_id' }, { status: 400 })
    }

    const events = listReviewEventsByContestId(contestId)
    let generated = 0
    for (const event of events) {
      const works = getEventFilteredWorks(event)
      for (const work of works) {
        regenerateWorkResults(work.workID as number)
        generated++
      }
    }
    return HttpResponse.json({ code: 200, msg: { generated } })
  }),

  http.post('/api/admin/admin/review-events/:eventId/judges/:judgeId/deadline', ({ params }) => {
    const eventId = Number(params.eventId)
    const judgeId = Number(params.judgeId)
    const event = mockReviewEvents.find((item) => item.eventID === eventId)
    if (!event || !(event.judgeIDs || []).includes(judgeId)) {
      return HttpResponse.json({ code: 404, msg: '评审事件或评委不存在' }, { status: 404 })
    }
    return HttpResponse.json({ code: 200, msg: null })
  }),

  http.get('/api/admin/admin/works', ({ request }) => {
    const url = new URL(request.url)
    const trackId = Number(url.searchParams.get('track_id') || '')
    const status = (url.searchParams.get('status') || '').trim().toLowerCase()
    const workTitle = (url.searchParams.get('work_title') || '').trim().toLowerCase()
    const authorName = (url.searchParams.get('author_name') || '').trim().toLowerCase()
    const offset = Number(url.searchParams.get('offset') || '0')
    const limit = Number(url.searchParams.get('limit') || '20')

    let works = getAllMockWorks()
    if (Number.isInteger(trackId) && trackId > 0) {
      works = works.filter((item) => item.trackID === trackId)
    }
    if (status) {
      works = works.filter((item) => (item.workStatus || '').trim().toLowerCase() === status)
    }
    if (workTitle) {
      works = works.filter((item) => (item.workTitle || '').toLowerCase().includes(workTitle))
    }
    if (authorName) {
      works = works.filter((item) => (item.authorName || '').toLowerCase().includes(authorName))
    }

    const safeOffset = Number.isInteger(offset) && offset > 0 ? offset : 0
    const safeLimit = Number.isInteger(limit) && limit > 0 ? Math.min(limit, 100) : 20
    const paged = works.slice(safeOffset, safeOffset + safeLimit)
    return HttpResponse.json({ code: 200, msg: paged })
  }),

  http.get('/api/admin/admin/works/:workId/file', ({ params }) => {
    const workId = Number(params.workId)
    const found = findWorkByID(workId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '作品不存在' }, { status: 404 })
    }

    const content = new TextEncoder().encode(`mock file for work ${workId}`)
    return HttpResponse.arrayBuffer(content.buffer, {
      status: 200,
      headers: {
        'Content-Type': 'application/octet-stream',
      },
    })
  }),

  http.get('/api/admin/admin/works/:workId', ({ params }) => {
    const workId = Number(params.workId)
    const found = findWorkByID(workId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '作品不存在' }, { status: 404 })
    }

    return HttpResponse.json({ code: 200, msg: found.work })
  }),

  http.delete('/api/admin/admin/works/:workId', ({ params }) => {
    const workId = Number(params.workId)
    const found = findWorkByID(workId)
    if (!found) {
      return HttpResponse.json({ code: 404, msg: '作品不存在' }, { status: 404 })
    }

    mockWorksByTrack[found.trackId].splice(found.index, 1)
    return HttpResponse.json({ code: 200, msg: null })
  }),
]
