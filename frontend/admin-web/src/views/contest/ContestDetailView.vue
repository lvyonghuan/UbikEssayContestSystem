<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import TrackDistributionPie from '@/components/TrackDistributionPie.vue'
import { featureFlags } from '@/features/flags'
import { fetchContestByID } from '@/services/repositories/contestRepository'
import { fetchContestTrackStatusStats } from '@/services/repositories/judgeRepository'
import { createTrack, fetchTracks, removeTrack, updateTrack } from '@/services/repositories/trackRepository'
import { fetchWorks } from '@/services/repositories/workRepository'
import type { Contest, ContestTrackStatusStat, DashboardDistributionPoint, Track, Work } from '@/types/api'
import { formatChinaDateTime, toChinaTimestamp } from '@/utils/date'

type ContestStatus = '未开始' | '进行中' | '已结束'

interface TrackWithStats extends Track {
  submissionCount: number
  totalAuthors: number
}

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const dialogVisible = ref(false)
const savingTrack = ref(false)
const editingTrackId = ref<number | null>(null)

const contest = ref<Contest | null>(null)
const tracks = ref<TrackWithStats[]>([])
const works = ref<Work[]>([])

const trackForm = reactive({
  trackName: '',
  trackDescription: '',
  trackSettingsText: '{}',
})

