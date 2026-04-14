<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteAuthor as deleteAuthorByID,
  fetchAuthorByID,
  fetchAuthors,
  updateAuthor,
} from '@/services/repositories/authorRepository'
import type { Author } from '@/types/api'

const loading = ref(false)
const authors = ref<Author[]>([])
const detailLoading = ref(false)
const detailDrawerVisible = ref(false)
const currentAuthor = ref<Author | null>(null)

const editDialogVisible = ref(false)
const editSubmitting = ref(false)
const editingAuthorID = ref<number | null>(null)
const editAuthorInfosText = ref('{}')

const editForm = reactive<Author>({
  authorName: '',
  penName: '',
  authorEmail: '',
  authorInfos: {},
})

const filters = reactive({
  authorName: '',
  offsetText: '0',
  limitText: '20',
})

function parseNonNegativeInteger(text: string, label: string) {
  const parsed = Number(text)
  if (!Number.isInteger(parsed) || parsed < 0) {
    ElMessage.warning(`${label} 需要是非负整数`)
    return null
  }
  return parsed
}

function parsePositiveInteger(text: string, label: string) {
  const parsed = Number(text)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    ElMessage.warning(`${label} 需要是正整数`)
    return null
  }
  return parsed
}

function formatAuthorInfos(infos: Author['authorInfos']) {
  if (!infos || Object.keys(infos).length === 0) {
    return '-'
  }
  const text = JSON.stringify(infos)
  return text.length > 80 ? `${text.slice(0, 80)}...` : text
}

async function queryAuthors() {
  const offset = parseNonNegativeInteger(filters.offsetText, '偏移量')
  const limit = parsePositiveInteger(filters.limitText, '每页条数')
  if (offset === null || limit === null) {
    return
  }

  loading.value = true
  try {
    authors.value = await fetchAuthors({
      authorName: filters.authorName.trim() || undefined,
      offset,
      limit,
    })
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作者查询失败')
  } finally {
    loading.value = false
  }
}

function resetEditForm() {
  Object.assign(editForm, {
    authorName: '',
    penName: '',
    authorEmail: '',
    authorInfos: {},
  })
  editingAuthorID.value = null
  editAuthorInfosText.value = '{}'
}

function normalizeOptionalText(value: string | undefined) {
  const text = (value || '').trim()
  return text || undefined
}

function parseAuthorInfosText() {
  const raw = editAuthorInfosText.value.trim()
  if (!raw) {
    return {}
  }

  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
    ElMessage.warning('扩展信息必须是 JSON 对象')
    return null
  } catch {
    ElMessage.warning('扩展信息不是合法 JSON')
    return null
  }
}

async function openAuthorDetail(authorID: number | undefined) {
  if (!authorID) {
    ElMessage.warning('作者 ID 不存在')
    return
  }

  detailDrawerVisible.value = true
  detailLoading.value = true
  try {
    currentAuthor.value = await fetchAuthorByID(authorID)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作者详情加载失败')
  } finally {
    detailLoading.value = false
  }
}

async function openEditDialog(authorID: number | undefined) {
  if (!authorID) {
    ElMessage.warning('作者 ID 不存在')
    return
  }

  editDialogVisible.value = true
  editSubmitting.value = true
  try {
    const author = await fetchAuthorByID(authorID)
    editingAuthorID.value = authorID
    Object.assign(editForm, {
      authorName: author.authorName || '',
      penName: author.penName || '',
      authorEmail: author.authorEmail || '',
      authorInfos: author.authorInfos || {},
    })
    editAuthorInfosText.value = JSON.stringify(author.authorInfos || {}, null, 2)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作者信息加载失败')
    editDialogVisible.value = false
  } finally {
    editSubmitting.value = false
  }
}

