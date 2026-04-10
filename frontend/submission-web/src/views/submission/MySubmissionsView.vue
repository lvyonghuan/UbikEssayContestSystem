<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import { fetchContests } from '@/services/repositories/contestRepository'
import {
  downloadSubmissionFile,
  fetchMySubmissions,
  removeSubmission,
} from '@/services/repositories/submissionRepository'
import type { Work } from '@/types/api'
import { calculateSHA256FromArrayBuffer } from '@/utils/hash'

const router = useRouter()
const loading = ref(false)
const submissions = ref<Work[]>([])
const endedContestIDs = ref<Set<number>>(new Set())
const contestNameByID = ref<Map<number, string>>(new Map())
const downloadingWorkIDs = ref<Set<number>>(new Set())

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

const filters = reactive({
  keyword: '',
  trackID: undefined as number | undefined,
})

const filteredSubmissions = computed(() => {
  return submissions.value.filter((work) => {
    const titleMatch =
      !filters.keyword ||
      (work.workTitle || '').toLowerCase().includes(filters.keyword.trim().toLowerCase())
    const trackMatch = !filters.trackID || work.trackID === filters.trackID
    return titleMatch && trackMatch
  })
})

const trackOptions = computed(() => {
  const seen = new Set<number>()
  const options: Array<{ label: string; value: number }> = []

  for (const work of submissions.value) {
    if (typeof work.trackID !== 'number' || seen.has(work.trackID)) {
      continue
    }
    seen.add(work.trackID)
    options.push({
      label: work.trackName || `赛道 ${work.trackID}`,
      value: work.trackID,
    })
  }

  return options
})

function formatDate(value?: string) {
  if (!value) {
    return '-'
  }
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value
}

function timelineText(work: Work) {
  const submittedAt = work.workInfos?.submittedAt
  const updatedAt = work.workInfos?.updatedAt

  if (typeof updatedAt === 'string' && updatedAt.trim()) {
    return formatDate(updatedAt)
  }
  if (typeof submittedAt === 'string' && submittedAt.trim()) {
    return formatDate(submittedAt)
  }
  return '-'
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

function parseContestID(work: Work) {
  const rawContestID = work.workInfos?.contestID
  if (typeof rawContestID === 'number' && Number.isInteger(rawContestID) && rawContestID > 0) {
    return rawContestID
  }
  if (typeof rawContestID === 'string') {
    const parsed = Number(rawContestID)
    if (Number.isInteger(parsed) && parsed > 0) {
      return parsed
    }
  }
  return null
}

function contestNameText(work: Work) {
  const rawContestName = work.workInfos?.contestName
  if (typeof rawContestName === 'string' && rawContestName.trim()) {
    return rawContestName.trim()
  }

  const contestID = parseContestID(work)
  if (!contestID) {
    return '-'
  }

  return contestNameByID.value.get(contestID) || `比赛 ${contestID}`
}

function isReadonlyWork(work: Work) {
  const contestID = parseContestID(work)
  if (!contestID) {
    return false
  }
  return endedContestIDs.value.has(contestID)
}

function isDownloadingWork(workID?: number) {
  return typeof workID === 'number' && downloadingWorkIDs.value.has(workID)
}

function setDownloadingWorkState(workID: number, downloading: boolean) {
  const next = new Set(downloadingWorkIDs.value)
  if (downloading) {
    next.add(workID)
  } else {
    next.delete(workID)
  }
  downloadingWorkIDs.value = next
}

function saveBlobAsFile(blob: Blob, fileName: string) {
  const objectURL = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectURL
  anchor.download = fileName
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(objectURL)
}

async function loadSubmissions() {
  loading.value = true
  try {
    const [workList, contests] = await Promise.all([
      fetchMySubmissions(),
      fetchContests().catch(() => []),
    ])

    const now = dayjs()
    const endedIDs = new Set<number>()
    const contestNameMap = new Map<number, string>()
    for (const contest of contests) {
      if (typeof contest.contestID !== 'number') {
        continue
      }

      if (contest.contestName && contest.contestName.trim()) {
        contestNameMap.set(contest.contestID, contest.contestName.trim())
      }

      const end = dayjs(contest.contestEndDate)
      if (end.isValid() && now.isAfter(end)) {
        endedIDs.add(contest.contestID)
      }
    }

    endedContestIDs.value = endedIDs
    contestNameByID.value = contestNameMap
    submissions.value = workList
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '稿件列表加载失败')
    submissions.value = []
    endedContestIDs.value = new Set()
    contestNameByID.value = new Map()
  } finally {
    loading.value = false
  }
}

