<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { updateAuthorProfile } from '@/services/repositories/authorAuthRepository'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const saving = ref(false)

const form = reactive({
  authorName: '',
  authorEmail: '',
  penName: '',
})

onMounted(() => {
  form.authorName = authStore.authorName || ''
})

async function saveProfile() {
  if (!form.authorName.trim()) {
    ElMessage.warning('用户名不能为空')
    return
  }

  saving.value = true
  try {
    const updated = await updateAuthorProfile({
      authorName: form.authorName.trim(),
      authorEmail: form.authorEmail.trim() || undefined,
      penName: form.penName.trim() || undefined,
    })

    authStore.authorName = updated.authorName || form.authorName
    ElMessage.success('账号信息更新成功')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '账号信息更新失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="page-card profile-page">
    <h1 class="page-title">账号信息</h1>
    <p class="page-subtitle">维护你的作者资料，便于赛事方联系和展示</p>

    <el-form label-position="top" class="profile-form">
      <el-form-item label="用户名">
        <el-input v-model="form.authorName" placeholder="请输入用户名" />
      </el-form-item>
      <el-form-item label="邮箱">
        <el-input v-model="form.authorEmail" placeholder="请输入邮箱" />
      </el-form-item>
      <el-form-item label="笔名">
        <el-input v-model="form.penName" placeholder="请输入笔名" />
      </el-form-item>
      <el-button type="primary" :loading="saving" @click="saveProfile">保存修改</el-button>
    </el-form>
  </section>
</template>

<style scoped>
.profile-page {
  max-width: 760px;
}

.profile-form {
  margin-top: 16px;
}
</style>
