<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchAuthors } from '@/services/repositories/authorRepository'
import type { Author } from '@/types/api'

const loading = ref(false)
const authors = ref<Author[]>([])

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
    </el-table>
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
</style>
