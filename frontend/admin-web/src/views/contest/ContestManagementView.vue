<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import { createContest, fetchContests, removeContest, updateContest } from '@/services/repositories/contestRepository'
import { createTrack, fetchTracks, removeTrack, updateTrack } from '@/services/repositories/trackRepository'
import type { Contest, Track } from '@/types/api'

interface TrackDraft {
  rowKey: string
  trackID?: number
  trackName: string
  trackDescription: string
  trackSettingsText: string
}

interface PreparedTrackRow {
  draft: TrackDraft
  payload: Track
}

const router = useRouter()
const loading = ref(false)
const dialogVisible = ref(false)
const dialogLoading = ref(false)
const editingId = ref<number | null>(null)
const contests = ref<Contest[]>([])
const trackRows = ref<TrackDraft[]>([])
const removedTrackIds = ref<number[]>([])

let trackRowSeed = 1

const form = reactive<Contest>({
  contestName: '',
  contestIntroduction: '',
  contestStartDate: '',
  contestEndDate: '',
})

async function loadContests() {
  loading.value = true
  try {
    contests.value = await fetchContests()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛事加载失败')
  } finally {
    loading.value = false
  }
}

function nextTrackRowKey() {
  const key = `track-row-${trackRowSeed}`
  trackRowSeed += 1
  return key
}

function createTrackDraft(track?: Track): TrackDraft {
  return {
    rowKey: nextTrackRowKey(),
    trackID: track?.trackID,
    trackName: track?.trackName || '',
    trackDescription: track?.trackDescription || '',
    trackSettingsText: JSON.stringify(track?.trackSettings || {}, null, 2),
  }
}

function resetTrackRows() {
  trackRows.value = []
  removedTrackIds.value = []
}

function addTrackRow(track?: Track) {
  trackRows.value.push(createTrackDraft(track))
}

function removeTrackRow(rowKey: string) {
  const index = trackRows.value.findIndex((row) => row.rowKey === rowKey)
  if (index < 0) {
    return
  }

  const row = trackRows.value[index]
  if (typeof row.trackID === 'number' && row.trackID > 0 && !removedTrackIds.value.includes(row.trackID)) {
    removedTrackIds.value.push(row.trackID)
  }
  trackRows.value.splice(index, 1)
}

async function loadTrackRowsForContest(contestId: number) {
  dialogLoading.value = true
  try {
    const tracks = await fetchTracks(contestId)
    trackRows.value = tracks.map((track) => createTrackDraft(track))
    removedTrackIds.value = []
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛道加载失败')
    trackRows.value = []
    removedTrackIds.value = []
  } finally {
    dialogLoading.value = false
  }
}

function openCreateDialog() {
  editingId.value = null
  Object.assign(form, {
    contestName: '',
    contestIntroduction: '',
    contestStartDate: '',
    contestEndDate: '',
  })
  resetTrackRows()
  addTrackRow()
  dialogVisible.value = true
}

async function openEditDialog(contest: Contest) {
  editingId.value = contest.contestID || null
  Object.assign(form, {
    ...contest,
    contestStartDate: toPickerValue(contest.contestStartDate),
    contestEndDate: toPickerValue(contest.contestEndDate),
  })
  resetTrackRows()
  dialogVisible.value = true

  if (typeof contest.contestID === 'number' && contest.contestID > 0) {
    await loadTrackRowsForContest(contest.contestID)
  }
}

function toPickerValue(value: string) {
  const parsed = dayjs(value)
  if (!parsed.isValid()) {
    return value
  }
  return parsed.format('YYYY-MM-DD HH:mm')
}

function buildPreparedTrackRows(contestId: number): PreparedTrackRow[] | null {
  const preparedRows: PreparedTrackRow[] = []

  for (let i = 0; i < trackRows.value.length; i += 1) {
    const row = trackRows.value[i]
    const rowName = row.trackName.trim()
    if (!rowName) {
      ElMessage.warning(`请填写第 ${i + 1} 条赛道名称`)
      return null
    }

    let parsedSettings: Record<string, unknown> = {}
    try {
      parsedSettings = row.trackSettingsText.trim()
        ? (JSON.parse(row.trackSettingsText) as Record<string, unknown>)
        : {}
    } catch {
      ElMessage.error(`第 ${i + 1} 条赛道设置不是合法 JSON`)
      return null
    }

    preparedRows.push({
      draft: row,
      payload: {
        contestID: contestId,
        trackName: rowName,
        trackDescription: row.trackDescription.trim(),
        trackSettings: parsedSettings,
      },
    })
  }

  return preparedRows
}

async function persistTrackRows(preparedRows: PreparedTrackRow[]) {
  let succeeded = 0
  let failed = 0
  let firstError = ''

  for (const row of preparedRows) {
    try {
      if (typeof row.draft.trackID === 'number' && row.draft.trackID > 0) {
        await updateTrack(row.draft.trackID, row.payload)
      } else {
        const created = await createTrack(row.payload)
        if (typeof created?.trackID === 'number' && created.trackID > 0) {
          row.draft.trackID = created.trackID
        }
      }
      succeeded += 1
    } catch (error) {
      failed += 1
      if (!firstError) {
        firstError = error instanceof Error ? error.message : '赛道保存失败'
      }
    }
  }

  const remainRemovedTrackIds: number[] = []
  for (const trackId of removedTrackIds.value) {
    try {
      await removeTrack(trackId)
      succeeded += 1
    } catch (error) {
      failed += 1
      remainRemovedTrackIds.push(trackId)
      if (!firstError) {
        firstError = error instanceof Error ? error.message : '赛道删除失败'
      }
    }
  }

  removedTrackIds.value = remainRemovedTrackIds
  return { succeeded, failed, firstError }
}

