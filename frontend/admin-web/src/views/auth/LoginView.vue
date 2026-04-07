<script setup lang="ts">
import { reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const form = reactive({
  adminName: '',
  password: '',
})

async function onSubmit() {
  try {
    await authStore.signIn(form.adminName, form.password)
    ElMessage.success('登录成功')
    const redirectTarget = String(route.query.redirect || '/')
    router.replace(redirectTarget)
  } catch {
    ElMessage.error(authStore.errorMessage || '登录失败，请检查账号或服务状态')
  }
}
</script>

<template>
  <div class="login-page">
    <div class="aurora" />
    <el-card class="login-card">
      <h1>Ubik 管理后台</h1>
      <p class="subtitle">现代化赛事管理与流程控制平台</p>

      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="管理员用户名">
          <el-input v-model="form.adminName" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-button class="submit-button" type="primary" :loading="authStore.loading" @click="onSubmit">
          登录后台
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  position: relative;
  padding: 24px;
}

.aurora {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 20% 20%, rgba(11, 110, 242, 0.2), transparent 40%),
    radial-gradient(circle at 80% 80%, rgba(15, 157, 88, 0.2), transparent 35%);
  filter: blur(0.4px);
}

.login-card {
  width: min(420px, 100%);
  border-radius: 16px;
  z-index: 1;
}

h1 {
  margin: 0;
  font-size: 28px;
}

.subtitle {
  margin: 8px 0 20px;
  color: var(--text-secondary);
}

.submit-button {
  width: 100%;
  margin-top: 8px;
}

.tip {
  margin-top: 14px;
  font-size: 12px;
  color: var(--text-secondary);
}
</style>
