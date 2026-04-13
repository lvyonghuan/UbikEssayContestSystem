<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { fetchContests } from '@/services/repositories/contestRepository'
import {
  exportTrackReviewExcel,
  fetchContestJudgeProgressStats,
  fetchTrackReviewRanking,
  fetchTrackStatuses,
  fetchWorkReviewResults,
  fetchWorkReviewStatus,
  regenerateContestReviewResults,
  regenerateWorkReviewResults,
} from '@/services/repositories/judgeRepository'
import { fetchTracks } from '@/services/repositories/trackRepository'
import type { Contest, JudgeProgressStat, ReviewResult, Track, TrackRankItem, WorkReviewStatus } from '@/types/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const workInspectLoading = ref(false)
const exportLoading = ref(false)
const regenerateContestLoading = ref(false)

const contests = ref<Contest[]>([])
const tracks = ref<Track[]>([])

const selectedContestId = ref<number | null>(null)
const selectedTrackId = ref<number | null>(null)

const judgeProgress = ref<JudgeProgressStat[]>([])
const trackStatuses = ref<string[]>([])
const trackRanking = ref<TrackRankItem[]>([])

const workIdText = ref('')
const currentWorkStatus = ref<WorkReviewStatus | null>(null)
const currentWorkResults = ref<ReviewResult[]>([])

const statusTextMap: Record<string, string> = {
  submission_success: '投稿成功',
  pending: '待审核',
  reviewing: '评审中',
  approved: '已通过',
  rejected: '已驳回',
}

const selectedContestName = computed(() => {
  const matched = contests.value.find((item) => item.contestID === selectedContestId.value)
  return matched?.contestName || '-'
})

const selectedTrackName = computed(() => {
  const matched = tracks.value.find((item) => item.trackID === selectedTrackId.value)
  return matched?.trackName || '-'
})

const contestOptions = computed(() => {
  return contests.value
    .filter((item): item is Contest & { contestID: number } => typeof item.contestID === 'number')
    .map((item) => ({
      value: item.contestID,
      label: item.contestName,
    }))
})

const trackOptions = computed(() => {
  return tracks.value
    .filter((item): item is Track & { trackID: number } => typeof item.trackID === 'number')
    .map((item) => ({
      value: item.trackID,
      label: item.trackName || `赛道 ${item.trackID}`,
    }))
})

const completedJudgeCount = computed(() => {
  return judgeProgress.value.filter((item) => item.assignedCount > 0 && item.submittedCount >= item.assignedCount).length
})

function parsePositiveInteger(text: string, label: string) {
  const parsed = Number(text.trim())
  if (!Number.isInteger(parsed) || parsed <= 0) {
    ElMessage.warning(`${label} 需要是正整数`)
    return null
  }
  return parsed
}

function toNumber(value: unknown) {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : 0
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}

function formatStatusLabel(status: string) {
  return statusTextMap[status] || status
}

function formatPercent(rate: number) {
  return `${Math.round(rate * 100)}%`
}

function finalScoreFromResult(result: ReviewResult) {
  const score = toNumber(result.reviews?.finalScore)
  return score.toFixed(2)
}

function reviewCountFromResult(result: ReviewResult) {
  return Math.round(toNumber(result.reviews?.reviewCount))
}

function generatedAtFromResult(result: ReviewResult) {
  const generatedAt = result.reviews?.generatedAt
  return typeof generatedAt === 'string' && generatedAt.trim() ? generatedAt : '-'
}

async function loadTrackScopedData(trackId: number) {
  const [statuses, ranking] = await Promise.all([
    fetchTrackStatuses(trackId),
    fetchTrackReviewRanking(trackId),
  ])
  trackStatuses.value = statuses
  trackRanking.value = ranking
}

