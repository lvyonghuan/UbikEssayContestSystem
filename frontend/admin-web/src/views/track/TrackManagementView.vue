<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchContests } from '@/services/repositories/contestRepository'
import { createTrack, fetchTracks, removeTrack, updateTrack } from '@/services/repositories/trackRepository'
import type { Contest, Track } from '@/types/api'

const contests = ref<Contest[]>([])
const tracks = ref<Track[]>([])
const selectedContestId = ref<number>()
const loading = ref(false)
const dialogVisible = ref(false)
const editingTrackId = ref<number | null>(null)

const form = reactive({
  contestID: undefined as number | undefined,
  trackName: '',
  trackDescription: '',
  trackSettingsText: '{\n  "reviewMode": "double-blind"\n}',
})

const canQueryTracks = computed(() => Boolean(selectedContestId.value))

async function loadContests() {
  contests.value = await fetchContests()
  if (contests.value.length > 0) {
    selectedContestId.value = contests.value[0].contestID
  }
}

async function loadTracks() {
  if (!selectedContestId.value) {
    tracks.value = []
    return
  }
  loading.value = true
  try {
    tracks.value = await fetchTracks(selectedContestId.value)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛道加载失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingTrackId.value = null
  Object.assign(form, {
    contestID: selectedContestId.value,
    trackName: '',
    trackDescription: '',
    trackSettingsText: '{\n  "reviewMode": "double-blind"\n}',
  })
  dialogVisible.value = true
}

function openEditDialog(track: Track) {
  editingTrackId.value = track.trackID || null
  Object.assign(form, {
    contestID: track.contestID,
    trackName: track.trackName,
    trackDescription: track.trackDescription || '',
    trackSettingsText: JSON.stringify(track.trackSettings || {}, null, 2),
  })
  dialogVisible.value = true
}

async function saveTrack() {
  let parsedSettings: Record<string, unknown> = {}
  try {
    parsedSettings = JSON.parse(form.trackSettingsText)
  } catch {
    ElMessage.error('赛道设置必须是合法 JSON')
    return
  }

  const payload: Track = {
    contestID: form.contestID,
    trackName: form.trackName,
    trackDescription: form.trackDescription,
    trackSettings: parsedSettings,
  }

  try {
    if (editingTrackId.value) {
      await updateTrack(editingTrackId.value, payload)
      ElMessage.success('赛道更新成功')
    } else {
      await createTrack(payload)
      ElMessage.success('赛道创建成功')
    }
    dialogVisible.value = false
    await loadTracks()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  }
}

async function deleteTrack(trackId: number | undefined) {
  if (!trackId) {
    return
  }

  try {
    await ElMessageBox.confirm('删除后无法恢复，确认继续吗？', '删除赛道', {
      type: 'warning',
    })
    await removeTrack(trackId)
    ElMessage.success('赛道已删除')
    await loadTracks()
  } catch {
    // canceled
  }
}

watch(selectedContestId, loadTracks)

onMounted(async () => {
  await loadContests()
  await loadTracks()
})
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">赛道管理</h1>
        <p class="page-subtitle">支持按赛事筛选并维护赛道配置</p>
      </div>
      <el-space>
        <el-select v-model="selectedContestId" placeholder="选择赛事" style="width: 220px">
          <el-option v-for="contest in contests" :key="contest.contestID" :label="contest.contestName" :value="contest.contestID" />
        </el-select>
        <el-button type="primary" :disabled="!canQueryTracks" @click="openCreateDialog">新建赛道</el-button>
      </el-space>
    </div>

    <el-table :data="tracks" v-loading="loading" style="width: 100%">
      <el-table-column prop="trackID" label="ID" width="80" />
      <el-table-column prop="trackName" label="赛道名称" min-width="180" />
      <el-table-column prop="trackDescription" label="描述" min-width="220" />
      <el-table-column label="操作" width="180">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openEditDialog(scope.row)">编辑</el-button>
            <el-button link type="danger" @click="deleteTrack(scope.row.trackID)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingTrackId ? '编辑赛道' : '新建赛道'" width="620px">
      <el-form label-position="top">
        <el-form-item label="所属赛事" required>
          <el-select v-model="form.contestID" style="width: 100%">
            <el-option v-for="contest in contests" :key="contest.contestID" :label="contest.contestName" :value="contest.contestID" />
          </el-select>
        </el-form-item>
        <el-form-item label="赛道名称" required>
          <el-input v-model="form.trackName" />
        </el-form-item>
        <el-form-item label="赛道描述">
          <el-input v-model="form.trackDescription" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="赛道设置(JSON)">
          <el-input v-model="form.trackSettingsText" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveTrack">保存</el-button>
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
  gap: 12px;
  flex-wrap: wrap;
}
</style>
