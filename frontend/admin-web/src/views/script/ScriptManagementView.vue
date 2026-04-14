<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  activateScriptVersion,
  createScriptDefinition,
  fetchScriptDefinitions,
  fetchScriptVersions,
  updateScriptDefinition,
  updateScriptDefinitionStatus,
  uploadScriptVersion,
} from '@/services/repositories/scriptRepository'
import type { ScriptDefinition, ScriptVersion } from '@/types/api'

const loading = ref(false)
const scripts = ref<ScriptDefinition[]>([])

const scriptKeyPattern = /^[a-zA-Z0-9_-]+$/
const interpreterOptions = [
  { label: 'Python 3', value: 'python3' },
  { label: 'Python (兼容)', value: 'python' },
  { label: 'Bash', value: 'bash' },
  { label: 'Shell (sh)', value: 'sh' },
  { label: 'Node.js', value: 'node' },
  { label: '内置 Go 执行器', value: 'builtin_go' },
]

const dialogVisible = ref(false)
const editingScriptId = ref<number | null>(null)
const form = reactive({
  scriptName: '',
  scriptKey: '',
  scriptDescription: '',
  interpreter: 'python3',
  metaText: '{\n  "runtime": "python3"\n}',
})

const versionDrawerVisible = ref(false)
const versionLoading = ref(false)
const versions = ref<ScriptVersion[]>([])
const currentScript = ref<ScriptDefinition | null>(null)
const uploadFile = ref<File | null>(null)
const uploading = ref(false)

const currentScriptName = computed(() => currentScript.value?.scriptName || '未命名脚本')
const currentScriptKey = computed(() => currentScript.value?.scriptKey || '-')
const currentActiveVersion = computed(() => versions.value.find((item) => item.isActive) || null)
const currentStoragePattern = computed(() => {
  if (!currentScript.value?.scriptKey) {
    return 'scripts/{scriptKey}/v{版本号}/'
  }
  return `scripts/${currentScript.value.scriptKey}/v{版本号}/`
})

function normalizeScriptDescription(script: ScriptDefinition) {
  return script.description || script.scriptDescription || ''
}

function normalizeScriptMeta(script: ScriptDefinition) {
  return script.meta || script.extensionData || {}
}

function getInterpreterLabel(interpreter: string | undefined) {
  const found = interpreterOptions.find((item) => item.value === interpreter)
  return found?.label || interpreter || '-'
}

function getExecutionFolderHint(script: ScriptDefinition) {
  if (!script.scriptKey) {
    return '-'
  }
  return `scripts/${script.scriptKey}/v{n}/`
}

function formatVersionName(version: ScriptVersion) {
  return version.versionName || version.fileName || (version.versionNum ? `v${version.versionNum}` : `#${version.versionID}`)
}

async function loadScripts() {
  loading.value = true
  try {
    scripts.value = await fetchScriptDefinitions()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '脚本列表加载失败')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, {
    scriptName: '',
    scriptKey: '',
    scriptDescription: '',
    interpreter: 'python3',
    metaText: '{\n  "runtime": "python3"\n}',
  })
}

function openCreateDialog() {
  editingScriptId.value = null
  resetForm()
  dialogVisible.value = true
}

function openEditDialog(script: ScriptDefinition) {
  const nextInterpreter = interpreterOptions.some((item) => item.value === script.interpreter)
    ? (script.interpreter as string)
    : 'python3'

  editingScriptId.value = script.scriptID || null
  Object.assign(form, {
    scriptName: script.scriptName,
    scriptKey: script.scriptKey || '',
    scriptDescription: normalizeScriptDescription(script),
    interpreter: nextInterpreter,
    metaText: JSON.stringify(normalizeScriptMeta(script), null, 2),
  })
  dialogVisible.value = true
}

function parseMeta() {
  try {
    const parsed = form.metaText.trim() ? JSON.parse(form.metaText) : {}
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error('invalid meta')
    }
    return parsed as Record<string, unknown>
  } catch {
    ElMessage.error('脚本元数据必须是合法 JSON 对象')
    return null
  }
}

