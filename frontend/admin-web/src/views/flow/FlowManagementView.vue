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
import { fetchScriptDefinitions } from '@/services/repositories/scriptRepository'
import type {
  FlowMount,
  FlowMountScope,
  FlowStep,
  ScriptDefinition,
  ScriptFlow,
} from '@/types/api'

type StepEditorMode = 'form' | 'json'

interface StepRowForm {
  stepID?: number
  stepOrder: number
  stepName: string
  scriptID?: number
  scriptVersionID?: number
  isEnabled: boolean
  failureStrategy: string
  timeoutMs: number
  inputTemplateText: string
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
  triggerEvent: '',
  metaText: '{\n  "trigger": "work_created"\n}',
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
  scope: 'track' as FlowMountScope,
  targetType: 'track',
  targetIDText: '',
  eventKey: 'work_created',
  isEnabled: true,
})

const eventKeyOptions = [
  'work_created',
  'work_submitted',
  'review_assigned',
  'review_completed',
]

const failureStrategyOptions = [
  { label: '继续后续步骤', value: 'CONTINUE' },
  { label: '立即终止流程', value: 'STOP' },
  { label: '按策略重试', value: 'RETRY' },
]

const mountScopeOptions = [
  { label: '全局', value: 'global' as FlowMountScope },
  { label: '比赛', value: 'contest' as FlowMountScope },
  { label: '赛道', value: 'track' as FlowMountScope },
]

const targetTypeOptions = computed(() => {
  if (mountForm.scope === 'global') {
    return [{ label: '全局', value: 'global' }]
  }
  if (mountForm.scope === 'contest') {
    return [{ label: '比赛', value: 'contest' }]
  }
  return [{ label: '赛道', value: 'track' }]
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
  if (!flow) {
    return 'work_created'
  }
  const meta = normalizeFlowMeta(flow)
  if (typeof meta.trigger === 'string' && meta.trigger.trim()) {
    return meta.trigger.trim()
  }
  if (typeof meta.eventKey === 'string' && meta.eventKey.trim()) {
    return meta.eventKey.trim()
  }
  return 'work_created'
}

function defaultStepRow(order: number): StepRowForm {
  return {
    stepOrder: order,
    stepName: `步骤 ${order}`,
    scriptID: scripts.value[0]?.scriptID,
    scriptVersionID: undefined,
    isEnabled: true,
    failureStrategy: 'CONTINUE',
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
    failureStrategy: typeof step.failureStrategy === 'string' ? step.failureStrategy : 'CONTINUE',
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
      failureStrategy: row.failureStrategy,
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

function resolveScope(mount: FlowMount): FlowMountScope {
  if (mount.scope === 'contest' || mount.scope === 'track' || mount.scope === 'global') {
    return mount.scope
  }
  if (mount.targetType === 'contest' || mount.containerType === 'contest') {
    return 'contest'
  }
  if (mount.targetType === 'track' || mount.containerType === 'track') {
    return 'track'
  }
  return 'global'
}

function resolveTargetType(mount: FlowMount) {
  return mount.targetType || mount.containerType || (resolveScope(mount) === 'global' ? 'global' : resolveScope(mount))
}

function resolveTargetID(mount: FlowMount) {
  const targetID = mount.targetID ?? mount.containerID
  if (typeof targetID === 'number') {
    return targetID
  }
  return resolveScope(mount) === 'global' ? 0 : null
}

function resetFlowForm() {
  Object.assign(flowForm, {
    flowName: '',
    flowKey: '',
    flowDescription: '',
    triggerEvent: 'work_created',
    metaText: '{\n  "trigger": "work_created"\n}',
  })
}

function resetMountForm(flow: ScriptFlow | null) {
  Object.assign(mountForm, {
    scope: 'track' as FlowMountScope,
    targetType: 'track',
    targetIDText: '',
    eventKey: detectFlowTrigger(flow),
    isEnabled: true,
  })
}

function syncTargetTypeWithScope() {
  if (mountForm.scope === 'global') {
    mountForm.targetType = 'global'
    mountForm.targetIDText = '0'
    return
  }

  mountForm.targetType = mountForm.scope
  if (mountForm.targetIDText === '0') {
    mountForm.targetIDText = ''
  }
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
  if (!flowName) {
    ElMessage.warning('请填写流程名称')
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
    flowKey: flowForm.flowKey.trim() || undefined,
    flowDescription: flowForm.flowDescription.trim(),
    description: flowForm.flowDescription.trim(),
    meta,
    extensionData: meta,
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
  syncTargetTypeWithScope()
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
  if (mountForm.scope !== 'global') {
    targetID = Number(mountForm.targetIDText)
    if (!Number.isInteger(targetID) || targetID <= 0) {
      ElMessage.warning('目标 ID 需要是正整数')
      return
    }
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
        <h1 class="page-title">流程管理</h1>
        <p class="page-subtitle">管理流程定义、步骤编排与全局/比赛/赛道三层挂载</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">新建流程</el-button>
    </div>

    <el-table :data="flows" v-loading="loading" style="width: 100%">
      <el-table-column prop="flowID" label="ID" width="90" />
      <el-table-column prop="flowKey" label="流程键" min-width="140" />
      <el-table-column prop="flowName" label="流程名称" min-width="160" />
      <el-table-column label="描述" min-width="200">
        <template #default="scope">
          {{ scope.row.description || scope.row.flowDescription || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="触发事件" width="150">
        <template #default="scope">
          {{ (scope.row.meta && scope.row.meta.trigger) || (scope.row.extensionData && scope.row.extensionData.trigger) || '-' }}
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
            <el-form-item label="流程键(flowKey)">
              <el-input v-model="flowForm.flowKey" placeholder="建议用于业务唯一标识" />
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
            placeholder="例如: work_created"
            style="width: 100%"
          >
            <el-option v-for="item in eventKeyOptions" :key="item" :label="item" :value="item" />
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
        步骤支持表单模式与 JSON 模式。字段包含 scriptVersionID、failureStrategy、timeoutMs、inputTemplate。
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
        挂载层级固定为 global / contest / track。eventKey 默认取流程触发器，可手动修改。
      </el-alert>

      <div class="mount-form">
        <el-form label-position="top">
          <el-row :gutter="10">
            <el-col :xs="24" :md="6">
              <el-form-item label="层级(scope)">
                <el-select v-model="mountForm.scope" style="width: 100%" @change="onMountScopeChange">
                  <el-option v-for="item in mountScopeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item label="目标类型(targetType)">
                <el-select v-model="mountForm.targetType" style="width: 100%">
                  <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :md="6">
              <el-form-item label="目标ID(targetID)">
                <el-input
                  v-model="mountForm.targetIDText"
                  :disabled="mountForm.scope === 'global'"
                  placeholder="contest/track 需填正整数"
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
                  placeholder="例如: work_created"
                  style="width: 100%"
                >
                  <el-option v-for="item in eventKeyOptions" :key="item" :label="item" :value="item" />
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
        <el-table-column label="层级" width="110">
          <template #default="scope">
            {{ resolveScope(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column label="目标类型" width="120">
          <template #default="scope">
            {{ resolveTargetType(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column label="目标ID" width="120">
          <template #default="scope">
            {{ resolveTargetID(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column prop="eventKey" label="事件键" min-width="160" />
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
