<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
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
import type { FlowMount, FlowStep, ScriptDefinition, ScriptFlow } from '@/types/api'

const loading = ref(false)
const flows = ref<ScriptFlow[]>([])
const scripts = ref<ScriptDefinition[]>([])

const dialogVisible = ref(false)
const editingFlowId = ref<number | null>(null)
const form = reactive({
  flowName: '',
  flowDescription: '',
  extensionDataText: '{\n  "trigger": "manual"\n}',
})

const stepDrawerVisible = ref(false)
const stepLoading = ref(false)
const stepSaving = ref(false)
const stepEditorText = ref('[]')
const currentStepFlow = ref<ScriptFlow | null>(null)

const mountDrawerVisible = ref(false)
const mountLoading = ref(false)
const mounts = ref<FlowMount[]>([])
const currentMountFlow = ref<ScriptFlow | null>(null)
const mountForm = reactive({
  containerType: 'track',
  containerIDText: '',
  mountConfigText: '{\n  "phase": "submission"\n}',
})

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

function resetForm() {
  Object.assign(form, {
    flowName: '',
    flowDescription: '',
    extensionDataText: '{\n  "trigger": "manual"\n}',
  })
}

function openCreateDialog() {
  editingFlowId.value = null
  resetForm()
  dialogVisible.value = true
}

function openEditDialog(flow: ScriptFlow) {
  editingFlowId.value = flow.flowID || null
  Object.assign(form, {
    flowName: flow.flowName,
    flowDescription: flow.flowDescription || '',
    extensionDataText: JSON.stringify(flow.extensionData || {}, null, 2),
  })
  dialogVisible.value = true
}

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

async function saveFlow() {
  const flowName = form.flowName.trim()
  if (!flowName) {
    ElMessage.warning('请填写流程名称')
    return
  }

  const extensionData = parseJsonObject(form.extensionDataText, '扩展配置必须是合法 JSON 对象')
  if (!extensionData) {
    return
  }

  const payload: ScriptFlow = {
    flowName,
    flowDescription: form.flowDescription.trim(),
    extensionData,
  }

  try {
    if (editingFlowId.value) {
      await updateScriptFlow(editingFlowId.value, payload)
      ElMessage.success('流程更新成功')
    } else {
      await createScriptFlow(payload)
      ElMessage.success('流程创建成功')
    }
    dialogVisible.value = false
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
  stepEditorText.value = '[]'
  stepDrawerVisible.value = true
  stepLoading.value = true

  try {
    const steps = await fetchFlowSteps(flow.flowID)
    stepEditorText.value = JSON.stringify(steps, null, 2)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '流程步骤加载失败')
  } finally {
    stepLoading.value = false
  }
}

async function saveFlowSteps() {
  const flowId = currentStepFlow.value?.flowID
  if (!flowId) {
    return
  }

  let parsedSteps: FlowStep[]
  try {
    const parsed = JSON.parse(stepEditorText.value || '[]')
    if (!Array.isArray(parsed)) {
      throw new Error('steps should be array')
    }
    parsedSteps = parsed as FlowStep[]
  } catch {
    ElMessage.error('步骤配置必须是合法 JSON 数组')
    return
  }

  stepSaving.value = true
  try {
    await replaceFlowSteps(flowId, parsedSteps)
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
  Object.assign(mountForm, {
    containerType: 'track',
    containerIDText: '',
    mountConfigText: '{\n  "phase": "submission"\n}',
  })

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

async function addMount() {
  const flowId = currentMountFlow.value?.flowID
  if (!flowId) {
    return
  }

  const containerID = Number(mountForm.containerIDText)
  if (!Number.isInteger(containerID) || containerID <= 0) {
    ElMessage.warning('挂载目标 ID 需要是正整数')
    return
  }

  const mountConfig = parseJsonObject(mountForm.mountConfigText, '挂载配置必须是合法 JSON 对象')
  if (!mountConfig) {
    return
  }

  const payload: FlowMount = {
    flowID: flowId,
    containerType: mountForm.containerType,
    containerID,
    mountConfig,
  }

  try {
    await createFlowMount(payload)
    ElMessage.success('挂载创建成功')
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
        <p class="page-subtitle">管理流程定义、步骤配置与挂载目标</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">新建流程</el-button>
    </div>

    <el-table :data="flows" v-loading="loading" style="width: 100%">
      <el-table-column prop="flowID" label="ID" width="90" />
      <el-table-column prop="flowName" label="流程名称" min-width="180" />
      <el-table-column prop="flowDescription" label="描述" min-width="220" />
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

    <el-dialog v-model="dialogVisible" :title="editingFlowId ? '编辑流程' : '新建流程'" width="640px">
      <el-form label-position="top">
        <el-form-item label="流程名称" required>
          <el-input v-model="form.flowName" />
        </el-form-item>
        <el-form-item label="流程描述">
          <el-input v-model="form.flowDescription" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="扩展配置(JSON，可选)">
          <el-input v-model="form.extensionDataText" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveFlow">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="stepDrawerVisible"
      size="760px"
      :title="`步骤编辑 - ${currentStepFlow?.flowName || '未命名流程'}`"
    >
      <el-alert type="info" :closable="false" show-icon>
        步骤编辑采用 JSON 数组格式，每项可包含 stepOrder、stepName、scriptID、stepConfig。
      </el-alert>
      <el-input
        v-model="stepEditorText"
        type="textarea"
        :rows="18"
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
      size="760px"
      :title="`挂载管理 - ${currentMountFlow?.flowName || '未命名流程'}`"
    >
      <div class="mount-form">
        <el-form label-position="top" inline>
          <el-form-item label="挂载类型">
            <el-select v-model="mountForm.containerType" style="width: 140px">
              <el-option label="赛道" value="track" />
              <el-option label="赛事" value="contest" />
              <el-option label="作品" value="work" />
            </el-select>
          </el-form-item>
          <el-form-item label="目标 ID">
            <el-input v-model="mountForm.containerIDText" placeholder="请输入正整数 ID" style="width: 180px" />
          </el-form-item>
          <el-form-item label="挂载配置(JSON，可选)">
            <el-input v-model="mountForm.mountConfigText" type="textarea" :rows="3" style="width: 320px" />
          </el-form-item>
          <el-form-item class="mount-action">
            <el-button type="primary" @click="addMount">新增挂载</el-button>
          </el-form-item>
        </el-form>
      </div>

      <el-table :data="mounts" v-loading="mountLoading" style="width: 100%">
        <el-table-column prop="mountID" label="挂载ID" width="100" />
        <el-table-column prop="containerType" label="类型" width="120" />
        <el-table-column prop="containerID" label="目标ID" width="120" />
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button link type="danger" @click="deleteMount(scope.row.mountID)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-alert v-if="scripts.length === 0" type="warning" :closable="false" class="hint">
      当前未发现可用脚本，建议先在“脚本管理”创建脚本后再配置流程步骤。
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
  margin: 10px 0 14px;
}

.mount-action {
  margin-top: 26px;
}

.hint {
  margin-top: 14px;
}
</style>
