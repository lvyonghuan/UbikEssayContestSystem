<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { fetchContestByID } from '@/services/repositories/contestRepository'
import {
  fetchContestTrackStatusStats,
  exportTrackReviewExcel,
  fetchTrackReviewRanking,
} from '@/services/repositories/judgeRepository'
import { fetchTracks } from '@/services/repositories/trackRepository'
import type { Contest, ContestTrackStatusStat, Track, TrackRankItem } from '@/types/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const exportLoading = ref(false)

const contest = ref<Contest | null>(null)
const track = ref<Track | null>(null)
const trackStats = ref<ContestTrackStatusStat | null>(null)
const ranking = ref<TrackRankItem[]>([])

const contestId = computed(() => {
  const parsed = Number(route.params.contestId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

const trackId = computed(() => {
  const parsed = Number(route.params.trackId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

const submissionCount = computed(() => trackStats.value?.totalWorks || 0)

const authorCount = computed(() => trackStats.value?.totalAuthors || 0)

const reviewedWorkCount = computed(() => ranking.value.length)

const reviewCoverage = computed(() => {
  if (submissionCount.value === 0) {
    return 0
  }
  return Math.round((reviewedWorkCount.value / submissionCount.value) * 100)
})

const averageFinalScore = computed(() => {
  if (!ranking.value.length) {
    return '0.00'
  }
  const total = ranking.value.reduce((sum, item) => sum + Number(item.finalScore || 0), 0)
  return (total / ranking.value.length).toFixed(2)
})

const workStatusSummary = computed(() => {
  const merged = trackStats.value?.statusCounts || {}

  return Object.entries(merged)
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
})

async function loadDetail() {
  if (!contestId.value || !trackId.value) {
    ElMessage.error('页面参数不合法')
    return
  }

  loading.value = true
  contest.value = null
  track.value = null
  trackStats.value = null
  ranking.value = []
  try {
    const currentContest = await fetchContestByID(contestId.value)
    if (!currentContest) {
      ElMessage.error('比赛不存在或已被删除')
      return
    }

    const [rawTracks, nextTrackStats, nextRanking] = await Promise.all([
      fetchTracks(contestId.value),
      fetchContestTrackStatusStats(contestId.value),
      fetchTrackReviewRanking(trackId.value),
    ])

    const currentTrack = rawTracks.find((item) => item.trackID === trackId.value)
    if (!currentTrack) {
      ElMessage.error('赛道不存在或不属于当前比赛')
      return
    }

    const currentTrackStats = nextTrackStats.find((item) => item.trackID === trackId.value) || null

    contest.value = currentContest
    track.value = currentTrack
    trackStats.value = currentTrackStats
    ranking.value = nextRanking
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛道页面加载失败')
  } finally {
    loading.value = false
  }
}

function goBack() {
  if (!contestId.value) {
    router.push({ name: 'dashboard' })
    return
  }
  router.push({ name: 'contest-detail', params: { contestId: contestId.value } })
}

async function exportRanking() {
  if (!trackId.value) {
    ElMessage.warning('赛道参数不合法')
    return
  }

  exportLoading.value = true
  try {
    const { blob, filename } = await exportTrackReviewExcel(trackId.value)
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

onMounted(loadDetail)
</script>

<template>
  <section class="page-card track-detail-page" v-loading="loading">
    <div class="header-row">
      <div>
        <h1 class="page-title">赛道页面</h1>
        <p class="page-subtitle">赛道二级视图，展示投稿分布、投稿量和评审情况</p>
      </div>
      <el-space wrap>
        <el-button type="primary" plain :loading="exportLoading" @click="exportRanking">导出赛道排名</el-button>
        <el-button plain @click="goBack">返回比赛页面</el-button>
      </el-space>
    </div>

    <template v-if="contest && track">
      <el-descriptions border :column="2" class="base-card">
        <el-descriptions-item label="比赛名称">{{ contest.contestName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="赛道名称">{{ track.trackName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="赛道 ID">{{ track.trackID || '-' }}</el-descriptions-item>
        <el-descriptions-item label="赛道描述">{{ track.trackDescription || '-' }}</el-descriptions-item>
      </el-descriptions>

      <div class="summary-row">
        <article class="summary-card">
          <div class="summary-label">赛道投稿总数</div>
          <div class="summary-value">{{ submissionCount }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">投稿作者数</div>
          <div class="summary-value">{{ authorCount }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">已生成评审结果</div>
          <div class="summary-value">{{ reviewedWorkCount }}</div>
        </article>
        <article class="summary-card">
          <div class="summary-label">评审覆盖率</div>
          <div class="summary-value">{{ reviewCoverage }}%</div>
          <div class="summary-subline">平均分 {{ averageFinalScore }}</div>
        </article>
      </div>

      <el-row :gutter="16">
        <el-col :xs="24" :lg="8">
          <section class="inner-card">
            <h3 class="inner-title">投稿分布（按作品状态）</h3>
            <el-space v-if="workStatusSummary.length" wrap>
              <el-tag
                v-for="item in workStatusSummary"
                :key="item.name"
                type="success"
                effect="light"
              >
                {{ item.name }}: {{ item.count }}
              </el-tag>
            </el-space>
            <p v-else class="muted-text">暂无投稿分布数据</p>
          </section>
        </el-col>

        <el-col :xs="24" :lg="16">
          <section class="inner-card">
            <h3 class="inner-title">评审情况</h3>
            <el-table :data="ranking" size="small" empty-text="暂无评审结果">
              <el-table-column label="排名" width="70">
                <template #default="scope">{{ scope.$index + 1 }}</template>
              </el-table-column>
              <el-table-column prop="workTitle" label="作品" min-width="190" />
              <el-table-column prop="authorName" label="作者" min-width="130" />
              <el-table-column label="最终分" width="100">
                <template #default="scope">{{ Number(scope.row.finalScore || 0).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column prop="reviewCount" label="评审数" width="90" />
              <el-table-column prop="workID" label="作品ID" width="100" />
            </el-table>
          </section>
        </el-col>
      </el-row>
    </template>
  </section>
</template>

<style scoped>
.track-detail-page {
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

.base-card {
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
  margin-top: 6px;
  color: var(--text-secondary, #606266);
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

.sub-title {
  margin: 16px 0 10px;
  font-size: 14px;
  color: var(--text-secondary, #606266);
}

.muted-text {
  margin: 0;
  color: var(--text-secondary, #909399);
  font-size: 12px;
}
</style>