const contestId = computed(() => {
  const parsed = Number(route.params.contestId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

function formatDate(value: string) {
  return formatChinaDateTime(value)
}

function getContestStatus(current: Contest | null): ContestStatus {
  if (!current) {
    return '未开始'
  }

  const now = Date.now()
  const start = toChinaTimestamp(current.contestStartDate)
  const end = toChinaTimestamp(current.contestEndDate)

  if (start === null || end === null) {
    return '未开始'
  }
  if (now < start) {
    return '未开始'
  }
  if (now > end) {
    return '已结束'
  }
  return '进行中'
}

function buildContestProgress(current: Contest | null) {
  if (!current) {
    return 0
  }

  const now = Date.now()
  const start = toChinaTimestamp(current.contestStartDate)
  const end = toChinaTimestamp(current.contestEndDate)

  if (start === null || end === null) {
    return 0
  }
  if (now < start) {
    return 0
  }
  if (now > end) {
    return 100
  }

  const totalMs = end - start
  const elapsedMs = now - start
  if (totalMs <= 0) {
    return 0
  }
  return Math.min(100, Math.max(0, Math.round((elapsedMs / totalMs) * 100)))
}

const contestStatus = computed(() => getContestStatus(contest.value))
const contestProgress = computed(() => buildContestProgress(contest.value))

const totalSubmissions = computed(() => {
  return tracks.value.reduce((sum, track) => sum + track.submissionCount, 0)
})

const participatingAuthors = computed(() => {
  const authorSet = new Set<string>()
  for (const work of works.value) {
    if (typeof work.authorID === 'number' && work.authorID > 0) {
      authorSet.add(`id:${work.authorID}`)
      continue
    }
    const authorName = (work.authorName || '').trim()
    if (authorName) {
      authorSet.add(`name:${authorName.toLowerCase()}`)
    }
  }
  return authorSet.size
})

const pieData = computed<DashboardDistributionPoint[]>(() => {
  return tracks.value
    .map((track) => ({
      name: track.trackName || `赛道 ${track.trackID || '-'}`,
      value: track.submissionCount,
    }))
    .filter((item) => item.value > 0)
    .sort((a, b) => b.value - a.value)
})

async function loadDetail() {
  if (!contestId.value) {
    ElMessage.error('比赛参数不合法')
    return
  }

  loading.value = true
  try {
    const currentContest = await fetchContestByID(contestId.value)
    if (!currentContest) {
      ElMessage.error('比赛不存在或已被删除')
      contest.value = null
      tracks.value = []
      works.value = []
      return
    }

    contest.value = currentContest

    const [rawTracks, trackStats] = await Promise.all([
      fetchTracks(contestId.value),
      fetchContestTrackStatusStats(contestId.value),
    ])

    const statMap = new Map<number, ContestTrackStatusStat>()
    for (const item of trackStats) {
      statMap.set(item.trackID, item)
    }

    tracks.value = rawTracks.map((track) => {
      const stat = typeof track.trackID === 'number' ? statMap.get(track.trackID) : undefined
      return {
        ...track,
        submissionCount: stat?.totalWorks || 0,
        totalAuthors: stat?.totalAuthors || 0,
      }
    })

    const worksByTrack = await Promise.all(
      tracks.value
        .filter((track): track is TrackWithStats & { trackID: number } => typeof track.trackID === 'number')
        .map(async (track) => {
          try {
            return await fetchWorks({ trackID: track.trackID, limit: 100 })
          } catch {
            return [] as Work[]
          }
        }),
    )
    works.value = worksByTrack.flat()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '比赛详情加载失败')
    contest.value = null
    tracks.value = []
    works.value = []
  } finally {
    loading.value = false
  }
}

function openCreateTrackDialog() {
  editingTrackId.value = null
  trackForm.trackName = ''
  trackForm.trackDescription = ''
  trackForm.trackSettingsText = '{}'
  dialogVisible.value = true
}

function openEditTrackDialog(track: TrackWithStats) {
  editingTrackId.value = track.trackID || null
  trackForm.trackName = track.trackName || ''
  trackForm.trackDescription = track.trackDescription || ''
  trackForm.trackSettingsText = JSON.stringify(track.trackSettings || {}, null, 2)
  dialogVisible.value = true
}

async function saveTrack() {
  if (!contestId.value) {
    ElMessage.warning('比赛参数不合法')
    return
  }

  const trackName = trackForm.trackName.trim()
  if (!trackName) {
    ElMessage.warning('请填写赛道名称')
    return
  }

  let parsedSettings: Record<string, unknown> = {}
  try {
    parsedSettings = trackForm.trackSettingsText.trim()
      ? (JSON.parse(trackForm.trackSettingsText) as Record<string, unknown>)
      : {}
  } catch {
    ElMessage.error('赛道设置必须是合法 JSON')
    return
  }

  const payload: Track = {
    contestID: contestId.value,
    trackName,
    trackDescription: trackForm.trackDescription.trim(),
    trackSettings: parsedSettings,
  }

  savingTrack.value = true
  try {
    if (editingTrackId.value) {
      await updateTrack(editingTrackId.value, payload)
      ElMessage.success('赛道更新成功')
    } else {
      await createTrack(payload)
      ElMessage.success('赛道创建成功')
    }

    dialogVisible.value = false
    await loadDetail()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛道保存失败')
  } finally {
    savingTrack.value = false
  }
}

async function deleteTrackRow(trackID?: number) {
  if (!trackID) {
    return
  }

  try {
    await ElMessageBox.confirm('删除赛道后无法恢复，确认继续吗？', '删除赛道', {
      type: 'warning',
    })
  } catch {
    return
  }

  try {
    await removeTrack(trackID)
    ElMessage.success('赛道已删除')
    await loadDetail()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛道删除失败')
  }
}

function openTrackDetail(trackID?: number) {
  if (!contestId.value || !trackID) {
    ElMessage.warning('赛道参数不合法')
    return
  }

  router.push({
    name: 'contest-track-detail',
    params: { contestId: contestId.value, trackId: trackID },
  })
}

function goBack() {
  router.push({ name: 'dashboard' })
}

function openJudgeReview() {
  if (!contestId.value) {
    ElMessage.warning('比赛参数不合法')
    return
  }
  router.push({ name: 'judge-review', params: { contestId: contestId.value } })
}

onMounted(loadDetail)
</script>

<template>
  <section class="page-card detail-page" v-loading="loading">
    <div class="header-row">
      <div>
        <h1 class="page-title">比赛页面</h1>
        <p class="page-subtitle">比赛二级视图，聚合投稿指标并维护赛道</p>
      </div>
      <el-space wrap>
        <el-button v-if="featureFlags.judgeModule" type="primary" plain @click="openJudgeReview">评审管理</el-button>
        <el-button plain @click="goBack">返回比赛管理</el-button>
      </el-space>
    </div>

    <template v-if="contest">
      <el-descriptions border :column="2" class="contest-basic">
        <el-descriptions-item label="比赛名称">{{ contest.contestName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="比赛状态">
          <el-tag v-if="contestStatus === '进行中'" type="success">进行中</el-tag>
          <el-tag v-else-if="contestStatus === '未开始'" type="warning">未开始</el-tag>
          <el-tag v-else type="info">已结束</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatDate(contest.contestStartDate) }}</el-descriptions-item>
        <el-descriptions-item label="结束时间">{{ formatDate(contest.contestEndDate) }}</el-descriptions-item>
        <el-descriptions-item label="比赛简介" :span="2">{{ contest.contestIntroduction || '-' }}</el-descriptions-item>
      </el-descriptions>

      <div class="summary-row">
        <article class="summary-card">
          <div class="summary-label">参与作者</div>
          <div class="summary-value">{{ participatingAuthors }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">投稿总数</div>
          <div class="summary-value">{{ totalSubmissions }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">赛道总数</div>
          <div class="summary-value">{{ tracks.length }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">比赛进度</div>
          <el-progress :percentage="contestProgress" :stroke-width="12" />
          <div class="summary-subline">{{ contestStatus }}</div>
        </article>
      </div>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="10">
          <TrackDistributionPie :data="pieData" title="赛道投稿分布" />
        </el-col>
        <el-col :xs="24" :lg="14">
          <section class="inner-card">
            <div class="inner-header">
              <h3 class="inner-title">赛道列表</h3>
              <el-button type="primary" plain @click="openCreateTrackDialog">新增赛道</el-button>
            </div>
            <el-table :data="tracks" style="width: 100%" size="small" empty-text="当前比赛暂无赛道">
              <el-table-column prop="trackName" label="赛道名称" min-width="160" />
              <el-table-column prop="trackDescription" label="赛道说明" min-width="200" />
              <el-table-column prop="submissionCount" label="投稿数量" width="110" />
              <el-table-column label="操作" width="230">
                <template #default="scope">
                  <el-space>
                    <el-button link type="primary" @click="openTrackDetail(scope.row.trackID)">赛道页</el-button>
                    <el-button link type="success" @click="openEditTrackDialog(scope.row)">编辑</el-button>
                    <el-button link type="danger" @click="deleteTrackRow(scope.row.trackID)">删除</el-button>
                  </el-space>
                </template>
              </el-table-column>
            </el-table>
          </section>
        </el-col>
      </el-row>

      <el-dialog v-model="dialogVisible" :title="editingTrackId ? '编辑赛道' : '新增赛道'" width="640px">
        <el-form label-position="top">
          <el-form-item label="赛道名称" required>
            <el-input v-model="trackForm.trackName" />
          </el-form-item>
          <el-form-item label="赛道描述">
            <el-input v-model="trackForm.trackDescription" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item label="赛道设置(JSON，可选)">
            <el-input v-model="trackForm.trackSettingsText" type="textarea" :rows="8" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button :disabled="savingTrack" @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="savingTrack" @click="saveTrack">保存</el-button>
        </template>
      </el-dialog>
    </template>
  </section>
</template>

<style scoped>
.detail-page {
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

.contest-basic {
  margin-bottom: 6px;
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

.summary-subline {
  font-size: 13px;
  color: var(--text-primary, #303133);
  margin-top: 6px;
}

.inner-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  padding: 14px;
  background: #fff;
}

.inner-title {
  margin: 0 0 12px;
  font-size: 16px;
}

.inner-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

</style>
