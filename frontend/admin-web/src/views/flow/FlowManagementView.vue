<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createFlowMount,
  createScriptFlow,
  fetchFlowMounts,
  fetchFlowSteps,
  fetchScriptFlows,
  removeFlowMount,
  replaceFlowSteps,
  updateScriptFlow,
  updateScriptFlowStatus,
} from '@/services/repositories/scriptFlowRepository'
import { fetchScriptDefinitions, fetchScriptVersions } from '@/services/repositories/scriptRepository'
import type {
  FlowMount,
  FlowMountScope,
  FlowMountTargetType,
  FlowStep,
  ScriptDefinition,
  ScriptFlow,
  ScriptVersion,
} from '@/types/api'

type FailureStrategy = 'fail_close' | 'fail_open' | 'retry'
type StepVersionMode = 'active' | 'fixed'

interface StepRowForm {
  stepID?: number
  stepOrder: number
  stepName: string
  scriptID?: number
  versionMode: StepVersionMode
  scriptVersionID?: number
  isEnabled: boolean
  failureStrategy: FailureStrategy | string
  timeoutMs: number
  inputTemplateText: string
}

const flowKeyPattern = /^[a-zA-Z0-9_-]+$/

const eventKeyOptions = [
  'submission_pre',
  'submission_update_pre',
  'submission_delete_pre',
  'file_pre',
  'file_post',
  'contest_end',
]

const scopeEventMap: Record<FlowMountScope, string[]> = {
  submission: ['submission_pre', 'submission_update_pre', 'submission_delete_pre', 'file_pre', 'file_post'],
  system: ['contest_end'],
  judge: [],
}

const eventKeyLabelMap: Record<string, string> = {
  submission_pre: '投稿前',
  submission_update_pre: '投稿更新前',
  submission_delete_pre: '投稿删除前',
  file_pre: '文件上传前',
  file_post: '文件上传后',
  contest_end: '比赛结束',
}

const scopeLabelMap: Record<FlowMountScope, string> = {
  submission: '投稿域',
  system: '系统域',
  judge: '评审域',
}

const targetTypeLabelMap: Record<FlowMountTargetType, string> = {
  global: '全局',
  contest: '比赛',
  track: '赛道',
}

const loading = ref(false)
const flows = ref<ScriptFlow[]>([])
const scripts = ref<ScriptDefinition[]>([])

const flowDialogVisible = ref(false)
const editingFlowId = ref<number | null>(null)
const flowForm = reactive({
  flowName: '',
  flowKey: '',
  flowDescription: '',
  triggerEvent: 'file_post',
  metaText: '{\n  "trigger": "file_post"\n}',
})

const stepDrawerVisible = ref(false)
const stepLoading = ref(false)
const stepSaving = ref(false)
const stepEditorText = ref('[]')
const advancedJsonOpen = ref<string[]>([])
const stepRows = ref<StepRowForm[]>([])
const currentStepFlow = ref<ScriptFlow | null>(null)
const draggingStepIndex = ref<number | null>(null)
const scriptVersionsByScriptID = ref<Record<number, ScriptVersion[]>>({})
const scriptVersionLoadingByScriptID = ref<Record<number, boolean>>({})

const mountDrawerVisible = ref(false)
const mountLoading = ref(false)
const mounts = ref<FlowMount[]>([])
const currentMountFlow = ref<ScriptFlow | null>(null)
const mountForm = reactive({
  scope: 'submission' as FlowMountScope,
  targetType: 'track' as FlowMountTargetType,
  targetIDText: '',
  eventKey: 'file_post',
  isEnabled: true,
})

const failureStrategyOptions = [
  { label: '失败即阻断(推荐)', value: 'fail_close' as FailureStrategy },
  { label: '失败放行', value: 'fail_open' as FailureStrategy },
  { label: '失败重试', value: 'retry' as FailureStrategy },
]

const mountScopeOptions = [
  { label: '投稿域', value: 'submission' as FlowMountScope },
  { label: '系统域', value: 'system' as FlowMountScope },
  { label: '评审域', value: 'judge' as FlowMountScope },
]

const targetTypeOptions = [
  { label: '全局', value: 'global' as FlowMountTargetType },
  { label: '比赛', value: 'contest' as FlowMountTargetType },
  { label: '赛道', value: 'track' as FlowMountTargetType },
]

const mountEventOptions = computed(() => {
  const options = scopeEventMap[mountForm.scope] || []
  return options.length > 0 ? options : eventKeyOptions
})

const mountPreviewText = computed(() => {
  const target = mountForm.targetType === 'global'
    ? '全局'
    : `${formatTargetType(mountForm.targetType)} #${mountForm.targetIDText || '?'}`
  return `将在 ${formatScope(mountForm.scope)} 的 ${formatEventKey(mountForm.eventKey)} 事件下触发，作用范围：${target}`
})