async function saveContest() {
  try {
    const wasEditing = typeof editingId.value === 'number' && editingId.value > 0
    let targetContestId = editingId.value
    if (wasEditing && targetContestId) {
      await updateContest(targetContestId, form)
    } else {
      const createdContest = await createContest(form)
      if (!createdContest?.contestID) {
        throw new Error('赛事创建成功但未返回 contestID')
      }
      targetContestId = createdContest.contestID
      editingId.value = targetContestId
    }

    const preparedRows = buildPreparedTrackRows(targetContestId)
    if (!preparedRows) {
      return
    }

    const trackPersistResult = await persistTrackRows(preparedRows)
    await loadContests()

    if (trackPersistResult.failed === 0) {
      ElMessage.success(wasEditing ? '赛事与赛道保存成功' : '赛事与赛道创建成功')
      dialogVisible.value = false
      return
    }

    ElMessage.warning(
      `赛事已保存，赛道操作成功 ${trackPersistResult.succeeded} 项，失败 ${trackPersistResult.failed} 项。${
        trackPersistResult.firstError ? `首个错误：${trackPersistResult.firstError}` : ''
      }`,
    )
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  }
}

async function deleteContest(contestId: number | undefined) {
  if (!contestId) {
    return
  }
  try {
    await ElMessageBox.confirm('删除后无法恢复，确认继续吗？', '删除赛事', {
      type: 'warning',
    })
    await removeContest(contestId)
    ElMessage.success('赛事已删除')
    await loadContests()
  } catch {
    // canceled
  }
}

function openDetail(contestId: number | undefined) {
  if (!contestId) {
    ElMessage.warning('比赛 ID 不存在')
    return
  }
  router.push({ name: 'contest-detail', params: { contestId } })
}

onMounted(loadContests)
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">赛事管理</h1>
        <p class="page-subtitle">支持创建、编辑、删除征文活动</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">新建赛事</el-button>
    </div>

    <el-table :data="contests" v-loading="loading" style="width: 100%">
      <el-table-column prop="contestName" label="赛事名称" min-width="160" />
      <el-table-column prop="contestID" label="ID" width="80" />
      <el-table-column prop="contestStartDate" label="开始时间" min-width="160" />
      <el-table-column prop="contestEndDate" label="结束时间" min-width="160" />
      <el-table-column label="操作" width="240">
        <template #default="scope">
          <el-space>
            <el-button link type="success" @click="openDetail(scope.row.contestID)">详情</el-button>
            <el-button link type="primary" @click="openEditDialog(scope.row)">编辑</el-button>
            <el-button link type="danger" @click="deleteContest(scope.row.contestID)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑赛事' : '新建赛事'" width="860px">
      <el-form label-position="top">
        <el-form-item label="赛事名称" required>
          <el-input v-model="form.contestName" />
        </el-form-item>
        <el-form-item label="赛事简介">
          <el-input v-model="form.contestIntroduction" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="开始时间" required>
          <el-date-picker
            v-model="form.contestStartDate"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm"
            format="YYYY-MM-DD HH:mm"
            placeholder="选择开始时间"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="结束时间" required>
          <el-date-picker
            v-model="form.contestEndDate"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm"
            format="YYYY-MM-DD HH:mm"
            placeholder="选择结束时间"
            style="width: 100%"
          />
        </el-form-item>

        <el-divider content-position="left">赛道配置</el-divider>
        <div class="track-toolbar">
          <p class="track-hint">赛道设置 JSON 为可选项，留空时默认保存为空对象。</p>
          <el-button type="primary" plain @click="addTrackRow">新增赛道</el-button>
        </div>

        <div v-loading="dialogLoading" class="track-list-wrap">
          <el-empty v-if="trackRows.length === 0" description="暂无赛道配置" :image-size="70" />

          <section v-for="(trackRow, index) in trackRows" :key="trackRow.rowKey" class="track-card">
            <div class="track-card-header">
              <h4 class="track-card-title">赛道 {{ index + 1 }}</h4>
              <el-button link type="danger" @click="removeTrackRow(trackRow.rowKey)">移除</el-button>
            </div>

            <el-form-item :label="`赛道名称 #${index + 1}`" required>
              <el-input v-model="trackRow.trackName" />
            </el-form-item>
            <el-form-item label="赛道描述">
              <el-input v-model="trackRow.trackDescription" type="textarea" :rows="2" />
            </el-form-item>
            <el-form-item label="赛道设置(JSON，可选)">
              <el-input v-model="trackRow.trackSettingsText" type="textarea" :rows="6" />
            </el-form-item>
          </section>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveContest">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped>
.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.track-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.track-hint {
  margin: 0;
  color: var(--text-secondary, #909399);
  font-size: 12px;
}

.track-list-wrap {
  min-height: 80px;
}

.track-card {
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}

.track-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.track-card-title {
  margin: 0;
  font-size: 14px;
}
</style>
