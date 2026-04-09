<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import type { FormInstance, UploadFile, UploadInstance } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { fetchContestByID } from '@/services/repositories/contestRepository'
import { fetchSubmissionByID, updateSubmission } from '@/services/repositories/submissionRepository'
import type { JsonObject, SubmissionUpdatePayload, Work } from '@/types/api'
import { validateDocFile } from '@/utils/file'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const submitting = ref(false)
const currentWork = ref<Work | null>(null)
const selectedFile = ref<File | null>(null)
const lockedByContest = ref(false)

const formRef = ref<FormInstance>()
const uploadRef = ref<UploadInstance>()

const form = reactive({
  workTitle: '',
  description: '',
})

const workID = computed(() => {
  const parsed = Number(route.params.workId)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null
})

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
  router.push({ name: 'my-submissions' })
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

async function loadWork() {
  if (!workID.value) {
    ElMessage.error('稿件参数不合法')
    return
  }

  loading.value = true
  try {
    const work = await fetchSubmissionByID(workID.value)
    if (!work) {
      ElMessage.error('未找到对应稿件')
      return
    }

    currentWork.value = work
    form.workTitle = work.workTitle || ''
    const description = work.workInfos?.description
    form.description = typeof description === 'string' ? description : ''

    lockedByContest.value = false
    const contestID = parseContestID(work)
    if (contestID) {
      try {
        const contest = await fetchContestByID(contestID)
        if (contest) {
          const contestEnd = dayjs(contest.contestEndDate)
          lockedByContest.value = contestEnd.isValid() && dayjs().isAfter(contestEnd)
        }
      } catch {
        lockedByContest.value = false
      }
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '稿件加载失败')
  } finally {
    loading.value = false
  }
}

async function submitUpdate() {
  if (!formRef.value || !currentWork.value?.workID || typeof currentWork.value.trackID !== 'number') {
    return
  }

  if (lockedByContest.value) {
    ElMessage.warning('所属比赛已结束，禁止修改稿件')
    return
  }

  await formRef.value.validate()

  const description = form.description.trim()

  const mergedWorkInfos: JsonObject = {
    ...(currentWork.value.workInfos || {}),
    updatedAt: dayjs().toISOString(),
  }
  if (description) {
    mergedWorkInfos.description = description
  } else {
    delete mergedWorkInfos.description
  }

  const payload: SubmissionUpdatePayload = {
    workID: currentWork.value.workID,
    trackID: currentWork.value.trackID,
    workTitle: form.workTitle.trim(),
    workInfos: mergedWorkInfos,
  }

  submitting.value = true
  try {
    await updateSubmission(payload, selectedFile.value || undefined)
    ElMessage.success('稿件更新成功')
    router.push({ name: 'my-submissions' })
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '稿件更新失败')
  } finally {
    submitting.value = false
  }
}

onMounted(loadWork)
</script>

<template>
  <section class="page-card edit-page" v-loading="loading">
    <div class="header-row">
      <div>
        <h1 class="page-title">修改稿件</h1>
        <p class="page-subtitle">更新稿件信息并可选替换上传文件，已结束比赛仅支持查看</p>
      </div>
      <el-button plain @click="goBack">返回我的稿件</el-button>
    </div>

    <el-descriptions v-if="currentWork" border :column="2">
      <el-descriptions-item label="稿件ID">{{ currentWork.workID || '-' }}</el-descriptions-item>
      <el-descriptions-item label="赛道">
        {{ currentWork.trackName || `赛道 ${currentWork.trackID || '-'}` }}
      </el-descriptions-item>
    </el-descriptions>

    <el-alert
      v-if="lockedByContest"
      title="所属比赛已结束，该稿件仅支持查看，不能继续修改"
      type="info"
      show-icon
      :closable="false"
    />

    <el-form ref="formRef" :model="form" label-position="top" class="edit-form" :disabled="lockedByContest">
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

      <el-form-item label="替换上传文件（可选，仅 .doc/.docx）">
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :limit="1"
          :on-change="handleFileChange"
          :on-remove="handleFileRemove"
          accept=".doc,.docx"
        >
          <el-button type="primary" plain>选择新文件</el-button>
        </el-upload>
      </el-form-item>

      <el-button v-if="!lockedByContest" type="primary" :loading="submitting" @click="submitUpdate">
        保存修改
      </el-button>
    </el-form>
  </section>
</template>

<style scoped>
.edit-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 860px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.edit-form {
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 14px;
}
</style>
