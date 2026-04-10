import { describe, expect, it } from 'vitest'
import { login } from '@/services/repositories/authRepository'
import { fetchContests } from '@/services/repositories/contestRepository'
import {
  createFlowMount,
  createScriptFlow,
  fetchFlowMounts,
  fetchFlowSteps,
  fetchScriptFlows,
  removeFlowMount,
  replaceFlowSteps,
  updateScriptFlowStatus,
} from '@/services/repositories/scriptFlowRepository'
import {
  activateScriptVersion,
  createScriptDefinition,
  fetchScriptDefinitions,
  fetchScriptVersions,
  updateScriptDefinitionStatus,
  uploadScriptVersion,
} from '@/services/repositories/scriptRepository'
import { fetchTracks } from '@/services/repositories/trackRepository'
import {
  fetchWorkByID,
  fetchWorks,
  removeWork,
} from '@/services/repositories/workRepository'
import {
  createSubAdmin,
  disableSubAdmin,
  fetchSubAdmins,
  removeSubAdmin,
  updateSubAdminPermissions,
} from '@/services/repositories/subAdminRepository'
import { fetchAuthors } from '@/services/repositories/authorRepository'

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

  it('supports scripts and flows repositories', async () => {
    const scripts = await fetchScriptDefinitions()
    expect(scripts.length).toBeGreaterThan(0)

    const createdScript = await createScriptDefinition({
      scriptName: '测试脚本',
      scriptKey: 'integration-script',
      description: 'integration test',
      interpreter: 'python3',
      meta: { language: 'python' },
    })
    expect(createdScript.scriptID).toBeTruthy()
    expect(createdScript.description || createdScript.scriptDescription).toContain('integration')

    const scriptId = createdScript.scriptID as number
    await updateScriptDefinitionStatus(scriptId, { isEnabled: true })

    const uploadedVersion = await uploadScriptVersion(
      scriptId,
      new File(['print(1)'], 'test_script.py', { type: 'text/plain' }),
    )
    expect(uploadedVersion.versionID).toBeTruthy()

    const versionList = await fetchScriptVersions(scriptId)
    expect(versionList.length).toBeGreaterThan(0)

    const versionId = uploadedVersion.versionID as number
    await activateScriptVersion(scriptId, versionId)

    const flows = await fetchScriptFlows()
    expect(Array.isArray(flows)).toBe(true)

    const createdFlow = await createScriptFlow({
      flowName: '测试流程',
      flowKey: 'integration-flow',
      description: 'integration flow',
      meta: { trigger: 'work_created' },
    })
    expect(createdFlow.flowID).toBeTruthy()

    const flowId = createdFlow.flowID as number
    await updateScriptFlowStatus(flowId, { isEnabled: true })

    await replaceFlowSteps(flowId, [
      {
        stepOrder: 1,
        stepName: 'step1',
        scriptID: scriptId,
        scriptVersionID: versionId,
        failureStrategy: 'CONTINUE',
        timeoutMs: 5000,
        inputTemplate: { retry: 0 },
        isEnabled: true,
      },
    ])

    const steps = await fetchFlowSteps(flowId)
    expect(steps.length).toBeGreaterThan(0)
    expect(steps[0].scriptVersionID).toBe(versionId)
    expect(steps[0].failureStrategy).toBe('CONTINUE')

    const mount = await createFlowMount({
      flowID: flowId,
      scope: 'track',
      targetType: 'track',
      targetID: 101,
      eventKey: 'work_created',
      isEnabled: true,
    })
    expect(mount.mountID).toBeTruthy()
    expect(mount.scope).toBe('track')

    const mounts = await fetchFlowMounts(flowId)
    expect(mounts.length).toBeGreaterThan(0)
    expect(mounts[0].eventKey).toBeTruthy()

    await removeFlowMount(mount.mountID as number)
    const mountsAfterDelete = await fetchFlowMounts(flowId)
    expect(mountsAfterDelete.find((item) => item.mountID === mount.mountID)).toBeUndefined()
  })

  it('supports works repositories', async () => {
    const worksByTrack = await fetchWorks({ trackID: 101, limit: 100 })
    expect(worksByTrack.length).toBeGreaterThan(0)

    const worksByAuthor = await fetchWorks({ authorName: '陈', limit: 100 })
    expect(worksByAuthor.length).toBeGreaterThan(0)

    const workId = worksByTrack[0].workID as number
    const detail = await fetchWorkByID(workId)
    expect(detail.workID).toBe(workId)

    await removeWork(workId)
    const worksAfterDelete = await fetchWorks({ trackID: 101, limit: 100 })
    expect(worksAfterDelete.find((item) => item.workID === workId)).toBeUndefined()
  })

  it('supports sub-admin repositories', async () => {
    const beforeList = await fetchSubAdmins()
    const created = await createSubAdmin({
      adminName: '测试子管理员',
      adminEmail: 'tester@ubik.com',
      permissionNames: ['contest.read'],
    })
    expect(created.adminID).toBeTruthy()

    await updateSubAdminPermissions(created.adminID, {
      permissionNames: ['contest.read', 'work.read'],
    })

    await disableSubAdmin(created.adminID)

    const afterList = await fetchSubAdmins()
    expect(afterList.length).toBeGreaterThanOrEqual(beforeList.length + 1)

    await removeSubAdmin(created.adminID)
    const finalList = await fetchSubAdmins()
    expect(finalList.find((item) => item.adminID === created.adminID)).toBeUndefined()
  })

  it('supports authors repository list query', async () => {
    const allAuthors = await fetchAuthors({ offset: 0, limit: 20 })
    expect(allAuthors.length).toBeGreaterThan(0)

    const filteredAuthors = await fetchAuthors({ authorName: '陈', offset: 0, limit: 20 })
    expect(filteredAuthors.length).toBeGreaterThan(0)
    expect(filteredAuthors.every((item) => (item.authorName || '').includes('陈') || (item.penName || '').includes('陈'))).toBe(true)
  })
})
