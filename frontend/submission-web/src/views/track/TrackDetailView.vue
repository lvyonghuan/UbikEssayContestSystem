<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import type { FormInstance, UploadFile, UploadInstance } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { fetchContestByID } from '@/services/repositories/contestRepository'
import { createSubmission, fetchMySubmissions } from '@/services/repositories/submissionRepository'
import { fetchTrackByID } from '@/services/repositories/trackRepository'
import type { Contest, ContestStatus, SubmissionPayload, Track, Work } from '@/types/api'
import { validateDocFile } from '@/utils/file'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const submitting = ref(false)
const contest = ref<Contest | null>(null)
const track = ref<Track | null>(null)
const mySubmissions = ref<Work[]>([])
const selectedFile = ref<File | null>(null)

const formRef = ref<FormInstance>()
const uploadRef = ref<UploadInstance>()

const form = reactive({
  workTitle: '',
  description: '',
})

const contestID = computed(() => {
  const parsed = Number(route.params.contestId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

const trackID = computed(() => {
  const parsed = Number(route.params.trackId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

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

const canSubmit = computed(() => contestStatus.value === '进行中')

const submitBlockedMessage = computed(() => {
  if (contestStatus.value === '未开始') {
    return '比赛未开始，该赛道暂不开放投稿'
  }
  return '比赛已结束，该赛道仅可查看历史稿件'
})

const myTrackSubmissions = computed(() => {
  if (!trackID.value) {
    return []
  }
  return mySubmissions.value.filter((work) => work.trackID === trackID.value)
})

function formatDate(value?: string) {
  if (!value) {
    return '-'
  }
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value
}

function updateCurrentFile(file: File | null) {
  selectedFile.value = file
  if (!file) {
    return
  }

  const errorText = validateDocFile(file)
  if (errorText) {
    selectedFile.value = null
    uploadRef.value?.clearFiles()
    ElMessage.warning(errorText)
  }
}

function handleFileChange(uploadFile: UploadFile) {
  updateCurrentFile(uploadFile.raw || null)
}

function handleFileRemove() {
  updateCurrentFile(null)
}

function goBack() {
  if (!contestID.value) {
    router.push({ name: 'dashboard' })
    return
  }
  router.push({ name: 'contest-detail', params: { contestId: contestID.value } })
}

function openEditSubmission(workID?: number) {
  if (!workID) {
    ElMessage.warning('稿件 ID 缺失，无法修改')
    return
  }

  if (!canSubmit.value) {
    ElMessage.warning('当前比赛阶段仅支持查看，不支持修改稿件')
    return
  }

  router.push({ name: 'edit-submission', params: { workId: workID } })
}

async function loadPage() {
  if (!contestID.value || !trackID.value) {
    ElMessage.error('页面参数不完整')
    return
  }

  loading.value = true
  try {
    const [contestInfo, trackInfo, submissions] = await Promise.all([
      fetchContestByID(contestID.value),
      fetchTrackByID(trackID.value),
      fetchMySubmissions(),
    ])

    if (!contestInfo) {
      ElMessage.error('比赛不存在或已删除')
      return
    }

    contest.value = contestInfo
    track.value = trackInfo
    mySubmissions.value = submissions
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '赛道详情加载失败')
  } finally {
    loading.value = false
  }
}

async function submitWork() {
  if (!formRef.value || !track.value?.trackID) {
    return
  }

  if (!canSubmit.value) {
    ElMessage.warning(submitBlockedMessage.value)
    return
  }

  await formRef.value.validate()

  if (!selectedFile.value) {
    ElMessage.warning('请上传稿件文件（.doc/.docx）')
    return
  }

  const payload: SubmissionPayload = {
    workTitle: form.workTitle.trim(),
    trackID: track.value.trackID,
    workInfos: {
      description: form.description.trim(),
      submittedAt: dayjs().toISOString(),
      contestID: contestID.value,
      trackName: track.value.trackName,
    },
  }

  submitting.value = true
  try {
    await createSubmission(payload, selectedFile.value)
    ElMessage.success('投稿成功')
    router.push({ name: 'my-submissions' })
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '投稿失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}

onMounted(loadPage)
</script>

<template>
  <section class="page-card track-detail-page" v-loading="loading">
    <div class="header-row">
      <div>
        <h1 class="page-title">赛道详情与投稿</h1>
        <p class="page-subtitle">查看赛道要求并上传 doc/docx 稿件</p>
      </div>
      <el-button plain @click="goBack">返回比赛详情</el-button>
    </div>

    <el-descriptions v-if="contest && track" border :column="2">
      <el-descriptions-item label="比赛名称">{{ contest.contestName || '-' }}</el-descriptions-item>
      <el-descriptions-item label="当前状态">
        <el-tag v-if="contestStatus === '进行中'" type="success">进行中</el-tag>
        <el-tag v-else-if="contestStatus === '未开始'" type="warning">未开始</el-tag>
        <el-tag v-else type="info">已结束</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="赛道名称">{{ track.trackName || '-' }}</el-descriptions-item>
      <el-descriptions-item label="赛道ID">{{ track.trackID || '-' }}</el-descriptions-item>
      <el-descriptions-item label="赛道说明" :span="2">{{ track.trackDescription || '-' }}</el-descriptions-item>
    </el-descriptions>

    <section v-if="canSubmit" class="form-card">
      <h3>提交稿件</h3>
      <el-form ref="formRef" :model="form" label-position="top">
        <el-form-item
          label="稿件标题"
          prop="workTitle"
          :rules="[{ required: true, message: '请输入稿件标题', trigger: 'blur' }]"
        >
          <el-input v-model="form.workTitle" placeholder="请输入稿件标题" />
        </el-form-item>

        <el-form-item label="稿件简介（可选）" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="4" placeholder="请简要描述稿件内容" />
        </el-form-item>

        <el-form-item label="上传文件（仅 .doc/.docx）">
          <el-upload
            ref="uploadRef"
            :auto-upload="false"
            :limit="1"
            :on-change="handleFileChange"
            :on-remove="handleFileRemove"
            accept=".doc,.docx"
          >
            <el-button type="primary" plain>选择文件</el-button>
          </el-upload>
        </el-form-item>

        <el-button type="primary" :loading="submitting" @click="submitWork">
          提交投稿
        </el-button>
      </el-form>
    </section>

    <section v-else class="form-card">
      <h3>提交稿件</h3>
      <el-alert :title="submitBlockedMessage" type="info" show-icon :closable="false" />
    </section>

    <section class="history-card">
      <h3>我在该赛道的稿件</h3>
      <div class="table-wrap">
        <el-table :data="myTrackSubmissions" empty-text="该赛道暂无投稿记录" style="width: 100%">
          <el-table-column prop="workID" label="稿件ID" width="100" />
          <el-table-column prop="workTitle" label="稿件标题" min-width="220" />
          <el-table-column label="时间" min-width="180">
            <template #default="scope">
              {{ formatDate(String(scope.row.workInfos?.submittedAt || scope.row.workInfos?.updatedAt || '')) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="scope">
              <el-button v-if="canSubmit" link type="primary" @click="openEditSubmission(scope.row.workID)">
                修改稿件
              </el-button>
              <el-text v-else type="info">仅可查看</el-text>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </section>
  </section>
</template>

<style scoped>
.track-detail-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.form-card,
.history-card {
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 14px;
  width: 100%;
  min-width: 0;
}

.form-card h3,
.history-card h3 {
  margin: 0 0 12px;
}

.table-wrap {
  width: 100%;
  min-width: 0;
  overflow-x: auto;
}
</style>