function validateScriptKey(scriptKey: string) {
  if (!scriptKey) {
    ElMessage.warning('请填写脚本系统标识(scriptKey)')
    return false
  }
  if (!scriptKeyPattern.test(scriptKey)) {
    ElMessage.warning('脚本系统标识仅支持字母、数字、下划线和连字符')
    return false
  }
  return true
}

async function saveScript() {
  const scriptName = form.scriptName.trim()
  const scriptKey = form.scriptKey.trim()
  const interpreter = form.interpreter.trim()

  if (!scriptName) {
    ElMessage.warning('请填写显示名称(scriptName)')
    return
  }
  if (!validateScriptKey(scriptKey)) {
    return
  }
  if (!interpreterOptions.some((item) => item.value === interpreter)) {
    ElMessage.warning('请选择受支持的解释器')
    return
  }

  const meta = parseMeta()
  if (!meta) {
    return
  }

  const payload: ScriptDefinition = {
    scriptName,
    scriptKey,
    description: form.scriptDescription.trim(),
    scriptDescription: form.scriptDescription.trim(),
    interpreter,
    meta,
    extensionData: meta,
  }

  try {
    if (editingScriptId.value) {
      await updateScriptDefinition(editingScriptId.value, payload)
      ElMessage.success('脚本定义更新成功')
    } else {
      await createScriptDefinition(payload)
      ElMessage.success('脚本定义创建成功')
    }
    dialogVisible.value = false
    await loadScripts()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存脚本定义失败')
  }
}

async function changeScriptStatus(script: ScriptDefinition, isEnabled: boolean) {
  if (!script.scriptID) {
    return
  }

  try {
    await updateScriptDefinitionStatus(script.scriptID, { isEnabled })
    script.isEnabled = isEnabled
    ElMessage.success('脚本状态已更新')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '脚本状态更新失败')
  }
}

function handleScriptStatusChange(script: ScriptDefinition, value: unknown) {
  void changeScriptStatus(script, Boolean(value))
}

async function loadVersions(scriptId: number) {
  versionLoading.value = true
  try {
    versions.value = await fetchScriptVersions(scriptId)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '版本列表加载失败')
  } finally {
    versionLoading.value = false
  }
}

async function openVersionDrawer(script: ScriptDefinition) {
  if (!script.scriptID) {
    ElMessage.warning('脚本 ID 不存在，无法管理版本')
    return
  }

  currentScript.value = script
  versions.value = []
  uploadFile.value = null
  versionDrawerVisible.value = true
  await loadVersions(script.scriptID)
}

function onFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  uploadFile.value = target.files?.[0] || null
}

async function submitVersionUpload() {
  const scriptId = currentScript.value?.scriptID
  if (!scriptId) {
    return
  }
  if (!uploadFile.value) {
    ElMessage.warning('请先选择上传文件')
    return
  }

  uploading.value = true
  try {
    await uploadScriptVersion(scriptId, uploadFile.value)
    ElMessage.success('脚本版本上传成功')
    uploadFile.value = null
    await loadVersions(scriptId)
    await loadScripts()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '脚本版本上传失败')
  } finally {
    uploading.value = false
  }
}

async function activateVersion(version: ScriptVersion) {
  const scriptId = currentScript.value?.scriptID
  const versionId = version.versionID
  if (!scriptId || !versionId) {
    return
  }

  try {
    await activateScriptVersion(scriptId, versionId)
    ElMessage.success('版本激活成功')
    await loadVersions(scriptId)
    await loadScripts()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '版本激活失败')
  }
}