function parseJsonObject(text: string, errorMessage: string) {
  try {
    const parsed = text.trim() ? JSON.parse(text) : {}
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error('invalid json object')
    }
    return parsed as Record<string, unknown>
  } catch {
    ElMessage.error(errorMessage)
    return null
  }
}

function normalizeFlowDescription(flow: ScriptFlow) {
  return flow.description || flow.flowDescription || ''
}

function normalizeFlowMeta(flow: ScriptFlow) {
  return flow.meta || flow.extensionData || {}
}

function detectFlowTrigger(flow: ScriptFlow | null) {
  const fallback = 'file_post'
  if (!flow) {
    return fallback
  }

  const meta = normalizeFlowMeta(flow)
  if (typeof meta.trigger === 'string' && meta.trigger.trim()) {
    const eventKey = meta.trigger.trim()
    return eventKeyOptions.includes(eventKey) ? eventKey : fallback
  }
  if (typeof meta.eventKey === 'string' && meta.eventKey.trim()) {
    const eventKey = meta.eventKey.trim()
    return eventKeyOptions.includes(eventKey) ? eventKey : fallback
  }
  return fallback
}

function normalizeFailureStrategy(value: unknown): FailureStrategy {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : ''
  if (normalized === 'fail_open' || normalized === 'retry') {
    return normalized
  }
  return 'fail_close'
}

function formatEventKey(eventKey: string | undefined) {
  if (!eventKey || !eventKey.trim()) {
    return '-'
  }
  const normalized = eventKey.trim()
  return eventKeyLabelMap[normalized] ? `${eventKeyLabelMap[normalized]} (${normalized})` : normalized
}

function formatScope(scope: string | undefined) {
  if (!scope) {
    return '-'
  }
  return scopeLabelMap[scope as FlowMountScope] || scope
}

function formatTargetType(targetType: string | undefined) {
  if (!targetType) {
    return '-'
  }
  return targetTypeLabelMap[targetType as FlowMountTargetType] || targetType
}

function formatVersionOption(version: ScriptVersion) {
  const name = version.versionName || (version.versionNum ? `v${version.versionNum}` : `#${version.versionID}`)
  const active = version.isActive ? ' [激活]' : ''
  const path = version.relativePath ? ` | ${version.relativePath}` : ''
  return `${name}${active}${path}`
}

function defaultStepRow(order: number): StepRowForm {
  return {
    stepOrder: order,
    stepName: `步骤 ${order}`,
    scriptID: scripts.value[0]?.scriptID,
    versionMode: 'active',
    scriptVersionID: undefined,
    isEnabled: true,
    failureStrategy: 'fail_close',
    timeoutMs: 5000,
    inputTemplateText: '{\n}',
  }
}

function normalizeStepRow(step: FlowStep, index: number): StepRowForm {
  const inputTemplate = step.inputTemplate || step.stepConfig || {}
  return {
    stepID: step.stepID,
    stepOrder: Number.isInteger(step.stepOrder) && step.stepOrder > 0 ? step.stepOrder : index + 1,
    stepName: step.stepName || `步骤 ${index + 1}`,
    scriptID: step.scriptID,
    versionMode: step.scriptVersionID ? 'fixed' : 'active',
    scriptVersionID: step.scriptVersionID,
    isEnabled: step.isEnabled ?? true,
    failureStrategy: normalizeFailureStrategy(step.failureStrategy),
    timeoutMs: Number.isInteger(step.timeoutMs) && (step.timeoutMs as number) > 0 ? (step.timeoutMs as number) : 5000,
    inputTemplateText: JSON.stringify(inputTemplate, null, 2),
  }
}

function mapStepRowsToPayload(rows: StepRowForm[], showWarning = true) {
  const warn = (message: string) => {
    if (showWarning) {
      ElMessage.warning(message)
    }
  }

  const nextSteps: FlowStep[] = []
  for (let index = 0; index < rows.length; index += 1) {
    const row = rows[index]
    if (!row.scriptID) {
      warn(`第 ${index + 1} 步需要选择脚本`)
      return null
    }
    if (!row.stepName.trim()) {
      warn(`第 ${index + 1} 步需要填写步骤名称`)
      return null
    }
    if (row.versionMode === 'fixed' && !row.scriptVersionID) {
      warn(`第 ${index + 1} 步已设置锁定版本，请选择脚本版本`)
      return null
    }
    if (!Number.isInteger(row.timeoutMs) || row.timeoutMs <= 0) {
      warn(`第 ${index + 1} 步超时时间需要是正整数`)
      return null
    }

    const inputTemplate = parseJsonObject(
      row.inputTemplateText,
      `第 ${index + 1} 步脚本运行参数必须是合法 JSON 对象`,
    )
    if (!inputTemplate) {
      return null
    }

    nextSteps.push({
      stepID: row.stepID,
      stepOrder: index + 1,
      stepName: row.stepName.trim(),
      scriptID: row.scriptID,
      scriptVersionID: row.versionMode === 'fixed' ? row.scriptVersionID : undefined,
      isEnabled: row.isEnabled,
      failureStrategy: normalizeFailureStrategy(row.failureStrategy),
      timeoutMs: row.timeoutMs,
      inputTemplate,
      stepConfig: inputTemplate,
    })
  }

  return nextSteps
}

