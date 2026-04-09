<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import TrackDistributionPie from '@/components/TrackDistributionPie.vue'
import { fetchContestByID } from '@/services/repositories/contestRepository'
import { fetchFlowMounts, fetchScriptFlows } from '@/services/repositories/scriptFlowRepository'
import { fetchTracks } from '@/services/repositories/trackRepository'
import { fetchWorks } from '@/services/repositories/workRepository'
import type { Contest, DashboardDistributionPoint, FlowMount, Track, Work } from '@/types/api'

type ContestStatus = '即将开始' | '进行中' | '已结束'

interface TrackWithStats extends Track {
  submissionCount: number
}

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const contest = ref<Contest | null>(null)
const tracks = ref<TrackWithStats[]>([])
const works = ref<Work[]>([])
const contestMountCount = ref(0)
const trackMountCount = ref(0)

const workFilters = reactive({
  workTitle: '',
  authorName: '',
  trackID: undefined as number | undefined,
})

const contestId = computed(() => {
  const parsed = Number(route.params.contestId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

function formatDate(value: string) {
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value || '-'
}

function getContestStatus(current: Contest | null): ContestStatus {
  if (!current) {
    return '即将开始'
  }

  const now = dayjs()
  const start = dayjs(current.contestStartDate)
  const end = dayjs(current.contestEndDate)

  if (!start.isValid() || !end.isValid()) {
    return '即将开始'
  }
  if (now.isBefore(start)) {
    return '即将开始'
  }
  if (now.isAfter(end)) {
    return '已结束'
  }
  return '进行中'
}

function buildContestProgress(current: Contest | null) {
  if (!current) {
    return 0
  }

  const now = dayjs()
  const start = dayjs(current.contestStartDate)
  const end = dayjs(current.contestEndDate)

  if (!start.isValid() || !end.isValid()) {
    return 0
  }
  if (now.isBefore(start)) {
    return 0
  }
  if (now.isAfter(end)) {
    return 100
  }

  const totalMs = end.diff(start)
  const elapsedMs = now.diff(start)
  if (totalMs <= 0) {
    return 0
  }
  return Math.min(100, Math.max(0, Math.round((elapsedMs / totalMs) * 100)))
}

const contestStatus = computed(() => getContestStatus(contest.value))
const contestProgress = computed(() => buildContestProgress(contest.value))
const totalSubmissions = computed(() => works.value.length)

const pieData = computed<DashboardDistributionPoint[]>(() => {
  return tracks.value
    .map((track) => ({
      name: track.trackName || `赛道 ${track.trackID || '-'}`,
      value: track.submissionCount,
    }))
    .filter((item) => item.value > 0)
    .sort((a, b) => b.value - a.value)
})

const filteredWorks = computed(() => {
  return works.value.filter((work) => {
    const titleMatched =
      !workFilters.workTitle ||
      (work.workTitle || '').toLowerCase().includes(workFilters.workTitle.trim().toLowerCase())
    const authorMatched =
      !workFilters.authorName ||
      (work.authorName || '').toLowerCase().includes(workFilters.authorName.trim().toLowerCase())
    const trackMatched = !workFilters.trackID || work.trackID === workFilters.trackID

    return titleMatched && authorMatched && trackMatched
  })
})

const trackOptions = computed(() => {
  return tracks.value
    .filter((track): track is TrackWithStats & { trackID: number } => typeof track.trackID === 'number')
    .map((track) => ({
      label: track.trackName || `赛道 ${track.trackID}`,
      value: track.trackID,
    }))
})

async function loadFlowMountStats(contestID: number, trackIDList: number[]) {
  const flows = await fetchScriptFlows()
  const mountGroups = await Promise.all(
    flows.map(async (flow) => {
      if (!flow.flowID) {
        return [] as FlowMount[]
      }
      try {
        return await fetchFlowMounts(flow.flowID)
      } catch {
        return [] as FlowMount[]
      }
    }),
  )

  const allMounts = mountGroups.flat()
  contestMountCount.value = allMounts.filter(
    (mount) => mount.containerType === 'contest' && mount.containerID === contestID,
  ).length
  trackMountCount.value = allMounts.filter(
    (mount) => mount.containerType === 'track' && trackIDList.includes(mount.containerID),
  ).length
}

async function loadDetail() {
  if (!contestId.value) {
    ElMessage.error('比赛参数不合法')
    return
  }

  loading.value = true
  contest.value = null
  tracks.value = []
  works.value = []
  try {
    const currentContest = await fetchContestByID(contestId.value)
    if (!currentContest) {
      ElMessage.error('比赛不存在或已被删除')
      return
    }

    contest.value = currentContest

    const rawTracks = await fetchTracks(contestId.value)
    const trackRows = await Promise.all(
      rawTracks.map(async (track) => {
        if (!track.trackID) {
          return { track, works: [] as Work[] }
        }
        const nextWorks = await fetchWorks({ trackID: track.trackID, limit: 100 })
        return { track, works: nextWorks }
      }),
    )

    const nextTracks: TrackWithStats[] = trackRows.map((row) => ({
      ...row.track,
      submissionCount: row.works.length,
    }))
    tracks.value = nextTracks
    works.value = trackRows.flatMap((row) => row.works)

    const validTrackIDs = nextTracks
      .filter((track): track is TrackWithStats & { trackID: number } => typeof track.trackID === 'number')
      .map((track) => track.trackID)
    await loadFlowMountStats(contestId.value, validTrackIDs)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '比赛详情加载失败')
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push({ name: 'dashboard' })
}

onMounted(loadDetail)
</script>

<template>
  <section class="page-card detail-page" v-loading="loading">
    <div class="header-row">
      <div>
        <h1 class="page-title">比赛详情</h1>
        <p class="page-subtitle">按比赛上下文查看赛道、投稿与脚本流程挂载情况</p>
      </div>
      <el-button plain @click="goBack">返回看板</el-button>
    </div>

    <template v-if="contest">
      <el-descriptions border :column="2" class="contest-basic">
        <el-descriptions-item label="比赛名称">{{ contest.contestName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="比赛状态">
          <el-tag v-if="contestStatus === '进行中'" type="success">进行中</el-tag>
          <el-tag v-else-if="contestStatus === '即将开始'" type="warning">未开始</el-tag>
          <el-tag v-else type="info">已结束</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatDate(contest.contestStartDate) }}</el-descriptions-item>
        <el-descriptions-item label="结束时间">{{ formatDate(contest.contestEndDate) }}</el-descriptions-item>
        <el-descriptions-item label="比赛简介" :span="2">{{ contest.contestIntroduction || '-' }}</el-descriptions-item>
      </el-descriptions>

      <div class="summary-row">
        <article class="summary-card">
          <div class="summary-label">投稿总量</div>
          <div class="summary-value">{{ totalSubmissions }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">赛道数量</div>
          <div class="summary-value">{{ tracks.length }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">比赛进度</div>
          <el-progress :percentage="contestProgress" :stroke-width="12" />
        </article>
        <article class="summary-card">
          <div class="summary-label">流程挂载</div>
          <div class="summary-subline">比赛级 {{ contestMountCount }} / 赛道级 {{ trackMountCount }}</div>
        </article>
      </div>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="10">
          <TrackDistributionPie :data="pieData" title="赛道投稿分布" />
        </el-col>
        <el-col :xs="24" :lg="14">
          <section class="inner-card">
            <h3 class="inner-title">赛道投稿情况</h3>
            <el-table :data="tracks" style="width: 100%" size="small" empty-text="当前比赛暂无赛道">
              <el-table-column prop="trackName" label="赛道名称" min-width="160" />
              <el-table-column prop="trackDescription" label="赛道说明" min-width="200" />
              <el-table-column prop="submissionCount" label="投稿数量" width="110" />
            </el-table>
          </section>
        </el-col>
      </el-row>

      <section class="inner-card">
        <div class="works-header">
          <h3 class="inner-title">作品列表（名称优先）</h3>
          <el-space wrap>
            <el-input v-model="workFilters.workTitle" placeholder="作品标题" style="width: 180px" clearable />
            <el-input v-model="workFilters.authorName" placeholder="作者名" style="width: 180px" clearable />
            <el-select v-model="workFilters.trackID" placeholder="赛道" style="width: 180px" clearable>
              <el-option v-for="track in trackOptions" :key="track.value" :label="track.label" :value="track.value" />
            </el-select>
          </el-space>
        </div>
        <el-table :data="filteredWorks" style="width: 100%" empty-text="暂无作品">
          <el-table-column prop="workTitle" label="作品标题" min-width="220" />
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
        </el-table>
      </section>
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
  font-size: 14px;
  color: var(--text-primary, #303133);
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

.works-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
</style>
