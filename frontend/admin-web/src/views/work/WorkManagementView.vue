<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  downloadWorkFile,
  fetchWorks,
  fetchWorkByID,
  removeWork,
} from '@/services/repositories/workRepository'
import type { Work } from '@/types/api'

const loading = ref(false)
const works = ref<Work[]>([])
const listTitle = ref('请使用筛选条件查询作品')

const filters = reactive({
  authorName: '',
  trackIdText: '',
  workTitle: '',
  offsetText: '0',
  limitText: '20',
})

const detailDrawerVisible = ref(false)
const detailLoading = ref(false)
const currentWork = ref<Work | null>(null)

const statusTextMap: Record<string, string> = {
  submission_success: '投稿成功',
  pending: '待审核',
  reviewing: '评审中',
  approved: '已通过',
  rejected: '已驳回',
}

const statusTagTypeMap: Record<string, 'success' | 'warning' | 'danger' | 'info'> = {
  submission_success: 'success',
  pending: 'warning',
  reviewing: 'info',
  approved: 'success',
  rejected: 'danger',
}

function parsePositiveInteger(text: string, label: string, allowEmpty = false) {
  if (allowEmpty && !text.trim()) {
    return null
  }
  const parsed = Number(text)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    ElMessage.warning(`${label} 需要是正整数`)
    return null
  }
  return parsed
}

function parseNonNegativeInteger(text: string, label: string) {
  const parsed = Number(text)
  if (!Number.isInteger(parsed) || parsed < 0) {
    ElMessage.warning(`${label} 需要是非负整数`)
    return null
  }
  return parsed
}

function normalizeStatusValue(value: unknown) {
  if (typeof value !== 'string') {
    return ''
  }
  return value.trim()
}

function resolveWorkStatus(work: Work) {
  const direct = normalizeStatusValue(work.workStatus)
  if (direct) {
    return direct
  }

  const fromInfosWorkStatus = normalizeStatusValue(work.workInfos?.['workStatus'])
  if (fromInfosWorkStatus) {
    return fromInfosWorkStatus
  }

  const fromInfosSnakeCase = normalizeStatusValue(work.workInfos?.['work_status'])
  if (fromInfosSnakeCase) {
    return fromInfosSnakeCase
  }

  return normalizeStatusValue(work.workInfos?.['status'])
}

function workStatusText(work: Work) {
  const status = resolveWorkStatus(work)
  if (!status) {
    return '-'
  }
  return statusTextMap[status] || status
}

function workStatusTagType(work: Work) {
  const status = resolveWorkStatus(work)
  if (!status) {
    return 'info'
  }
  return statusTagTypeMap[status] || 'info'
}