function parseStepRowsFromJsonEditor() {
  let parsed: unknown
  try {
    parsed = JSON.parse(stepEditorText.value || '[]')
  } catch {
    ElMessage.error('高级 JSON 必须是合法 JSON 数组')
    return null
  }

  if (!Array.isArray(parsed)) {
    ElMessage.error('高级 JSON 必须是数组')
    return null
  }

  return parsed.map((item, index) => normalizeStepRow(item as FlowStep, index))
}

function syncJsonFromRows(rows: StepRowForm[]) {
  const payload = mapStepRowsToPayload(rows, false)
  if (!payload) {
    return false
  }

  stepEditorText.value = JSON.stringify(payload, null, 2)
  return true
}

function resolveScope(mount: FlowMount): string {
  return mount.scope || '-'
}

function resolveTargetType(mount: FlowMount) {
  return mount.targetType || mount.containerType || '-'
}

function resolveTargetID(mount: FlowMount) {
  const targetID = mount.targetID ?? mount.containerID
  if (typeof targetID === 'number') {
    return targetID
  }
  return resolveTargetType(mount) === 'global' ? 0 : null
}

function getScriptByID(scriptID: number | undefined) {
  if (!scriptID) {
    return null
  }
  return scripts.value.find((item) => item.scriptID === scriptID) || null
}

function getVersionsForScript(scriptID: number | undefined) {
  if (!scriptID) {
    return []
  }
  return scriptVersionsByScriptID.value[scriptID] || []
}

function isVersionListLoading(scriptID: number | undefined) {
  if (!scriptID) {
    return false
  }
  return Boolean(scriptVersionLoadingByScriptID.value[scriptID])
}

function getActiveVersion(scriptID: number | undefined) {
  const versions = getVersionsForScript(scriptID)
  if (versions.length === 0) {
    return null
  }
  return versions.find((item) => item.isActive) || versions[0]
}

function resolveStepVersion(row: StepRowForm) {
  if (!row.scriptID) {
    return null
  }
  if (row.versionMode === 'active') {
    return getActiveVersion(row.scriptID)
  }
  return getVersionsForScript(row.scriptID).find((item) => item.versionID === row.scriptVersionID) || null
}

function resolveStepExecutionPath(row: StepRowForm) {
  const version = resolveStepVersion(row)
  return version?.relativePath || '-'
}

function resolveStepVersionLabel(row: StepRowForm) {
  const version = resolveStepVersion(row)
  if (!version) {
    return row.versionMode === 'active' ? '跟随激活版本（未加载）' : '锁定版本（未选择）'
  }
  const versionName = version.versionName || (version.versionNum ? `v${version.versionNum}` : `#${version.versionID}`)
  return row.versionMode === 'active' ? `跟随激活版本 ${versionName}` : `锁定版本 ${versionName}`
}

function resetFlowForm() {
  Object.assign(flowForm, {
    flowName: '',
    flowKey: '',
    flowDescription: '',
    triggerEvent: 'file_post',
    metaText: '{\n  "trigger": "file_post"\n}',
  })
}

function resetMountForm(flow: ScriptFlow | null) {
  const triggerEvent = detectFlowTrigger(flow)
  Object.assign(mountForm, {
    scope: 'submission' as FlowMountScope,
    targetType: 'track' as FlowMountTargetType,
    targetIDText: '',
    eventKey: triggerEvent,
    isEnabled: true,
  })

  syncMountEventWithScope()
  syncTargetIDWithTargetType()
}

function validateFlowKey(flowKey: string, fieldLabel: string) {
  if (!flowKey.trim()) {
    ElMessage.warning(`${fieldLabel}不能为空`)
    return false
  }
  if (!flowKeyPattern.test(flowKey.trim())) {
    ElMessage.warning(`${fieldLabel}仅支持字母、数字、下划线和连字符`)
    return false
  }
  return true
}

