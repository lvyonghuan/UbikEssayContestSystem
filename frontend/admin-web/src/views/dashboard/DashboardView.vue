<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { fetchContests } from '@/services/repositories/contestRepository'
import type { Contest } from '@/types/api'
import { formatChinaDateTime, toChinaTimestamp } from '@/utils/date'

type ContestStatus = '未开始' | '进行中' | '已结束'

interface ContestBoardItem {
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
  return formatChinaDateTime(value)
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

function buildContestRow(contest: Contest): ContestBoardItem {
  const now = nowTs.value
  const startTs = toChinaTimestamp(contest.contestStartDate)
  const endTs = toChinaTimestamp(contest.contestEndDate)

  if (startTs === null || endTs === null) {
    return {
      contest,
      status: '未开始',
      progress: 0,
      countdownText: '待配置时间',
      startTs: 0,
      endTs: 0,
    }
  }

  if (now < startTs) {
    return {
      contest,
      status: '未开始',
      progress: 0,
      countdownText: `距开始 ${formatDuration(startTs - now)}`,
      startTs,
      endTs,
    }
  }

  if (now > endTs) {
    return {
      contest,
      status: '已结束',
      progress: 100,
      countdownText: '比赛已结束',
      startTs,
      endTs,
    }
  }

  const totalMs = endTs - startTs
  const elapsedMs = now - startTs
  const progress = totalMs > 0 ? Math.min(100, Math.max(0, Math.round((elapsedMs / totalMs) * 100))) : 0

  return {
    contest,
    status: '进行中',
    progress,
    countdownText: `距结束 ${formatDuration(endTs - now)}`,
    startTs,
    endTs,
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
    ElMessage.error(error instanceof Error ? error.message : '比赛管理加载失败')
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

function openContestManage() {
  router.push({ name: 'contests' })
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
        <h1 class="page-title">比赛管理</h1>
        <p class="page-subtitle">比赛一级视图，按进行中、未开始、已结束分层展示</p>
      </div>
      <el-button type="primary" plain @click="openContestManage">赛事配置</el-button>
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
      <el-table-column prop="contest.contestName" label="比赛名称" min-width="220" />
      <el-table-column label="比赛时间" min-width="260">
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
      <el-table-column label="比赛进度" min-width="260">
        <template #default="scope">
          <el-progress :percentage="scope.row.progress" :stroke-width="12" />
        </template>
      </el-table-column>
      <el-table-column label="倒计时" min-width="160">
        <template #default="scope">
          <span>{{ scope.row.countdownText }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120">
        <template #default="scope">
          <el-button link type="primary" @click="openContestDetail(scope.row.contest.contestID)">进入比赛</el-button>
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
