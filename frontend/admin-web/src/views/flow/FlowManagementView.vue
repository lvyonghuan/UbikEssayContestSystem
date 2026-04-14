<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  buildSubmissionWordCountPreset,
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

type StepEditorMode = 'form' | 'json'
type FlowManageMode = 'guided' | 'expert'
type FailureStrategy = 'fail_close' | 'fail_open' | 'retry'

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

interface StepRowForm {
  stepID?: number
  stepOrder: number
  stepName: string
  scriptID?: number
  scriptVersionID?: number
  isEnabled: boolean
  failureStrategy: FailureStrategy | string
  timeoutMs: number
  inputTemplateText: string
}

const manageMode = ref<FlowManageMode>('guided')
const loading = ref(false)
const flows = ref<ScriptFlow[]>([])
const scripts = ref<ScriptDefinition[]>([])

const guidedSubmitting = ref(false)
const guidedVersionLoading = ref(false)
const guidedVersions = ref<ScriptVersion[]>([])
const guidedForm = reactive({
  scriptID: undefined as number | undefined,
  scriptVersionID: undefined as number | undefined,
  flowName: '投稿后字数统计',
  flowKey: '',
  scope: 'submission' as FlowMountScope,
  eventKey: 'file_post',
  targetType: 'track' as FlowMountTargetType,
  targetIDText: '',
  timeoutMs: 20000,
  failureStrategy: 'fail_close' as FailureStrategy,
})

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
const stepEditorMode = ref<StepEditorMode>('form')
const stepEditorText = ref('[]')
const stepRows = ref<StepRowForm[]>([])
const currentStepFlow = ref<ScriptFlow | null>(null)

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

function defaultStepRow(order: number): StepRowForm {
  return {
    stepOrder: order,
    stepName: `步骤 ${order}`,
    scriptID: scripts.value[0]?.scriptID,
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
    scriptVersionID: step.scriptVersionID,
    isEnabled: step.isEnabled ?? true,
    failureStrategy: normalizeFailureStrategy(step.failureStrategy),
    timeoutMs: Number.isInteger(step.timeoutMs) && (step.timeoutMs as number) > 0 ? (step.timeoutMs as number) : 5000,
    inputTemplateText: JSON.stringify(inputTemplate, null, 2),
  }
}

function mapStepRowsToPayload(rows: StepRowForm[]) {
  const nextSteps: FlowStep[] = []
  for (let index = 0; index < rows.length; index += 1) {
    const row = rows[index]
    if (!Number.isInteger(row.stepOrder) || row.stepOrder <= 0) {
      ElMessage.warning(`第 ${index + 1} 行步骤顺序需要是正整数`)
      return null
    }
    if (!row.scriptID) {
      ElMessage.warning(`第 ${index + 1} 行需要选择脚本`)
      return null
    }
    if (!row.stepName.trim()) {
      ElMessage.warning(`第 ${index + 1} 行需要填写步骤名称`)
      return null
    }
    if (!Number.isInteger(row.timeoutMs) || row.timeoutMs <= 0) {
      ElMessage.warning(`第 ${index + 1} 行超时时间需要是正整数`)
      return null
    }

    const inputTemplate = parseJsonObject(
      row.inputTemplateText,
      `第 ${index + 1} 行输入模板必须是合法 JSON 对象`,
    )
    if (!inputTemplate) {
      return null
    }

    nextSteps.push({
      stepID: row.stepID,
      stepOrder: row.stepOrder,
      stepName: row.stepName.trim(),
      scriptID: row.scriptID,
      scriptVersionID: row.scriptVersionID,
      isEnabled: row.isEnabled,
      failureStrategy: normalizeFailureStrategy(row.failureStrategy),
      timeoutMs: row.timeoutMs,
      inputTemplate,
      stepConfig: inputTemplate,
    })
  }
  return nextSteps
}