function parsePositiveInteger(text: string) {
  const value = Number(text)
  if (!Number.isInteger(value) || value <= 0) {
    return null
  }
  return value
}

function reindexStepOrders() {
  stepRows.value.forEach((row, index) => {
    row.stepOrder = index + 1
  })
}

function syncMountEventWithScope() {
  const options = scopeEventMap[mountForm.scope] || []
  if (options.length === 0) {
    return
  }
  if (!options.includes(mountForm.eventKey)) {
    mountForm.eventKey = options[0]
  }
}

function syncTargetIDWithTargetType() {
  if (mountForm.targetType === 'global') {
    mountForm.targetIDText = '0'
    return
  }

  if (mountForm.targetIDText === '0') {
    mountForm.targetIDText = ''
  }
}

async function ensureVersionsLoaded(scriptID: number | undefined) {
  if (!scriptID) {
    return
  }
  if (scriptVersionsByScriptID.value[scriptID]) {
    return
  }

  scriptVersionLoadingByScriptID.value = {
    ...scriptVersionLoadingByScriptID.value,
    [scriptID]: true,
  }

  try {
    const versions = await fetchScriptVersions(scriptID)
    scriptVersionsByScriptID.value = {
      ...scriptVersionsByScriptID.value,
      [scriptID]: versions,
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '脚本版本加载失败')
  } finally {
    scriptVersionLoadingByScriptID.value = {
      ...scriptVersionLoadingByScriptID.value,
      [scriptID]: false,
    }
  }
}

function moveStep(fromIndex: number, toIndex: number) {
  if (
    fromIndex < 0
    || toIndex < 0
    || fromIndex >= stepRows.value.length
    || toIndex >= stepRows.value.length
    || fromIndex === toIndex
  ) {
    return
  }

  const [moved] = stepRows.value.splice(fromIndex, 1)
  stepRows.value.splice(toIndex, 0, moved)
  reindexStepOrders()
  syncJsonFromRows(stepRows.value)
}

function moveStepUp(index: number) {
  moveStep(index, index - 1)
}

function moveStepDown(index: number) {
  moveStep(index, index + 1)
}

function onStepDragStart(index: number) {
  draggingStepIndex.value = index
}

function onStepDragOver(event: DragEvent) {
  event.preventDefault()
}

function onStepDrop(index: number) {
  if (draggingStepIndex.value === null) {
    return
  }
  moveStep(draggingStepIndex.value, index)
  draggingStepIndex.value = null
}

function onStepDragEnd() {
  draggingStepIndex.value = null
}

function refreshJsonFromRows() {
  if (!syncJsonFromRows(stepRows.value)) {
    ElMessage.warning('当前卡片存在未完成字段，无法生成 JSON')
    return
  }
  ElMessage.success('已根据卡片刷新高级 JSON')
}

async function applyJsonToRows() {
  const parsedRows = parseStepRowsFromJsonEditor()
  if (!parsedRows) {
    return
  }

  const payload = mapStepRowsToPayload(parsedRows)
  if (!payload) {
    return
  }

  stepRows.value = payload.map((item, index) => normalizeStepRow(item, index))
  reindexStepOrders()

  await Promise.all(stepRows.value.map((item) => ensureVersionsLoaded(item.scriptID)))
  syncJsonFromRows(stepRows.value)
  ElMessage.success('已将高级 JSON 应用到步骤卡片')
}