onMounted(loadScripts)
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">脚本库</h1>
        <p class="page-subtitle">以“定义 -> 版本快照 -> 执行文件路径”管理脚本资产</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">新建脚本定义</el-button>
    </div>

    <div class="relation-card">
      <div class="relation-item">
        <strong>定义层</strong>
        <p>scriptKey / scriptName / interpreter。用于标识脚本身份与执行环境。</p>
      </div>
      <div class="relation-item">
        <strong>版本层</strong>
        <p>每次上传都会形成版本快照，可激活一个版本作为默认执行版本。</p>
      </div>
      <div class="relation-item">
        <strong>文件层</strong>
        <p>流程步骤最终执行的是版本的 relativePath，不是你本地临时文件名。</p>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      关系说明：你本地准备脚本文件 -> 在这里上传形成版本快照 -> 流程步骤引用该版本（锁定或跟随激活版本）执行。
    </el-alert>

    <el-table :data="scripts" v-loading="loading" style="width: 100%">
      <el-table-column prop="scriptID" label="ID" width="90" />
      <el-table-column prop="scriptKey" label="系统标识(scriptKey)" min-width="180" />
      <el-table-column prop="scriptName" label="显示名称(scriptName)" min-width="180" />
      <el-table-column label="执行环境" width="160">
        <template #default="scope">
          {{ getInterpreterLabel(scope.row.interpreter) }}
        </template>
      </el-table-column>
      <el-table-column label="激活版本" width="130">
        <template #default="scope">
          <el-tag v-if="scope.row.activeVersionID" type="success">#{{ scope.row.activeVersionID }}</el-tag>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="执行文件目录" min-width="220">
        <template #default="scope">
          {{ getExecutionFolderHint(scope.row) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="scope">
          <el-switch
            :model-value="Boolean(scope.row.isEnabled)"
            @change="handleScriptStatusChange(scope.row, $event)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openEditDialog(scope.row)">编辑</el-button>
            <el-button link type="success" @click="openVersionDrawer(scope.row)">版本</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingScriptId ? '编辑脚本定义' : '新建脚本定义'" width="680px">
      <el-form label-position="top">
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="显示名称(scriptName)" required>
              <el-input v-model="form.scriptName" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="系统标识(scriptKey)" required>
              <el-input v-model="form.scriptKey" placeholder="仅支持字母、数字、下划线和连字符" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="执行环境(interpreter)">
              <el-select v-model="form.interpreter" placeholder="请选择解释器" style="width: 100%">
                <el-option v-for="item in interpreterOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="描述">
              <el-input v-model="form.scriptDescription" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="元数据(meta, JSON)">
          <el-input v-model="form.metaText" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveScript">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="versionDrawerVisible" size="860px" :title="`版本快照 - ${currentScriptName}`">
      <el-alert type="info" :closable="false" show-icon>
        上传后的文件会被系统存储并版本化；流程执行时读取的是 relativePath 指向的文件。
      </el-alert>

      <el-descriptions border :column="2" size="small" class="script-summary">
        <el-descriptions-item label="系统标识(scriptKey)">{{ currentScriptKey }}</el-descriptions-item>
        <el-descriptions-item label="执行环境">{{ getInterpreterLabel(currentScript?.interpreter) }}</el-descriptions-item>
        <el-descriptions-item label="默认存储目录">{{ currentStoragePattern }}</el-descriptions-item>
        <el-descriptions-item label="当前激活版本路径">{{ currentActiveVersion?.relativePath || '-' }}</el-descriptions-item>
      </el-descriptions>

      <div class="version-toolbar">
        <input type="file" @change="onFileChange" />
        <el-button type="primary" :loading="uploading" @click="submitVersionUpload">上传新版本</el-button>
      </div>

      <el-table :data="versions" v-loading="versionLoading" style="width: 100%">
        <el-table-column prop="versionID" label="版本ID" width="100" />
        <el-table-column prop="versionNum" label="序号" width="90" />
        <el-table-column label="版本名/文件名" min-width="220">
          <template #default="scope">
            {{ formatVersionName(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column prop="relativePath" label="部署文件路径(relativePath)" min-width="260" show-overflow-tooltip />
        <el-table-column prop="checksum" label="校验值" min-width="180" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="上传时间" min-width="170" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.isActive ? 'success' : 'info'">{{ scope.row.isActive ? '已激活' : '未激活' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button link type="primary" :disabled="scope.row.isActive" @click="activateVersion(scope.row)">激活</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>
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

.relation-card {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin: 10px 0 12px;
}

.relation-item {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  padding: 10px 12px;
  background: #fff;
}

.relation-item p {
  margin: 6px 0 0;
  color: #4b5563;
  font-size: 13px;
  line-height: 1.45;
}

.script-summary {
  margin: 12px 0;
}

.version-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

@media (max-width: 900px) {
  .relation-card {
    grid-template-columns: 1fr;
  }
}
</style>