async function submitEdit() {
  if (!editingAuthorID.value) {
    ElMessage.warning('缺少作者 ID')
    return
  }

  const normalizedName = (editForm.authorName || '').trim()
  if (!normalizedName) {
    ElMessage.warning('作者名不能为空')
    return
  }

  const authorInfos = parseAuthorInfosText()
  if (authorInfos === null) {
    return
  }

  editSubmitting.value = true
  try {
    await updateAuthor(editingAuthorID.value, {
      authorName: normalizedName,
      penName: normalizeOptionalText(editForm.penName),
      authorEmail: normalizeOptionalText(editForm.authorEmail),
      authorInfos,
    })
    ElMessage.success('作者更新成功')
    editDialogVisible.value = false
    await queryAuthors()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '作者更新失败')
  } finally {
    editSubmitting.value = false
  }
}

async function removeAuthor(authorID: number | undefined) {
  if (!authorID) {
    ElMessage.warning('作者 ID 不存在')
    return
  }

  try {
    await ElMessageBox.confirm('删除作者后无法恢复，确认继续吗？', '删除作者', {
      type: 'warning',
    })
    await deleteAuthorByID(authorID)
    authors.value = authors.value.filter((author) => author.authorID !== authorID)
    ElMessage.success('作者已删除')
  } catch {
    // canceled
  }
}

function resetFilters() {
  Object.assign(filters, {
    authorName: '',
    offsetText: '0',
    limitText: '20',
  })
}

onMounted(queryAuthors)
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">作者管理</h1>
        <p class="page-subtitle">按作者名与分页参数查询作者信息</p>
      </div>
    </div>

    <div class="filters-row">
      <el-space wrap>
        <el-input v-model="filters.authorName" placeholder="作者名" style="width: 220px" clearable />
        <el-input v-model="filters.offsetText" placeholder="偏移量" style="width: 140px" />
        <el-input v-model="filters.limitText" placeholder="每页条数" style="width: 140px" />
        <el-button type="primary" @click="queryAuthors">查询</el-button>
        <el-button @click="resetFilters">重置</el-button>
      </el-space>
    </div>

    <el-table :data="authors" v-loading="loading" style="width: 100%" empty-text="暂无作者数据">
      <el-table-column prop="authorName" label="作者名" min-width="160" />
      <el-table-column prop="penName" label="笔名" min-width="140" />
      <el-table-column prop="authorEmail" label="邮箱" min-width="220" />
      <el-table-column prop="authorID" label="作者ID" width="100" />
      <el-table-column label="扩展信息" min-width="220">
        <template #default="scope">
          {{ formatAuthorInfos(scope.row.authorInfos) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="210">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openAuthorDetail(scope.row.authorID)">详情</el-button>
            <el-button link type="success" @click="openEditDialog(scope.row.authorID)">编辑</el-button>
            <el-button link type="danger" @click="removeAuthor(scope.row.authorID)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="detailDrawerVisible" size="520px" title="作者详情">
      <el-skeleton v-if="detailLoading" :rows="6" animated />
      <el-descriptions v-else-if="currentAuthor" border :column="1" size="small">
        <el-descriptions-item label="作者ID">{{ currentAuthor.authorID || '-' }}</el-descriptions-item>
        <el-descriptions-item label="作者名">{{ currentAuthor.authorName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="笔名">{{ currentAuthor.penName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="邮箱">{{ currentAuthor.authorEmail || '-' }}</el-descriptions-item>
        <el-descriptions-item label="扩展信息">
          <pre class="author-json">{{ JSON.stringify(currentAuthor.authorInfos || {}, null, 2) }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>

    <el-dialog v-model="editDialogVisible" title="编辑作者" width="620px" @closed="resetEditForm">
      <el-form label-position="top" v-loading="editSubmitting">
        <el-form-item label="作者名" required>
          <el-input v-model="editForm.authorName" />
        </el-form-item>
        <el-form-item label="笔名">
          <el-input v-model="editForm.penName" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="editForm.authorEmail" />
        </el-form-item>
        <el-form-item label="扩展信息(JSON对象)">
          <el-input v-model="editAuthorInfosText" type="textarea" :rows="6" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="submitEdit">保存</el-button>
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

.filters-row {
  margin-bottom: 12px;
}

.author-json {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
