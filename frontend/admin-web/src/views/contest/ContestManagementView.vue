<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createContest, fetchContests, removeContest, updateContest } from '@/services/repositories/contestRepository'
import type { Contest } from '@/types/api'

const loading = ref(false)
const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const contests = ref<Contest[]>([])

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

function openCreateDialog() {
  editingId.value = null
  Object.assign(form, {
    contestName: '',
    contestIntroduction: '',
    contestStartDate: '',
    contestEndDate: '',
  })
  dialogVisible.value = true
}

function openEditDialog(contest: Contest) {
  editingId.value = contest.contestID || null
  Object.assign(form, {
    ...contest,
    contestStartDate: toPickerValue(contest.contestStartDate),
    contestEndDate: toPickerValue(contest.contestEndDate),
  })
  dialogVisible.value = true
}

function toPickerValue(value: string) {
  const parsed = dayjs(value)
  if (!parsed.isValid()) {
    return value
  }
  return parsed.format('YYYY-MM-DD HH:mm')
}

async function saveContest() {
  try {
    if (editingId.value) {
      await updateContest(editingId.value, form)
      ElMessage.success('赛事更新成功')
    } else {
      await createContest(form)
      ElMessage.success('赛事创建成功')
    }
    dialogVisible.value = false
    await loadContests()
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
      <el-table-column prop="contestID" label="ID" width="80" />
      <el-table-column prop="contestName" label="赛事名称" min-width="160" />
      <el-table-column prop="contestStartDate" label="开始时间" min-width="160" />
      <el-table-column prop="contestEndDate" label="结束时间" min-width="160" />
      <el-table-column label="操作" width="190">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openEditDialog(scope.row)">编辑</el-button>
            <el-button link type="danger" @click="deleteContest(scope.row.contestID)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑赛事' : '新建赛事'" width="520px">
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
</style>