async function loadFlows() {
  loading.value = true
  try {
    const [nextFlows, nextScripts] = await Promise.all([fetchScriptFlows(), fetchScriptDefinitions()])
    flows.value = nextFlows
    scripts.value = nextScripts
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程列表加载失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingFlowId.value = null
  resetFlowForm()
  flowDialogVisible.value = true
}

function openEditDialog(flow: ScriptFlow) {
  const meta = normalizeFlowMeta(flow)
  editingFlowId.value = flow.flowID || null
  Object.assign(flowForm, {
    flowName: flow.flowName,
    flowKey: flow.flowKey || '',
    flowDescription: normalizeFlowDescription(flow),
    triggerEvent: detectFlowTrigger(flow),
    metaText: JSON.stringify(meta, null, 2),
  })
  flowDialogVisible.value = true
}

async function saveFlow() {
  const flowName = flowForm.flowName.trim()
  const flowKey = flowForm.flowKey.trim()
  if (!flowName) {
    ElMessage.warning('请填写流程名称')
    return
  }
  if (!validateFlowKey(flowKey, '流程键(flowKey)')) {
    return
  }

  const meta = parseJsonObject(flowForm.metaText, '流程元数据必须是合法 JSON 对象')
  if (!meta) {
    return
  }

  const triggerEvent = flowForm.triggerEvent.trim()
  if (triggerEvent) {
    meta.trigger = triggerEvent
  }

  const payload: ScriptFlow = {
    flowName,
    flowKey,
    flowDescription: flowForm.flowDescription.trim(),
    description: flowForm.flowDescription.trim(),
    meta,
    extensionData: meta,
    isEnabled: true,
  }

  try {
    if (editingFlowId.value) {
      await updateScriptFlow(editingFlowId.value, payload)
      ElMessage.success('流程更新成功')
    } else {
      await createScriptFlow(payload)
      ElMessage.success('流程创建成功')
    }
    flowDialogVisible.value = false
    await loadFlows()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程保存失败')
  }
}

async function changeFlowStatus(flow: ScriptFlow, isEnabled: boolean) {
  if (!flow.flowID) {
    return
  }

  try {
    await updateScriptFlowStatus(flow.flowID, { isEnabled })
    flow.isEnabled = isEnabled
    ElMessage.success('流程状态已更新')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程状态更新失败')
  }
}

function handleFlowStatusChange(flow: ScriptFlow, value: unknown) {
  void changeFlowStatus(flow, Boolean(value))
}

async function openStepDrawer(flow: ScriptFlow) {
  if (!flow.flowID) {
    ElMessage.warning('流程 ID 不存在，无法编辑步骤')
    return
  }

  currentStepFlow.value = flow
  stepRows.value = [defaultStepRow(1)]
  stepEditorText.value = '[]'
  advancedJsonOpen.value = []
  stepDrawerVisible.value = true
  stepLoading.value = true

  try {
    const nextSteps = await fetchFlowSteps(flow.flowID)
    const normalizedRows = nextSteps.length > 0
      ? nextSteps.map((item, index) => normalizeStepRow(item, index))
      : [defaultStepRow(1)]

    stepRows.value = normalizedRows
    reindexStepOrders()
    await Promise.all(stepRows.value.map((item) => ensureVersionsLoaded(item.scriptID)))
    syncJsonFromRows(stepRows.value)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程步骤加载失败')
  } finally {
    stepLoading.value = false
  }
}

async function onStepScriptChange(row: StepRowForm) {
  row.versionMode = 'active'
  row.scriptVersionID = undefined
  await ensureVersionsLoaded(row.scriptID)
  syncJsonFromRows(stepRows.value)
}

function onStepVersionModeChange(row: StepRowForm) {
  if (row.versionMode === 'active') {
    row.scriptVersionID = undefined
    syncJsonFromRows(stepRows.value)
    return
  }

  if (!row.scriptVersionID) {
    const activeVersion = getActiveVersion(row.scriptID)
    row.scriptVersionID = activeVersion?.versionID
  }
  syncJsonFromRows(stepRows.value)
}

function addStepRow() {
  const maxOrder = stepRows.value.reduce((max, item) => Math.max(max, item.stepOrder), 0)
  stepRows.value.push(defaultStepRow(maxOrder + 1))
  reindexStepOrders()
  syncJsonFromRows(stepRows.value)
}

function removeStepRow(index: number) {
  stepRows.value.splice(index, 1)
  if (stepRows.value.length === 0) {
    stepRows.value.push(defaultStepRow(1))
  }
  reindexStepOrders()
  syncJsonFromRows(stepRows.value)
}

async function saveFlowSteps() {
  const flowId = currentStepFlow.value?.flowID
  if (!flowId) {
    return
  }

  const payload = mapStepRowsToPayload(stepRows.value)
  if (!payload) {
    return
  }

  stepSaving.value = true
  try {
    await replaceFlowSteps(flowId, payload)
    stepRows.value = payload.map((item, index) => normalizeStepRow(item, index))
    reindexStepOrders()
    await Promise.all(stepRows.value.map((item) => ensureVersionsLoaded(item.scriptID)))
    syncJsonFromRows(stepRows.value)
    ElMessage.success('流程步骤已更新')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程步骤更新失败')
  } finally {
    stepSaving.value = false
  }
}

async function openMountDrawer(flow: ScriptFlow) {
  if (!flow.flowID) {
    ElMessage.warning('流程 ID 不存在，无法管理挂载')
    return
  }

  currentMountFlow.value = flow
  mounts.value = []
  resetMountForm(flow)
  mountDrawerVisible.value = true
  await loadMounts(flow.flowID)
}

async function loadMounts(flowId: number) {
  mountLoading.value = true
  try {
    mounts.value = await fetchFlowMounts(flowId)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '挂载列表加载失败')
  } finally {
    mountLoading.value = false
  }
}

function onMountScopeChange() {
  syncMountEventWithScope()
}

function onMountTargetTypeChange() {
  syncTargetIDWithTargetType()
}

