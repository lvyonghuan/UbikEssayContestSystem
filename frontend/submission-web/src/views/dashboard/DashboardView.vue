<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { fetchContests } from '@/services/repositories/contestRepository'
import type { Contest, ContestStatus } from '@/types/api'

interface ContestBoardRow {
  contest: Contest
  status: ContestStatus
  progress: number
  countdownText: string
  startTs: number
  endTs: number
}

const router = useRouter()
const loading = ref(false)
const nowTs = ref(Date.now())
const contests = ref<Contest[]>([])

let timer: number | undefined

function formatDate(value: string) {
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value || '-'
}

function formatDuration(diffMs: number) {
  if (diffMs <= 0) {
    return '0分'
  }

  const totalMinutes = Math.floor(diffMs / 60000)
  const days = Math.floor(totalMinutes / (24 * 60))
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60)
  const minutes = totalMinutes % 60
  return `${days}天 ${hours}时 ${minutes}分`
}

function buildContestRow(contest: Contest): ContestBoardRow {
  const now = dayjs(nowTs.value)
  const startAt = dayjs(contest.contestStartDate)
  const endAt = dayjs(contest.contestEndDate)

  if (!startAt.isValid() || !endAt.isValid()) {
    return {
      contest,
      status: '未开始',
      progress: 0,
      countdownText: '待配置时间',
      startTs: 0,
      endTs: 0,
    }
  }

  if (now.isBefore(startAt)) {
    return {
      contest,
      status: '未开始',
      progress: 0,
      countdownText: `距开始 ${formatDuration(startAt.diff(now))}`,
      startTs: startAt.valueOf(),
      endTs: endAt.valueOf(),
    }
  }

  if (now.isAfter(endAt)) {
    return {
      contest,
      status: '已结束',
      progress: 100,
      countdownText: '比赛已结束',
      startTs: startAt.valueOf(),
      endTs: endAt.valueOf(),
    }
  }

  const totalMs = endAt.diff(startAt)
  const elapsedMs = now.diff(startAt)
  const progress = totalMs > 0 ? Math.min(100, Math.max(0, Math.round((elapsedMs / totalMs) * 100))) : 0

  return {
    contest,
    status: '进行中',
    progress,
    countdownText: `距结束 ${formatDuration(endAt.diff(now))}`,
    startTs: startAt.valueOf(),
    endTs: endAt.valueOf(),
  }
}

const sortedItems = computed(() => {
  const rank: Record<ContestStatus, number> = {
    '进行中': 0,
    '未开始': 1,
    '已结束': 2,
  }

  return contests.value
    .map((contest) => buildContestRow(contest))
    .sort((a, b) => {
      const statusDiff = rank[a.status] - rank[b.status]
      if (statusDiff !== 0) {
        return statusDiff
      }

      if (a.status === '进行中') {
        return a.endTs - b.endTs
      }
      if (a.status === '未开始') {
        return a.startTs - b.startTs
      }
      return b.endTs - a.endTs
    })
})

const ongoingCount = computed(() => sortedItems.value.filter((item) => item.status === '进行中').length)
const upcomingCount = computed(() => sortedItems.value.filter((item) => item.status === '未开始').length)
const endedCount = computed(() => sortedItems.value.filter((item) => item.status === '已结束').length)

async function loadBoard() {
  loading.value = true
  try {
    contests.value = await fetchContests()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '比赛看板加载失败')
    contests.value = []
  } finally {
    loading.value = false
  }
}

function openContestDetail(contestID?: number) {
  if (!contestID) {
    ElMessage.warning('比赛 ID 不存在')
    return
  }
  router.push({ name: 'contest-detail', params: { contestId: contestID } })
}

onMounted(() => {
  loadBoard()
  timer = window.setInterval(() => {
    nowTs.value = Date.now()
  }, 60000)
})

onBeforeUnmount(() => {
  if (timer) {
    window.clearInterval(timer)
  }
})
</script>

<template>
  <section class="page-card dashboard-page">
    <div class="header-row">
      <div>
        <h1 class="page-title">比赛看板</h1>
        <p class="page-subtitle">进行中优先，其次未开始，已结束自动归于末尾</p>
      </div>
    </div>

    <div class="metrics-row" v-loading="loading">
      <article class="metric-card">
        <div class="metric-label">全部比赛</div>
        <div class="metric-value">{{ sortedItems.length }}</div>
      </article>
      <article class="metric-card">
        <div class="metric-label">进行中</div>
        <div class="metric-value">{{ ongoingCount }}</div>
      </article>
      <article class="metric-card">
        <div class="metric-label">未开始</div>
        <div class="metric-value">{{ upcomingCount }}</div>
      </article>
      <article class="metric-card">
        <div class="metric-label">已结束</div>
        <div class="metric-value">{{ endedCount }}</div>
      </article>
    </div>

    <el-table :data="sortedItems" v-loading="loading" style="width: 100%" empty-text="暂无比赛数据">
      <el-table-column prop="contest.contestName" label="比赛名称" min-width="240" />
      <el-table-column label="比赛时间" min-width="280">
        <template #default="scope">
          <div>{{ formatDate(scope.row.contest.contestStartDate) }}</div>
          <div class="second-line">至 {{ formatDate(scope.row.contest.contestEndDate) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="scope">
          <el-tag v-if="scope.row.status === '进行中'" type="success">进行中</el-tag>
          <el-tag v-else-if="scope.row.status === '未开始'" type="warning">未开始</el-tag>
          <el-tag v-else type="info">已结束</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="倒计时进度" min-width="280">
        <template #default="scope">
          <el-progress
            :percentage="scope.row.progress"
            :stroke-width="12"
            :format="() => scope.row.countdownText"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="scope">
          <el-button link type="primary" @click="openContestDetail(scope.row.contest.contestID)">查看详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<style scoped>
.dashboard-page {
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

.metrics-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}

.metric-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 12px;
  padding: 16px;
  background: linear-gradient(120deg, #ffffff 0%, #f7fbff 100%);
}

.metric-label {
  color: var(--text-secondary, #909399);
  font-size: 12px;
}

.metric-value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 600;
}

.second-line {
  color: var(--text-secondary, #909399);
  font-size: 12px;
}
</style>
