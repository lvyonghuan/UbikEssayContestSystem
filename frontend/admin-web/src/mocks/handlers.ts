import { http, HttpResponse } from 'msw'
import type { FlowMount, FlowStep, ScriptVersion, Work } from '@/types/api'
import {
  getAllMockWorks,
  mockAuthors,
  mockContests,
  mockFlowMountsByFlow,
  mockFlowStepsByFlow,
  mockFlows,
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
    const nextScript = {
      ...payload,
      scriptID: nextScriptId++,
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
    mockScripts[index] = {
      ...mockScripts[index],
      ...payload,
      scriptID: scriptId,
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

    const nextVersion: ScriptVersion = {
      versionID: nextVersionId++,
      scriptID: scriptId,
      versionName: `v${Date.now()}`,
      fileName,
      fileURL: `/mock/script-files/${scriptId}/${fileName}`,
      isActive: false,
      createdAt: new Date().toISOString(),
    }

    const versions = ensureScriptVersionList(scriptId)
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
    const nextFlow = {
      ...payload,
      flowID: nextFlowId++,
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

    mockFlows[index] = {
      ...mockFlows[index],
      ...payload,
      flowID: flowId,
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

    const nextMount: FlowMount = {
      ...payload,
      mountID: nextMountId++,
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

  http.get('/api/admin/admin/works', ({ request }) => {
    const url = new URL(request.url)
    const trackId = Number(url.searchParams.get('track_id') || '')
    const workTitle = (url.searchParams.get('work_title') || '').trim().toLowerCase()
    const authorName = (url.searchParams.get('author_name') || '').trim().toLowerCase()
    const offset = Number(url.searchParams.get('offset') || '0')
    const limit = Number(url.searchParams.get('limit') || '20')

    let works = getAllMockWorks()
    if (Number.isInteger(trackId) && trackId > 0) {
      works = works.filter((item) => item.trackID === trackId)
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