async function loadContestScopedData(contestId: number) {
  const [nextTracks, nextJudgeProgress] = await Promise.all([
    fetchTracks(contestId),
    fetchContestJudgeProgressStats(contestId),
  ])

  tracks.value = nextTracks
  judgeProgress.value = nextJudgeProgress

  const trackIDs = nextTracks
    .map((item) => item.trackID)
    .filter((value): value is number => typeof value === 'number' && value > 0)

  if (trackIDs.length === 0) {
    selectedTrackId.value = null
    trackStatuses.value = []
    trackRanking.value = []
    return
  }

  if (!selectedTrackId.value || !trackIDs.includes(selectedTrackId.value)) {
    selectedTrackId.value = trackIDs[0]
  }

  await loadTrackScopedData(selectedTrackId.value)
}

async function initializePage() {
  loading.value = true
  try {
    const nextContests = await fetchContests()
    contests.value = nextContests

    if (!nextContests.length) {
      selectedContestId.value = null
      selectedTrackId.value = null
      tracks.value = []
      judgeProgress.value = []
      trackStatuses.value = []
      trackRanking.value = []
      return
    }

    const routeContestIdText = String(route.params.contestId || '').trim()
    const routeContestId = Number(routeContestIdText)
    const routeContestValid = Number.isInteger(routeContestId) && routeContestId > 0

    const firstContestId = nextContests[0].contestID || null
    const selectedFromRoute = routeContestValid
      ? nextContests.find((item) => item.contestID === routeContestId)?.contestID || null
      : null

    selectedContestId.value = selectedFromRoute || firstContestId

    if (!selectedContestId.value) {
      return
    }

    await loadContestScopedData(selectedContestId.value)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '评审管理页面加载失败')
  } finally {
    loading.value = false
  }
}

async function handleContestChange(contestId: number) {
  selectedContestId.value = contestId
  selectedTrackId.value = null
  loading.value = true
  try {
    await loadContestScopedData(contestId)
    await router.replace({ name: 'judge-review', params: { contestId } })
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '切换比赛失败')
  } finally {
    loading.value = false
  }
}

async function handleTrackChange(trackId: number) {
  selectedTrackId.value = trackId
  loading.value = true
  try {
    await loadTrackScopedData(trackId)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '切换赛道失败')
  } finally {
    loading.value = false
  }
}

function onContestModelUpdate(value: string | number) {
  void handleContestChange(Number(value))
}

function onTrackModelUpdate(value: string | number) {
  void handleTrackChange(Number(value))
}

async function refreshCurrent() {
  if (!selectedContestId.value) {
    return
  }

  loading.value = true
  try {
    await loadContestScopedData(selectedContestId.value)
    ElMessage.success('评审数据已刷新')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '刷新失败')
  } finally {
    loading.value = false
  }
}

async function exportRankingSheet() {
  if (!selectedTrackId.value) {
    ElMessage.warning('请先选择赛道')
    return
  }

  exportLoading.value = true
  try {
    const { blob, filename } = await exportTrackReviewExcel(selectedTrackId.value)
    const objectUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = objectUrl
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(objectUrl)
    ElMessage.success('导出成功')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '导出失败')
  } finally {
    exportLoading.value = false
  }
}

async function regenerateContestResults() {
  if (!selectedContestId.value) {
    ElMessage.warning('请先选择比赛')
    return
  }

  try {
    await ElMessageBox.confirm(
      '将按当前比赛全部评审事件重新生成结果，确认继续吗？',
      '重算评审结果',
      { type: 'warning' },
    )
  } catch {
    return
  }

  regenerateContestLoading.value = true
  try {
    const result = await regenerateContestReviewResults(selectedContestId.value)
    ElMessage.success(`重算完成，共生成 ${result.generated} 条结果`)
    await refreshCurrent()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '比赛结果重算失败')
  } finally {
    regenerateContestLoading.value = false
  }
}

