<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import { fetchContests } from '@/services/repositories/contestRepository'
import { fetchMySubmissions, removeSubmission } from '@/services/repositories/submissionRepository'
import type { Work } from '@/types/api'

const router = useRouter()
const loading = ref(false)
const submissions = ref<Work[]>([])
const endedContestIDs = ref<Set<number>>(new Set())

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

function isReadonlyWork(work: Work) {
  const contestID = parseContestID(work)
  if (!contestID) {
    return false
  }
  return endedContestIDs.value.has(contestID)
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
    for (const contest of contests) {
      if (typeof contest.contestID !== 'number') {
        continue
      }
      const end = dayjs(contest.contestEndDate)
      if (end.isValid() && now.isAfter(end)) {
        endedIDs.add(contest.contestID)
      }
    }

    endedContestIDs.value = endedIDs
    submissions.value = workList
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '稿件列表加载失败')
    submissions.value = []
    endedContestIDs.value = new Set()
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
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="scope">
          <el-text v-if="isReadonlyWork(scope.row)" type="info">仅可查看</el-text>
          <el-space v-else>
            <el-button link type="primary" @click="openEdit(scope.row)">修改</el-button>
            <el-button link type="danger" @click="onDelete(scope.row)">删除</el-button>
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
