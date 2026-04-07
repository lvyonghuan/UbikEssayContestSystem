<script setup lang="ts">
import { computed, onMounted, reactive } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { fetchContests } from '@/services/repositories/contestRepository'
import { fetchTracks } from '@/services/repositories/trackRepository'
import type { Contest, Track } from '@/types/api'

type ContestStatus = '即将开始' | '进行中' | '已结束'

interface ContestBoardItem {
  contest: Contest
  status: ContestStatus
  daysToStart: number | null
  trackCount: number
}

interface State {
  loading: boolean
  items: ContestBoardItem[]
  tracksByContest: Record<number, Track[]>
}

const state = reactive<State>({
  loading: true,
  items: [],
  tracksByContest: {},
})

function getContestStatus(contest: Contest): { status: ContestStatus; daysToStart: number | null } {
  const now = dayjs()
  const start = dayjs(contest.contestStartDate)
  const end = dayjs(contest.contestEndDate)

  if (now.isBefore(start)) {
    return { status: '即将开始', daysToStart: start.startOf('day').diff(now.startOf('day'), 'day') }
  }
  if (now.isAfter(end)) {
    return { status: '已结束', daysToStart: null }
  }
  return { status: '进行中', daysToStart: 0 }
}

const contestCount = computed(() => state.items.length)
const ongoingCount = computed(() => state.items.filter((item) => item.status === '进行中').length)
const upcomingCount = computed(() => state.items.filter((item) => item.status === '即将开始').length)
const totalTrackCount = computed(() => state.items.reduce((sum, item) => sum + item.trackCount, 0))

const sortedItems = computed(() => {
  const rank: Record<ContestStatus, number> = {
    '进行中': 0,
    '即将开始': 1,
    '已结束': 2,
  }

  return [...state.items].sort((a, b) => {
    const statusDiff = rank[a.status] - rank[b.status]
    if (statusDiff !== 0) {
      return statusDiff
    }
    return dayjs(a.contest.contestStartDate).valueOf() - dayjs(b.contest.contestStartDate).valueOf()
  })
})

async function loadBoard() {
  state.loading = true
  try {
    const contests = await fetchContests()
    const nextItems: ContestBoardItem[] = []
    const nextTracksByContest: Record<number, Track[]> = {}

    for (const contest of contests) {
      const contestId = contest.contestID
      const statusInfo = getContestStatus(contest)

      if (!contestId) {
        nextItems.push({
          contest,
          status: statusInfo.status,
          daysToStart: statusInfo.daysToStart,
          trackCount: 0,
        })
        continue
      }

      const tracks = await fetchTracks(contestId)
      nextTracksByContest[contestId] = tracks
      nextItems.push({
        contest,
        status: statusInfo.status,
        daysToStart: statusInfo.daysToStart,
        trackCount: tracks.length,
      })
    }

    state.items = nextItems
    state.tracksByContest = nextTracksByContest
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '看板加载失败')
  } finally {
    state.loading = false
  }
}

function tracksForContest(contestId?: number) {
  if (!contestId) {
    return []
  }
  return state.tracksByContest[contestId] || []
}

onMounted(loadBoard)
</script>

<template>
  <div class="dashboard-container">
    <section class="page-card">
      <h3 class="page-title">运营看板</h3>
      <p class="page-subtitle">按赛事组织管理视图，先看比赛，再看每个比赛下的赛道与作品入口</p>

      <div v-loading="state.loading" class="metrics-row">
        <article class="metric-card">
          <div class="metric-label">赛事数</div>
          <div class="metric-value">{{ contestCount }}</div>
        </article>
        <article class="metric-card">
          <div class="metric-label">进行中</div>
          <div class="metric-value">{{ ongoingCount }}</div>
        </article>
        <article class="metric-card">
          <div class="metric-label">即将开始</div>
          <div class="metric-value">{{ upcomingCount }}</div>
        </article>
        <article class="metric-card">
          <div class="metric-label">赛道总数</div>
          <div class="metric-value">{{ totalTrackCount }}</div>
        </article>
      </div>

      <div class="contests-section">
        <h4>赛事维度总览</h4>
        <el-table :data="sortedItems" style="width: 100%" v-loading="state.loading">
          <el-table-column type="expand">
            <template #default="{ row }">
              <div class="expand-wrap">
                <h5>赛道列表（{{ row.trackCount }}）</h5>
                <el-table :data="tracksForContest(row.contest.contestID)" size="small" empty-text="该赛事暂无赛道">
                  <el-table-column prop="trackID" label="赛道ID" width="90" />
                  <el-table-column prop="trackName" label="赛道名称" min-width="180" />
                  <el-table-column prop="trackDescription" label="赛道说明" min-width="220" />
                </el-table>
                <p class="works-hint">作品统计按赛事维度展示，待后端提供作品查询接口后接入真实数据。</p>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="contest.contestName" label="赛事名称" min-width="220" />
          <el-table-column prop="contest.contestStartDate" label="开始时间" min-width="170" />
          <el-table-column prop="contest.contestEndDate" label="结束时间" min-width="170" />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag v-if="row.status === '进行中'" type="success">进行中</el-tag>
              <el-tag v-else-if="row.status === '即将开始'" type="warning">即将开始</el-tag>
              <el-tag v-else type="info">已结束</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="距开始" width="110">
            <template #default="{ row }">
              <span v-if="row.status === '即将开始'">{{ row.daysToStart }} 天</span>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="trackCount" label="赛道数" width="100" />
        </el-table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard-container {
  padding: 20px;
}

.metrics-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.metric-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 12px;
  padding: 20px;
  background: linear-gradient(120deg, #ffffff 0%, #f7fbff 100%);
  text-align: center;
}

.metric-label {
  color: var(--text-secondary, #909399);
  font-size: 13px;
  margin-bottom: 8px;
}

.metric-value {
  font-size: 32px;
  font-weight: 600;
  color: var(--text-primary, #303133);
}

.charts-grid {
  margin-bottom: 24px;
}

.contests-section {
  margin-top: 16px;
}

.contests-section h4 {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
}

.page-card {
  background: white;
  border-radius: 8px;
  padding: 24px;
}

.page-title {
  margin: 0 0 8px 0;
  font-size: 20px;
  font-weight: 600;
}

.page-subtitle {
  margin: 0 0 24px 0;
  color: var(--text-secondary, #909399);
  font-size: 13px;
}

.expand-wrap {
  padding: 8px 12px 14px;
}

.expand-wrap h5 {
  margin: 0 0 10px;
  font-size: 14px;
}

.works-hint {
  margin: 10px 0 0;
  color: var(--text-secondary, #909399);
  font-size: 12px;
}
</style>
