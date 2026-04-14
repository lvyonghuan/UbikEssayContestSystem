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

function normalizeScriptDescription(script: ScriptDefinition) {
  return script.description || script.scriptDescription || ''
}

function normalizeScriptMeta(script: ScriptDefinition) {
  return script.meta || script.extensionData || {}
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
    ElMessage.warning('请填写脚本键(scriptKey)')
    return false
  }
  if (!scriptKeyPattern.test(scriptKey)) {
    ElMessage.warning('脚本键仅支持字母、数字、下划线和连字符')
    return false
  }
  return true
}

async function saveScript() {
  const scriptName = form.scriptName.trim()
  const scriptKey = form.scriptKey.trim()
  const interpreter = form.interpreter.trim()

  if (!scriptName) {
    ElMessage.warning('请填写脚本名称')
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
      ElMessage.success('脚本更新成功')
    } else {
      await createScriptDefinition(payload)
      ElMessage.success('脚本创建成功')
    }
    dialogVisible.value = false
    await loadScripts()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存脚本失败')
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
        <h1 class="page-title">脚本管理</h1>
        <p class="page-subtitle">维护脚本定义、解释器、版本与激活状态</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">新建脚本</el-button>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      新手建议：scriptKey 建议按业务命名（如 submission_word_count），解释器请从下拉中选择受支持项。
    </el-alert>

    <el-table :data="scripts" v-loading="loading" style="width: 100%">
      <el-table-column prop="scriptID" label="ID" width="90" />
      <el-table-column prop="scriptKey" label="脚本键" min-width="140" />
      <el-table-column prop="scriptName" label="脚本名称" min-width="180" />
      <el-table-column prop="interpreter" label="解释器" width="120" />
      <el-table-column label="描述" min-width="220">
        <template #default="scope">
          {{ scope.row.description || scope.row.scriptDescription || '-' }}
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
      <el-table-column label="激活版本" width="120">
        <template #default="scope">
          {{ scope.row.activeVersionID || '-' }}
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

    <el-dialog v-model="dialogVisible" :title="editingScriptId ? '编辑脚本' : '新建脚本'" width="640px">
      <el-form label-position="top">
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="脚本名称" required>
              <el-input v-model="form.scriptName" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="脚本键(scriptKey)" required>
              <el-input v-model="form.scriptKey" placeholder="仅支持字母、数字、下划线和连字符" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :xs="24" :md="12">
            <el-form-item label="解释器">
              <el-select v-model="form.interpreter" placeholder="请选择解释器" style="width: 100%">
                <el-option v-for="item in interpreterOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-form-item label="脚本描述">
              <el-input v-model="form.scriptDescription" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="脚本元数据(meta，JSON)">
          <el-input v-model="form.metaText" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveScript">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="versionDrawerVisible" size="720px" :title="`版本管理 - ${currentScriptName}`">
      <div class="version-toolbar">
        <input type="file" @change="onFileChange" />
        <el-button type="primary" :loading="uploading" @click="submitVersionUpload">上传新版本</el-button>
      </div>

      <el-table :data="versions" v-loading="versionLoading" style="width: 100%">
        <el-table-column prop="versionID" label="版本ID" width="100" />
        <el-table-column prop="versionNum" label="序号" width="90" />
        <el-table-column label="版本名/文件名" min-width="200">
          <template #default="scope">
            {{ scope.row.versionName || scope.row.fileName || (scope.row.versionNum ? `v${scope.row.versionNum}` : `v${scope.row.versionID}`) }}
          </template>
        </el-table-column>
        <el-table-column prop="checksum" label="校验值" min-width="170" />
        <el-table-column prop="createdBy" label="上传人" width="100" />
        <el-table-column prop="relativePath" label="存储路径" min-width="180" />
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

.version-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
</style>