async function queryWorkReviewData() {
  const workId = parsePositiveInteger(workIdText.value, '作品 ID')
  if (!workId) {
    return
  }

  workInspectLoading.value = true
  try {
    const [status, results] = await Promise.all([
      fetchWorkReviewStatus(workId),
      fetchWorkReviewResults(workId),
    ])
    currentWorkStatus.value = status
    currentWorkResults.value = results
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作品评审数据加载失败')
    currentWorkStatus.value = null
    currentWorkResults.value = []
  } finally {
    workInspectLoading.value = false
  }
}

async function regenerateCurrentWorkResults() {
  const workId = parsePositiveInteger(workIdText.value, '作品 ID')
  if (!workId) {
    return
  }

  workInspectLoading.value = true
  try {
    currentWorkResults.value = await regenerateWorkReviewResults(workId)
    currentWorkStatus.value = await fetchWorkReviewStatus(workId)
    ElMessage.success('作品评审结果已重算')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作品评审重算失败')
  } finally {
    workInspectLoading.value = false
  }
}

function openContestDetail() {
  if (!selectedContestId.value) {
    ElMessage.warning('请先选择比赛')
    return
  }
  router.push({ name: 'contest-detail', params: { contestId: selectedContestId.value } })
}

onMounted(initializePage)
</script>