async function addMount() {
  const flowId = currentMountFlow.value?.flowID
  if (!flowId) {
    return
  }

  const eventKey = mountForm.eventKey.trim()
  if (!eventKey) {
    ElMessage.warning('请填写事件键 eventKey')
    return
  }

  let targetID = 0
  if (mountForm.targetType !== 'global') {
    const parsedTargetID = parsePositiveInteger(mountForm.targetIDText)
    if (!parsedTargetID) {
      ElMessage.warning('目标 ID 需要是正整数')
      return
    }
    targetID = parsedTargetID
  }

  const payload: FlowMount = {
    flowID: flowId,
    scope: mountForm.scope,
    targetType: mountForm.targetType,
    targetID,
    eventKey,
    isEnabled: mountForm.isEnabled,
  }

  try {
    await createFlowMount(payload)
    ElMessage.success('挂载创建成功')
    resetMountForm(currentMountFlow.value)
    await loadMounts(flowId)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '挂载创建失败')
  }
}

async function deleteMount(mountId: number | undefined) {
  const flowId = currentMountFlow.value?.flowID
  if (!mountId || !flowId) {
    return
  }

  try {
    await removeFlowMount(mountId)
    ElMessage.success('挂载已删除')
    await loadMounts(flowId)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '挂载删除失败')
  }
}

