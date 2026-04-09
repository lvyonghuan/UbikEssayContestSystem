<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { fetchContestByID } from '@/services/repositories/contestRepository'
import { fetchTracksByContest } from '@/services/repositories/trackRepository'
import type { Contest, ContestStatus, Track } from '@/types/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const contest = ref<Contest | null>(null)
const tracks = ref<Track[]>([])

const contestID = computed(() => {
  const parsed = Number(route.params.contestId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

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

const contestStatus = computed<ContestStatus>(() => {
  if (!contest.value) {
    return '未开始'
  }

  const now = dayjs()
  const start = dayjs(contest.value.contestStartDate)
  const end = dayjs(contest.value.contestEndDate)

  if (!start.isValid() || !end.isValid() || now.isBefore(start)) {
    return '未开始'
  }
  if (now.isAfter(end)) {
    return '已结束'
  }
  return '进行中'
})

const countdownText = computed(() => {
  if (!contest.value) {
    return '-'
  }

  const now = dayjs()
  const start = dayjs(contest.value.contestStartDate)
  const end = dayjs(contest.value.contestEndDate)

  if (!start.isValid() || !end.isValid()) {
    return '待配置时间'
  }
  if (now.isBefore(start)) {
    return `距开始 ${formatDuration(start.diff(now))}`
  }
  if (now.isAfter(end)) {
    return '比赛已结束'
  }
  return `距结束 ${formatDuration(end.diff(now))}`
})

async function loadDetail() {
  if (!contestID.value) {
    ElMessage.error('比赛参数不合法')
    return
  }

  loading.value = true
  try {
    const [contestResult, trackResult] = await Promise.all([
      fetchContestByID(contestID.value),
      fetchTracksByContest(contestID.value),
    ])

    if (!contestResult) {
      ElMessage.error('比赛不存在或已被删除')
      return
    }

    contest.value = contestResult
    tracks.value = trackResult
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '比赛详情加载失败')
    contest.value = null
    tracks.value = []
  } finally {
    loading.value = false
  }
}

function openTrackDetail(trackID?: number) {
  if (!contestID.value || !trackID) {
    ElMessage.warning('赛道参数不完整')
    return
  }

  router.push({
    name: 'track-detail',
    params: { contestId: contestID.value, trackId: trackID },
  })
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
        <p class="page-subtitle">查看比赛信息与赛道列表，选择赛道直接投稿</p>
      </div>
      <el-button plain @click="goBack">返回看板</el-button>
    </div>

    <template v-if="contest">
      <el-descriptions border :column="2">
        <el-descriptions-item label="比赛名称">{{ contest.contestName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="比赛状态">
          <el-tag v-if="contestStatus === '进行中'" type="success">进行中</el-tag>
          <el-tag v-else-if="contestStatus === '未开始'" type="warning">未开始</el-tag>
          <el-tag v-else type="info">已结束</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatDate(contest.contestStartDate) }}</el-descriptions-item>
        <el-descriptions-item label="结束时间">{{ formatDate(contest.contestEndDate) }}</el-descriptions-item>
        <el-descriptions-item label="倒计时" :span="2">{{ countdownText }}</el-descriptions-item>
        <el-descriptions-item label="比赛简介" :span="2">{{ contest.contestIntroduction || '-' }}</el-descriptions-item>
      </el-descriptions>

      <section class="track-block">
        <h3>赛道列表</h3>
        <el-table :data="tracks" empty-text="该比赛暂未配置赛道" style="width: 100%">
          <el-table-column prop="trackName" label="赛道名称" min-width="180" />
          <el-table-column prop="trackDescription" label="赛道说明" min-width="260" />
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="scope">
              <el-button link type="primary" @click="openTrackDetail(scope.row.trackID)">查看并投稿</el-button>
            </template>
          </el-table-column>
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

.track-block {
  margin-top: 14px;
}

.track-block h3 {
  margin: 0 0 10px;
}
</style>