<template>
  <section class="page-card judge-page" v-loading="loading">
    <div class="header-row">
      <div>
        <h1 class="page-title">评审管理</h1>
        <p class="page-subtitle">对接评审进度、赛道状态、排名导出与结果重算</p>
      </div>
      <el-space wrap>
        <el-button plain @click="openContestDetail">比赛详情</el-button>
        <el-button @click="refreshCurrent">刷新</el-button>
      </el-space>
    </div>

    <section class="filter-row">
      <el-select
        :model-value="selectedContestId"
        placeholder="选择比赛"
        style="width: 260px"
        @update:model-value="onContestModelUpdate"
      >
        <el-option
          v-for="item in contestOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>

      <el-select
        :model-value="selectedTrackId"
        placeholder="选择赛道"
        style="width: 220px"
        :disabled="!trackOptions.length"
        @update:model-value="onTrackModelUpdate"
      >
        <el-option
          v-for="item in trackOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>

      <el-button type="primary" :loading="exportLoading" :disabled="!selectedTrackId" @click="exportRankingSheet">
        导出赛道排名
      </el-button>
      <el-button
        type="warning"
        plain
        :loading="regenerateContestLoading"
        :disabled="!selectedContestId"
        @click="regenerateContestResults"
      >
        重算比赛结果
      </el-button>
    </section>

    <div class="summary-row">
      <article class="summary-card">
        <div class="summary-label">当前比赛</div>
        <div class="summary-value summary-text">{{ selectedContestName }}</div>
      </article>
      <article class="summary-card">
        <div class="summary-label">当前赛道</div>
        <div class="summary-value summary-text">{{ selectedTrackName }}</div>
      </article>
      <article class="summary-card">
        <div class="summary-label">评委人数</div>
        <div class="summary-value">{{ judgeProgress.length }}</div>
      </article>
      <article class="summary-card">
        <div class="summary-label">已完成评委</div>
        <div class="summary-value">{{ completedJudgeCount }}</div>
      </article>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="12">
        <section class="inner-card">
          <div class="section-header">
            <h3 class="inner-title">赛道状态与排名</h3>
            <el-space wrap>
              <el-tag
                v-for="status in trackStatuses"
                :key="status"
                type="info"
                effect="plain"
              >
                {{ formatStatusLabel(status) }}
              </el-tag>
              <span v-if="!trackStatuses.length" class="muted-text">暂无状态数据</span>
            </el-space>
          </div>

          <el-table :data="trackRanking" size="small" empty-text="暂无排名数据">
            <el-table-column label="排名" width="70">
              <template #default="scope">{{ scope.$index + 1 }}</template>
            </el-table-column>
            <el-table-column prop="workTitle" label="作品" min-width="180" />
            <el-table-column prop="authorName" label="作者" min-width="120" />
            <el-table-column prop="finalScore" label="最终分" width="110">
              <template #default="scope">{{ Number(scope.row.finalScore).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="reviewCount" label="评审数" width="90" />
            <el-table-column prop="workID" label="作品ID" width="100" />
          </el-table>
        </section>
      </el-col>

      <el-col :xs="24" :lg="12">
        <section class="inner-card">
          <h3 class="inner-title">评委进度</h3>
          <el-table :data="judgeProgress" size="small" empty-text="暂无评委进度数据">
            <el-table-column prop="judgeName" label="评委" min-width="140" />
            <el-table-column prop="assignedCount" label="分配量" width="90" />
            <el-table-column prop="submittedCount" label="已提交" width="90" />
            <el-table-column label="完成率" width="110">
              <template #default="scope">{{ formatPercent(scope.row.completionRate) }}</template>
            </el-table-column>
          </el-table>
        </section>
      </el-col>
    </el-row>

    <section class="inner-card">
      <div class="section-header">
        <h3 class="inner-title">作品评审详情</h3>
        <el-space wrap>
          <el-input v-model="workIdText" placeholder="作品 ID" style="width: 160px" clearable />
          <el-button type="primary" :loading="workInspectLoading" @click="queryWorkReviewData">查询</el-button>
          <el-button :loading="workInspectLoading" @click="regenerateCurrentWorkResults">重算该作品</el-button>
        </el-space>
      </div>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="12">
          <h4 class="sub-title">事件完成情况</h4>
          <el-table :data="currentWorkStatus?.events || []" size="small" empty-text="暂无事件数据">
            <el-table-column prop="eventName" label="评审事件" min-width="140" />
            <el-table-column prop="assignedJudges" label="分配评委" width="90" />
            <el-table-column prop="submittedReviews" label="已提交" width="90" />
            <el-table-column label="状态" width="90">
              <template #default="scope">
                <el-tag :type="scope.row.completed ? 'success' : 'warning'">{{ scope.row.completed ? '完成' : '进行中' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
          <p v-if="currentWorkStatus" class="meta-text">
            事件总数 {{ currentWorkStatus.summary.eventCount || 0 }}，完成事件 {{ currentWorkStatus.summary.completedEvents || 0 }}
          </p>
        </el-col>

        <el-col :xs="24" :lg="12">
          <h4 class="sub-title">评审结果</h4>
          <el-table :data="currentWorkResults" size="small" empty-text="暂无评审结果">
            <el-table-column prop="reviewEventID" label="事件ID" width="90" />
            <el-table-column label="最终分" width="90">
              <template #default="scope">{{ finalScoreFromResult(scope.row) }}</template>
            </el-table-column>
            <el-table-column label="评分数" width="90">
              <template #default="scope">{{ reviewCountFromResult(scope.row) }}</template>
            </el-table-column>
            <el-table-column label="生成时间" min-width="180">
              <template #default="scope">{{ generatedAtFromResult(scope.row) }}</template>
            </el-table-column>
          </el-table>
        </el-col>
      </el-row>
    </section>
  </section>
</template>

<style scoped>
.judge-page {
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

.filter-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.summary-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.summary-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  padding: 14px;
  background: #fff;
}

.summary-label {
  color: var(--text-secondary, #909399);
  font-size: 12px;
  margin-bottom: 8px;
}

.summary-value {
  font-size: 28px;
  font-weight: 600;
}

.summary-text {
  font-size: 16px;
  line-height: 1.4;
}

.inner-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  padding: 14px;
  background: #fff;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
}

.inner-title {
  margin: 0;
  font-size: 16px;
}

.sub-title {
  margin: 0 0 10px;
  font-size: 14px;
  color: var(--text-secondary, #606266);
}

.muted-text {
  color: var(--text-secondary, #909399);
  font-size: 12px;
}

.meta-text {
  margin: 10px 0 0;
  color: var(--text-secondary, #909399);
  font-size: 12px;
}
</style>