function parseStepsFromJsonEditor() {
  let parsed: unknown
  try {
    parsed = JSON.parse(stepEditorText.value || '[]')
  } catch {
    ElMessage.error('步骤配置必须是合法 JSON 数组')
    return null
  }

  if (!Array.isArray(parsed)) {
    ElMessage.error('步骤配置必须是 JSON 数组')
    return null
  }

  const rows = parsed.map((item, index) => normalizeStepRow(item as FlowStep, index))
  return mapStepRowsToPayload(rows)
}

function syncJsonFromRows(rows: StepRowForm[]) {
  const payload = mapStepRowsToPayload(rows)
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

function generateGuidedFlowKey() {
  const now = new Date()
  const stamp = [
    now.getUTCFullYear(),
    String(now.getUTCMonth() + 1).padStart(2, '0'),
    String(now.getUTCDate()).padStart(2, '0'),
    String(now.getUTCHours()).padStart(2, '0'),
    String(now.getUTCMinutes()).padStart(2, '0'),
    String(now.getUTCSeconds()).padStart(2, '0'),
  ].join('')
  return `submission_word_count_${stamp}_${Math.floor(Math.random() * 1000)}`
}

function getPreferredScriptID() {
  return scripts.value.find((item) => item.isEnabled !== false && item.scriptID)?.scriptID || scripts.value[0]?.scriptID
}

function parsePositiveInteger(text: string) {
  const value = Number(text)
  if (!Number.isInteger(value) || value <= 0) {
    return null
  }
  return value
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

function resetGuidedForm() {
  guidedForm.flowName = '投稿后字数统计'
  guidedForm.flowKey = generateGuidedFlowKey()
  guidedForm.scope = 'submission'
  guidedForm.eventKey = 'file_post'
  guidedForm.targetType = 'track'
  guidedForm.targetIDText = ''
  guidedForm.timeoutMs = 20000
  guidedForm.failureStrategy = 'fail_close'
  guidedForm.scriptID = getPreferredScriptID()
  guidedForm.scriptVersionID = undefined
  guidedVersions.value = []
}

async function loadGuidedVersions(scriptID: number | undefined) {
  if (!scriptID) {
    guidedVersions.value = []
    guidedForm.scriptVersionID = undefined
    return
  }

  guidedVersionLoading.value = true
  try {
    const versions = await fetchScriptVersions(scriptID)
    guidedVersions.value = versions
    const activeVersion = versions.find((item) => item.isActive)
    guidedForm.scriptVersionID = activeVersion?.versionID || versions[0]?.versionID
  } catch (error) {
    guidedVersions.value = []
    guidedForm.scriptVersionID = undefined
    ElMessage.error(error instanceof Error ? error.message : '脚本版本加载失败')
  } finally {
    guidedVersionLoading.value = false
  }
}

function onGuidedTargetTypeChange() {
  if (guidedForm.targetType === 'global') {
    guidedForm.targetIDText = '0'
    return
  }
  if (guidedForm.targetIDText === '0') {
    guidedForm.targetIDText = ''
  }
}

async function createGuidedWordCountFlow() {
  const scriptID = guidedForm.scriptID
  const scriptVersionID = guidedForm.scriptVersionID
  const flowName = guidedForm.flowName.trim()
  const flowKey = guidedForm.flowKey.trim()

  if (!scriptID) {
    ElMessage.warning('请先选择脚本')
    return
  }
  if (!scriptVersionID) {
    ElMessage.warning('请先选择脚本版本')
    return
  }
  if (!flowName) {
    ElMessage.warning('请填写流程名称')
    return
  }
  if (!validateFlowKey(flowKey, '流程键(flowKey)')) {
    return
  }

  let targetID = 0
  if (guidedForm.targetType !== 'global') {
    const parsedTargetID = parsePositiveInteger(guidedForm.targetIDText)
    if (!parsedTargetID) {
      ElMessage.warning('目标 ID 需要是正整数')
      return
    }
    targetID = parsedTargetID
  }

  guidedSubmitting.value = true
  try {
    const preset = buildSubmissionWordCountPreset({
      flowName,
      flowKey,
      scriptID,
      scriptVersionID,
      scope: guidedForm.scope,
      eventKey: guidedForm.eventKey,
      targetType: guidedForm.targetType,
      targetID,
      failureStrategy: guidedForm.failureStrategy,
      timeoutMs: guidedForm.timeoutMs,
    })

    const createdFlow = await createScriptFlow(preset.flow)
    if (!createdFlow.flowID) {
      throw new Error('流程创建结果缺少 flowID')
    }

    await replaceFlowSteps(createdFlow.flowID, preset.steps)
    await createFlowMount({
      ...preset.mount,
      flowID: createdFlow.flowID,
    })
    await updateScriptFlowStatus(createdFlow.flowID, { isEnabled: true })

    ElMessage.success('已完成字数统计模板创建，可在专家模式继续调整')
    await loadFlows()
    resetGuidedForm()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '模板创建失败')
  } finally {
    guidedSubmitting.value = false
  }
}

async function loadFlows() {
  loading.value = true
  try {
    const [nextFlows, nextScripts] = await Promise.all([fetchScriptFlows(), fetchScriptDefinitions()])
    flows.value = nextFlows
    scripts.value = nextScripts

    if (!guidedForm.scriptID || !scripts.value.some((item) => item.scriptID === guidedForm.scriptID)) {
      guidedForm.scriptID = getPreferredScriptID()
    }
    if (!guidedForm.flowKey) {
      guidedForm.flowKey = generateGuidedFlowKey()
    }
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
  stepEditorMode.value = 'form'
  stepRows.value = [defaultStepRow(1)]
  stepEditorText.value = '[]'
  stepDrawerVisible.value = true
  stepLoading.value = true

  try {
    const nextSteps = await fetchFlowSteps(flow.flowID)
    const normalizedRows = nextSteps.length > 0
      ? nextSteps.map((item, index) => normalizeStepRow(item, index))
      : [defaultStepRow(1)]
    stepRows.value = normalizedRows
    syncJsonFromRows(stepRows.value)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程步骤加载失败')
  } finally {
    stepLoading.value = false
  }
}

function onStepEditorModeChange(mode: StepEditorMode) {
  if (mode === 'json') {
    syncJsonFromRows(stepRows.value)
    return
  }

  const parsed = parseStepsFromJsonEditor()
  if (!parsed) {
    stepEditorMode.value = 'json'
    return
  }
  stepRows.value = parsed.map((item, index) => normalizeStepRow(item, index))
}

function addStepRow() {
  const maxOrder = stepRows.value.reduce((max, item) => Math.max(max, item.stepOrder), 0)
  stepRows.value.push(defaultStepRow(maxOrder + 1))
}

function removeStepRow(index: number) {
  stepRows.value.splice(index, 1)
  if (stepRows.value.length === 0) {
    stepRows.value.push(defaultStepRow(1))
  }
}

async function saveFlowSteps() {
  const flowId = currentStepFlow.value?.flowID
  if (!flowId) {
    return
  }

  const payload = stepEditorMode.value === 'form'
    ? mapStepRowsToPayload(stepRows.value)
    : parseStepsFromJsonEditor()

  if (!payload) {
    return
  }

  stepSaving.value = true
  try {
    await replaceFlowSteps(flowId, payload)
    stepRows.value = payload.map((item, index) => normalizeStepRow(item, index))
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

watch(
  () => guidedForm.scriptID,
  (scriptID) => {
    void loadGuidedVersions(scriptID)
  },
)

onMounted(async () => {
  await loadFlows()
  resetGuidedForm()
})
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">流程管理</h1>
        <p class="page-subtitle">默认新手向导，专家模式保留完整编排能力</p>
      </div>
      <div class="header-actions">
        <el-radio-group v-model="manageMode" size="small">
          <el-radio-button label="guided">新手向导</el-radio-button>
          <el-radio-button label="expert">专家模式</el-radio-button>
        </el-radio-group>
        <el-button v-if="manageMode === 'expert'" type="primary" @click="openCreateDialog">新建流程</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      新手向导会自动创建“投稿后字数统计”所需的流程、步骤和挂载。专家模式用于复杂场景。
    </el-alert>

    <section v-if="manageMode === 'guided'" class="guided-panel">
      <h3 class="guided-title">一键配置：投稿后字数统计</h3>
      <p class="guided-hint">默认配置：scope=submission、eventKey=file_post、failureStrategy=fail_close。</p>

      <el-form label-position="top">
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="选择脚本" required>
              <el-select v-model="guidedForm.scriptID" filterable placeholder="请选择脚本" style="width: 100%">
                <el-option
                  v-for="script in scripts"
                  :key="script.scriptID"
                  :label="`${script.scriptName}${script.isEnabled === false ? ' (已停用)' : ''}`"
                  :value="script.scriptID"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="选择脚本版本" required>
              <el-select
                v-model="guidedForm.scriptVersionID"
                :loading="guidedVersionLoading"
                filterable
                placeholder="请选择版本"
                style="width: 100%"
              >
                <el-option
                  v-for="version in guidedVersions"
                  :key="version.versionID"
                  :label="`${version.versionName || `v${version.versionNum || version.versionID}`}${version.isActive ? ' (已激活)' : ''}`"
                  :value="version.versionID"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="流程名称" required>
              <el-input v-model="guidedForm.flowName" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="流程键(flowKey)" required>
              <el-input v-model="guidedForm.flowKey" placeholder="仅支持字母、数字、下划线和连字符" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :xs="24" :md="6">
            <el-form-item label="执行域(scope)">
              <el-select v-model="guidedForm.scope" disabled style="width: 100%">
                <el-option v-for="item in mountScopeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="6">
            <el-form-item label="事件键(eventKey)">
              <el-input :model-value="formatEventKey(guidedForm.eventKey)" disabled />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="6">
            <el-form-item label="目标范围(targetType)">
              <el-select v-model="guidedForm.targetType" style="width: 100%" @change="onGuidedTargetTypeChange">
                <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="6">
            <el-form-item label="目标ID(targetID)">
              <el-input
                v-model="guidedForm.targetIDText"
                :disabled="guidedForm.targetType === 'global'"
                placeholder="targetType 非 global 时需填正整数"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="失败策略">
              <el-select v-model="guidedForm.failureStrategy" style="width: 100%">
                <el-option v-for="item in failureStrategyOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="超时(ms)">
              <el-input-number v-model="guidedForm.timeoutMs" :min="1000" :step="500" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <div class="guided-actions">
        <el-button @click="resetGuidedForm">重置默认</el-button>
        <el-button type="primary" :loading="guidedSubmitting" @click="createGuidedWordCountFlow">创建模板流程</el-button>
      </div>
    </section>

    <template v-else>
    <el-table :data="flows" v-loading="loading" style="width: 100%">
      <el-table-column prop="flowID" label="ID" width="90" />
      <el-table-column prop="flowKey" label="流程键" min-width="140" />
      <el-table-column prop="flowName" label="流程名称" min-width="160" />
      <el-table-column label="描述" min-width="200">
        <template #default="scope">
          {{ scope.row.description || scope.row.flowDescription || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="触发事件" width="220">
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
      <el-table-column label="操作" width="260">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openEditDialog(scope.row)">编辑</el-button>
            <el-button link type="success" @click="openStepDrawer(scope.row)">步骤</el-button>
            <el-button link type="warning" @click="openMountDrawer(scope.row)">挂载</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="flowDialogVisible" :title="editingFlowId ? '编辑流程' : '新建流程'" width="720px">
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
        <el-form-item label="默认事件键(eventKey)">
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
        <el-form-item label="流程元数据(meta，JSON)">
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
      size="840px"
      :title="`步骤编辑 - ${currentStepFlow?.flowName || '未命名流程'}`"
    >
      <el-alert type="info" :closable="false" show-icon>
        步骤支持表单模式与 JSON 模式。failureStrategy 请使用 fail_close / fail_open / retry。
      </el-alert>

      <div class="step-mode-bar">
        <el-radio-group v-model="stepEditorMode" @change="onStepEditorModeChange">
          <el-radio-button label="form">表单模式</el-radio-button>
          <el-radio-button label="json">JSON 模式</el-radio-button>
        </el-radio-group>
      </div>

      <div v-if="stepEditorMode === 'form'" class="step-form-list" v-loading="stepLoading">
        <article v-for="(row, index) in stepRows" :key="`${row.stepID || 'new'}-${index}`" class="step-row-card">
          <div class="step-row-head">
            <strong>步骤 {{ index + 1 }}</strong>
            <el-button link type="danger" @click="removeStepRow(index)">删除</el-button>
          </div>

          <el-row :gutter="10">
            <el-col :xs="24" :md="8">
              <el-form-item label="顺序">
                <el-input-number v-model="row.stepOrder" :min="1" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="16">
              <el-form-item label="步骤名称">
                <el-input v-model="row.stepName" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="10">
            <el-col :xs="24" :md="12">
              <el-form-item label="脚本">
                <el-select v-model="row.scriptID" placeholder="请选择脚本" style="width: 100%">
                  <el-option
                    v-for="script in scripts"
                    :key="script.scriptID"
                    :label="script.scriptName"
                    :value="script.scriptID"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="12">
              <el-form-item label="脚本版本ID">
                <el-input-number v-model="row.scriptVersionID" :min="1" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="10">
            <el-col :xs="24" :md="8">
              <el-form-item label="失败策略">
                <el-select v-model="row.failureStrategy" style="width: 100%">
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
                <el-input-number v-model="row.timeoutMs" :min="1" :step="100" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="8">
              <el-form-item label="启用状态">
                <el-switch v-model="row.isEnabled" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item label="输入模板(JSON 对象)">
            <el-input v-model="row.inputTemplateText" type="textarea" :rows="4" />
          </el-form-item>
        </article>

        <el-button plain type="primary" @click="addStepRow">新增步骤</el-button>
      </div>

      <el-input
        v-else
        v-model="stepEditorText"
        type="textarea"
        :rows="20"
        class="editor-area"
        v-loading="stepLoading"
      />

      <div class="drawer-footer">
        <el-button @click="stepDrawerVisible = false">关闭</el-button>
        <el-button type="primary" :loading="stepSaving" @click="saveFlowSteps">保存步骤</el-button>
      </div>
    </el-drawer>

    <el-drawer
      v-model="mountDrawerVisible"
      size="840px"
      :title="`挂载管理 - ${currentMountFlow?.flowName || '未命名流程'}`"
    >
      <el-alert type="info" :closable="false" show-icon>
        scope 表示执行域（submission/system/judge），targetType 表示作用范围（global/contest/track）。
      </el-alert>

      <div class="mount-form">
        <el-form label-position="top">
          <el-row :gutter="10">
            <el-col :xs="24" :md="6">
              <el-form-item label="执行域(scope)">
                <el-select v-model="mountForm.scope" style="width: 100%" @change="onMountScopeChange">
                  <el-option v-for="item in mountScopeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item label="目标范围(targetType)">
                <el-select v-model="mountForm.targetType" style="width: 100%" @change="onMountTargetTypeChange">
                  <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item label="目标ID(targetID)">
                <el-input
                  v-model="mountForm.targetIDText"
                  :disabled="mountForm.targetType === 'global'"
                  placeholder="targetType 非 global 时需填正整数"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item label="启用状态">
                <el-switch v-model="mountForm.isEnabled" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="10">
            <el-col :xs="24" :md="18">
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
            <el-col :xs="24" :md="6" class="mount-action-col">
              <el-form-item label="操作">
                <el-button type="primary" style="width: 100%" @click="addMount">新增挂载</el-button>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
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
      当前未发现可用脚本，建议先在脚本管理页创建脚本后再配置流程步骤。
    </el-alert>
    </template>
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

.step-mode-bar {
  margin: 12px 0;
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

.step-row-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.editor-area {
  margin-top: 12px;
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

.mount-action-col {
  display: flex;
  align-items: flex-end;
}

.hint {
  margin-top: 14px;
}
</style>