function openEdit(work: Work) {
  if (isReadonlyWork(work)) {
    ElMessage.warning('所属比赛已结束，稿件仅支持查看，不能修改')
    return
  }

  const workID = work.workID
  if (!workID) {
    ElMessage.warning('稿件 ID 缺失，无法编辑')
    return
  }
  router.push({ name: 'edit-submission', params: { workId: workID } })
}

async function onDelete(work: Work) {
  if (isReadonlyWork(work)) {
    ElMessage.warning('所属比赛已结束，稿件仅支持查看，不能删除')
    return
  }

  const workID = work.workID
  if (!workID) {
    ElMessage.warning('稿件 ID 缺失，无法删除')
    return
  }

  try {
    await ElMessageBox.confirm('删除后不可恢复，确认继续？', '删除确认', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  try {
    await removeSubmission(workID)
    ElMessage.success('稿件已删除')
    await loadSubmissions()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '删除失败，请稍后重试')
  }
}

async function onDownload(work: Work) {
  const workID = work.workID
  if (!workID) {
    ElMessage.warning('稿件 ID 缺失，无法下载')
    return
  }
  if (isDownloadingWork(workID)) {
    return
  }

  setDownloadingWorkState(workID, true)
  try {
    const { fileBlob, fileName, fileHashSHA256 } = await downloadSubmissionFile(workID)
    const downloadedHash = await calculateSHA256FromArrayBuffer(await fileBlob.arrayBuffer())

    if (downloadedHash !== fileHashSHA256) {
      ElMessage.error('下载失败：文件完整性校验未通过，请重试')
      return
    }

    saveBlobAsFile(fileBlob, fileName || `submission-${workID}.docx`)
    ElMessage.success('稿件下载成功')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '下载失败，请稍后重试')
  } finally {
    setDownloadingWorkState(workID, false)
  }
}

onMounted(loadSubmissions)
</script>

<template>
  <section class="page-card submissions-page">
    <div class="header-row">
      <div>
        <h1 class="page-title">我的稿件</h1>
        <p class="page-subtitle">查看已投递稿件，已结束比赛仅支持查看</p>
      </div>
      <el-button type="primary" plain @click="router.push({ name: 'dashboard' })">去看板投稿</el-button>
    </div>

    <el-space wrap class="filters">
      <el-input v-model="filters.keyword" clearable placeholder="按稿件标题搜索" style="width: 220px" />
      <el-select v-model="filters.trackID" clearable placeholder="按赛道筛选" style="width: 200px">
        <el-option v-for="item in trackOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
    </el-space>

    <el-table :data="filteredSubmissions" v-loading="loading" empty-text="暂无投稿记录" style="width: 100%">
      <el-table-column prop="workID" label="稿件ID" width="100" />
      <el-table-column prop="workTitle" label="稿件标题" min-width="220" />
      <el-table-column label="比赛" min-width="180">
        <template #default="scope">
          {{ contestNameText(scope.row) }}
        </template>
      </el-table-column>
      <el-table-column label="赛道" min-width="160">
        <template #default="scope">
          {{ scope.row.trackName || `赛道 ${scope.row.trackID || '-'}` }}
        </template>
      </el-table-column>
      <el-table-column label="更新时间" min-width="180">
        <template #default="scope">
          {{ timelineText(scope.row) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="130">
        <template #default="scope">
          <el-tag :type="workStatusTagType(scope.row)">{{ workStatusText(scope.row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="scope">
          <el-space>
            <el-button
              link
              type="primary"
              :loading="isDownloadingWork(scope.row.workID)"
              @click="onDownload(scope.row)"
            >
              下载
            </el-button>
            <template v-if="!isReadonlyWork(scope.row)">
              <el-button link type="primary" @click="openEdit(scope.row)">修改</el-button>
              <el-button link type="danger" @click="onDelete(scope.row)">删除</el-button>
            </template>
            <el-text v-else type="info">仅可查看</el-text>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<style scoped>
.submissions-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.filters {
  display: flex;
}
</style>