onMounted(loadFlows)
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">流程工作台</h1>
        <p class="page-subtitle">面向 B 端运营的流程编排与挂载配置中心</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">新建流程</el-button>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      执行链路：脚本定义(scriptKey) -> 版本(激活或锁定) -> 流程步骤 -> 挂载(scope/eventKey/target) -> 事件触发执行。
    </el-alert>

    <el-table :data="flows" v-loading="loading" style="width: 100%">
      <el-table-column prop="flowID" label="ID" width="90" />
      <el-table-column prop="flowKey" label="流程键" min-width="160" />
      <el-table-column prop="flowName" label="流程名称" min-width="180" />
      <el-table-column label="描述" min-width="220">
        <template #default="scope">
          {{ scope.row.description || scope.row.flowDescription || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="默认触发事件" width="240">
        <template #default="scope">
          {{ formatEventKey((scope.row.meta && scope.row.meta.trigger) || (scope.row.extensionData && scope.row.extensionData.trigger)) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="scope">
          <el-switch
            :model-value="Boolean(scope.row.isEnabled)"
            @change="handleFlowStatusChange(scope.row, $event)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="300">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openEditDialog(scope.row)">编辑流程</el-button>
            <el-button link type="success" @click="openStepDrawer(scope.row)">编排步骤</el-button>
            <el-button link type="warning" @click="openMountDrawer(scope.row)">管理挂载</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="flowDialogVisible" :title="editingFlowId ? '编辑流程' : '新建流程'" width="760px">
      <el-form label-position="top">
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="流程名称" required>
              <el-input v-model="flowForm.flowName" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="流程键(flowKey)" required>
              <el-input v-model="flowForm.flowKey" placeholder="仅支持字母、数字、下划线和连字符" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="流程描述">
          <el-input v-model="flowForm.flowDescription" type="textarea" :rows="2" />
        </el-form-item>

        <el-form-item label="默认触发事件(eventKey)">
          <el-select
            v-model="flowForm.triggerEvent"
            filterable
            allow-create
            default-first-option
            placeholder="例如: file_post"
            style="width: 100%"
          >
            <el-option v-for="item in eventKeyOptions" :key="item" :label="formatEventKey(item)" :value="item" />
          </el-select>
        </el-form-item>

        <el-form-item label="流程元数据(meta, JSON)">
          <el-input v-model="flowForm.metaText" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="flowDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveFlow">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="stepDrawerVisible"
      size="920px"
      :title="`流程编排 - ${currentStepFlow?.flowName || '未命名流程'}`"
    >
      <el-alert type="info" :closable="false" show-icon>
        主编辑区为“步骤卡片”。卡片支持拖拽排序。高级 JSON 仅作为批量编辑入口。
      </el-alert>

      <el-alert type="success" :closable="false" class="runtime-input-hint" show-icon>
        脚本固定收到两个输入对象：<strong>payload</strong>（事件现场数据）和 <strong>context.stepInput</strong>（这里填写的“脚本运行参数”）。
        以 submission/file_post 为例，payload 常见字段有 savedPath、workID、authorID、trackID、fileHash、fileSize。
      </el-alert>

      <div class="step-toolbar">
        <el-button plain type="primary" @click="addStepRow">新增步骤</el-button>
      </div>

      <div class="step-form-list" v-loading="stepLoading">
        <article
          v-for="(row, index) in stepRows"
          :key="`${row.stepID || 'new'}-${index}`"
          class="step-row-card"
          :class="{ 'is-dragging': draggingStepIndex === index }"
          draggable="true"
          @dragstart="onStepDragStart(index)"
          @dragover="onStepDragOver"
          @drop="onStepDrop(index)"
          @dragend="onStepDragEnd"
        >
          <div class="step-row-head">
            <div class="step-row-head-left">
              <span class="drag-handle">::</span>
              <strong>步骤 {{ row.stepOrder }}</strong>
            </div>
            <el-space>
              <el-button link type="primary" :disabled="index === 0" @click="moveStepUp(index)">上移</el-button>
              <el-button link type="primary" :disabled="index === stepRows.length - 1" @click="moveStepDown(index)">下移</el-button>
              <el-button link type="danger" @click="removeStepRow(index)">删除</el-button>
            </el-space>
          </div>

          <el-row :gutter="10">
            <el-col :xs="24" :md="12">
              <el-form-item label="步骤名称" required>
                <el-input v-model="row.stepName" @change="syncJsonFromRows(stepRows)" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item label="脚本定义" required>
                <el-select
                  v-model="row.scriptID"
                  filterable
                  placeholder="请选择脚本"
                  style="width: 100%"
                  @change="onStepScriptChange(row)"
                >
                  <el-option
                    v-for="script in scripts"
                    :key="script.scriptID"
                    :label="`${script.scriptName} (${script.scriptKey || 'no-key'})${script.isEnabled === false ? ' [停用]' : ''}`"
                    :value="script.scriptID"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="10">
            <el-col :xs="24" :md="12">
              <el-form-item label="版本策略">
                <el-radio-group v-model="row.versionMode" size="small" @change="onStepVersionModeChange(row)">
                  <el-radio-button label="active">跟随激活版本</el-radio-button>
                  <el-radio-button label="fixed">锁定版本</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12" v-if="row.versionMode === 'fixed'">
              <el-form-item label="锁定版本" required>
                <el-select
                  v-model="row.scriptVersionID"
                  filterable
                  clearable
                  :loading="isVersionListLoading(row.scriptID)"
                  placeholder="请选择脚本版本"
                  style="width: 100%"
                  @visible-change="(visible) => visible && ensureVersionsLoaded(row.scriptID)"
                  @change="syncJsonFromRows(stepRows)"
                >
                  <el-option
                    v-for="version in getVersionsForScript(row.scriptID)"
                    :key="version.versionID"
                    :label="formatVersionOption(version)"
                    :value="version.versionID"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-alert type="info" :closable="false" class="step-exec-path">
            {{ resolveStepVersionLabel(row) }}；执行文件路径：{{ resolveStepExecutionPath(row) }}
          </el-alert>

          <el-row :gutter="10">
            <el-col :xs="24" :md="8">
              <el-form-item label="失败策略">
                <el-select v-model="row.failureStrategy" style="width: 100%" @change="syncJsonFromRows(stepRows)">
                  <el-option
                    v-for="item in failureStrategyOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="8">
              <el-form-item label="超时(ms)">
                <el-input-number v-model="row.timeoutMs" :min="1" :step="100" style="width: 100%" @change="syncJsonFromRows(stepRows)" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="8">
              <el-form-item label="启用状态">
                <el-switch v-model="row.isEnabled" @change="syncJsonFromRows(stepRows)" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item label="脚本运行参数(JSON 对象，可留空 {})">
            <el-input
              v-model="row.inputTemplateText"
              type="textarea"
              :rows="4"
              placeholder='例如字数统计脚本可留空 {}; 或 {"savedPathField":"savedPath","patchKey":"word_count"}'
              @change="syncJsonFromRows(stepRows)"
            />
            <div class="field-help">
              这里填写的是 <code>context.stepInput</code>。不要把 payload 填在这里。字数统计脚本默认不需要参数，直接留空 <code>{}</code> 即可。
            </div>
          </el-form-item>
        </article>
      </div>

      <el-collapse v-model="advancedJsonOpen" class="advanced-json">
        <el-collapse-item name="json">
          <template #title>高级 JSON 编辑（可选）</template>
          <el-alert type="warning" :closable="false" show-icon>
            仅用于批量修改。建议先点击“从卡片生成 JSON”，修改后再点击“应用到卡片”。
          </el-alert>

          <el-input v-model="stepEditorText" type="textarea" :rows="14" class="editor-area" />

          <div class="json-actions">
            <el-button @click="refreshJsonFromRows">从卡片生成 JSON</el-button>
            <el-button type="primary" @click="applyJsonToRows">应用到卡片</el-button>
          </div>
        </el-collapse-item>
      </el-collapse>

      <div class="drawer-footer">
        <el-button @click="stepDrawerVisible = false">关闭</el-button>
        <el-button type="primary" :loading="stepSaving" @click="saveFlowSteps">保存步骤</el-button>
      </div>
    </el-drawer>

    <el-drawer
      v-model="mountDrawerVisible"
      size="840px"
      :title="`挂载配置 - ${currentMountFlow?.flowName || '未命名流程'}`"
    >
      <el-alert type="info" :closable="false" show-icon>
        挂载由两部分组成：触发上下文(scope/eventKey) 与作用范围(targetType/targetID)。
      </el-alert>

      <div class="mount-form">
        <h4 class="section-title">触发上下文</h4>
        <el-form label-position="top">
          <el-row :gutter="10">
            <el-col :xs="24" :md="8">
              <el-form-item label="执行域(scope)">
                <el-select v-model="mountForm.scope" style="width: 100%" @change="onMountScopeChange">
                  <el-option v-for="item in mountScopeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="10">
              <el-form-item label="事件键(eventKey)">
                <el-select
                  v-model="mountForm.eventKey"
                  filterable
                  allow-create
                  default-first-option
                  placeholder="例如: file_post"
                  style="width: 100%"
                >
                  <el-option v-for="item in mountEventOptions" :key="item" :label="formatEventKey(item)" :value="item" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item label="启用状态">
                <el-switch v-model="mountForm.isEnabled" />
              </el-form-item>
            </el-col>
          </el-row>

          <h4 class="section-title">作用范围</h4>
          <el-row :gutter="10">
            <el-col :xs="24" :md="8">
              <el-form-item label="目标范围(targetType)">
                <el-select v-model="mountForm.targetType" style="width: 100%" @change="onMountTargetTypeChange">
                  <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="8">
              <el-form-item label="目标ID(targetID)">
                <el-input
                  v-model="mountForm.targetIDText"
                  :disabled="mountForm.targetType === 'global'"
                  placeholder="targetType 非 global 时需填正整数"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="8" class="mount-action-col">
              <el-form-item label="操作">
                <el-button type="primary" style="width: 100%" @click="addMount">新增挂载</el-button>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>

        <el-alert type="success" :closable="false" class="mount-preview">
          {{ mountPreviewText }}
        </el-alert>
      </div>

      <el-table :data="mounts" v-loading="mountLoading" style="width: 100%">
        <el-table-column prop="mountID" label="挂载ID" width="100" />
        <el-table-column label="执行域" width="130">
          <template #default="scope">
            {{ formatScope(resolveScope(scope.row)) }}
          </template>
        </el-table-column>
        <el-table-column label="目标范围" width="120">
          <template #default="scope">
            {{ formatTargetType(resolveTargetType(scope.row)) }}
          </template>
        </el-table-column>
        <el-table-column label="目标ID" width="120">
          <template #default="scope">
            {{ resolveTargetID(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column label="事件键" min-width="220">
          <template #default="scope">
            {{ formatEventKey(scope.row.eventKey) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.isEnabled ? 'success' : 'info'">{{ scope.row.isEnabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" min-width="180" />
        <el-table-column label="操作" width="100">
          <template #default="scope">
            <el-button link type="danger" @click="deleteMount(scope.row.mountID)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-alert v-if="scripts.length === 0" type="warning" :closable="false" class="hint">
      当前未发现可用脚本，请先在脚本库创建脚本定义并上传版本。
    </el-alert>
  </section>
</template>

<style scoped>
.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
  gap: 12px;
  flex-wrap: wrap;
}

.step-toolbar {
  margin: 12px 0;
  display: flex;
  justify-content: flex-end;
}

.runtime-input-hint {
  margin-top: 10px;
}

.step-form-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 12px;
}

.step-row-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  padding: 12px;
  background: #fff;
}

.step-row-card.is-dragging {
  opacity: 0.65;
  border-color: #409eff;
}

.field-help {
  font-size: 12px;
  color: #6b7280;
  line-height: 1.4;
  margin-top: 6px;
}

.step-row-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.step-row-head-left {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.drag-handle {
  color: #9ca3af;
  user-select: none;
  cursor: move;
  font-weight: bold;
}

.step-exec-path {
  margin-bottom: 12px;
}

.advanced-json {
  margin-top: 12px;
}

.editor-area {
  margin-top: 12px;
}

.json-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.drawer-footer {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.mount-form {
  margin: 12px 0;
}

.section-title {
  margin: 8px 0;
  font-size: 14px;
  color: #111827;
}

.mount-action-col {
  display: flex;
  align-items: flex-end;
}

.mount-preview {
  margin: 8px 0 12px;
}

.hint {
  margin-top: 14px;
}
</style>