async function queryWorks() {
  const offset = parseNonNegativeInteger(filters.offsetText, '偏移量')
  const limit = parsePositiveInteger(filters.limitText, '每页条数')
  let trackID: number | undefined

  const trackIdText = filters.trackIdText.trim()
  if (trackIdText) {
    const parsedTrackID = parsePositiveInteger(trackIdText, '赛道 ID')
    if (parsedTrackID === null) {
      return
    }
    trackID = parsedTrackID
  }

  if (offset === null || limit === null) {
    return
  }

  loading.value = true
  try {
    works.value = await fetchWorks({
      authorName: filters.authorName.trim() || undefined,
      workTitle: filters.workTitle.trim() || undefined,
      trackID,
      offset,
      limit,
    })
    listTitle.value = `查询结果（${works.value.length}）`
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作品查询失败')
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  Object.assign(filters, {
    authorName: '',
    trackIdText: '',
    workTitle: '',
    offsetText: '0',
    limitText: '20',
  })
}

async function openDetail(workId: number | undefined) {
  if (!workId) {
    ElMessage.warning('作品 ID 不存在')
    return
  }

  detailLoading.value = true
  detailDrawerVisible.value = true
  try {
    currentWork.value = await fetchWorkByID(workId)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作品详情加载失败')
  } finally {
    detailLoading.value = false
  }
}

async function deleteWork(workId: number | undefined) {
  if (!workId) {
    return
  }

  try {
    await ElMessageBox.confirm('删除作品将同时删除存储文件，确认继续吗？', '删除作品', {
      type: 'warning',
    })
    await removeWork(workId)
    works.value = works.value.filter((item) => item.workID !== workId)
    ElMessage.success('作品已删除')
  } catch {
    // canceled
  }
}

async function downloadFile(workId: number | undefined) {
  if (!workId) {
    return
  }

  try {
    const blob = await downloadWorkFile(workId)
    const objectUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = objectUrl
    link.download = `work-${workId}.bin`
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(objectUrl)
    ElMessage.success('作品文件下载成功')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作品文件下载失败')
  }
}
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">作品管理</h1>
        <p class="page-subtitle">支持按标题、作者名、赛道联合查询，并提供详情、删除、文件下载</p>
      </div>
    </div>

    <div class="filters-row">
      <el-space wrap>
        <el-input v-model="filters.workTitle" placeholder="作品标题" style="width: 180px" clearable />
        <el-input v-model="filters.authorName" placeholder="作者名" style="width: 180px" clearable />
        <el-input v-model="filters.trackIdText" placeholder="赛道 ID（可选）" style="width: 180px" clearable />
        <el-input v-model="filters.offsetText" placeholder="偏移量" style="width: 120px" />
        <el-input v-model="filters.limitText" placeholder="每页条数" style="width: 120px" />
        <el-button type="primary" @click="queryWorks">查询</el-button>
        <el-button @click="resetFilters">重置</el-button>
      </el-space>
    </div>

    <p class="list-title">{{ listTitle }}</p>

    <el-table :data="works" v-loading="loading" style="width: 100%">
      <el-table-column prop="workTitle" label="标题" min-width="220" />
      <el-table-column prop="authorName" label="作者名" min-width="140">
        <template #default="scope">
          {{ scope.row.authorName || '待后端返回 authorName' }}
        </template>
      </el-table-column>
      <el-table-column prop="trackName" label="赛道名" min-width="140">
        <template #default="scope">
          {{ scope.row.trackName || '待后端返回 trackName' }}
        </template>
      </el-table-column>
      <el-table-column prop="workID" label="作品ID" width="100" />
      <el-table-column label="状态" width="130">
        <template #default="scope">
          <el-tag :type="workStatusTagType(scope.row)">{{ workStatusText(scope.row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openDetail(scope.row.workID)">详情</el-button>
            <el-button link type="success" @click="downloadFile(scope.row.workID)">下载文件</el-button>
            <el-button link type="danger" @click="deleteWork(scope.row.workID)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="detailDrawerVisible" size="560px" title="作品详情">
      <el-skeleton v-if="detailLoading" :rows="6" animated />
      <template v-else>
        <el-descriptions v-if="currentWork" border :column="1" size="small">
          <el-descriptions-item label="标题">{{ currentWork.workTitle || '-' }}</el-descriptions-item>
          <el-descriptions-item label="作者名">{{ currentWork.authorName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="赛道名">{{ currentWork.trackName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="作品ID">{{ currentWork.workID || '-' }}</el-descriptions-item>
          <el-descriptions-item label="作者ID">{{ currentWork.authorID || '-' }}</el-descriptions-item>
          <el-descriptions-item label="赛道ID">{{ currentWork.trackID || '-' }}</el-descriptions-item>
          <el-descriptions-item label="扩展信息">
            <pre class="work-json">{{ JSON.stringify(currentWork.workInfos || {}, null, 2) }}</pre>
          </el-descriptions-item>
        </el-descriptions>
      </template>
    </el-drawer>
  </section>
</template>

<style scoped>
.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.filters-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.list-title {
  margin: 0 0 12px;
  color: var(--text-secondary, #909399);
  font-size: 13px;
}

.work-json {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
