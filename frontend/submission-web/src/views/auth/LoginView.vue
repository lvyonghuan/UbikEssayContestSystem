<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const form = reactive({
  identifier: '',
  password: '',
})

const registerRoute = computed(() => {
  const redirect = route.query.redirect
  if (redirect) {
    return { name: 'register', query: { redirect: String(redirect) } }
  }
  return { name: 'register' }
})

async function onSubmit() {
  if (!form.identifier.trim() || !form.password) {
    ElMessage.warning('请填写账号和密码')
    return
  }

  try {
    await authStore.signIn(form.identifier, form.password)
    ElMessage.success('登录成功')
    const redirectTarget = String(route.query.redirect || '/dashboard')
    router.replace(redirectTarget)
  } catch {
    ElMessage.error(authStore.errorMessage || '登录失败，请检查账号或服务状态')
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="hero-grid" />
    <el-card class="auth-card">
      <h1 class="title">作者登录</h1>
      <p class="subtitle">进入投稿工作台，查看赛事并管理你的稿件</p>

      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="账号（用户名或邮箱）">
          <el-input v-model="form.identifier" placeholder="请输入用户名或邮箱" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-button class="submit-button" type="primary" :loading="authStore.loading" @click="onSubmit">
          登录
        </el-button>
      </el-form>

      <div class="extra-row">
        <span>还没有账号？</span>
        <router-link :to="registerRoute">立即注册</router-link>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  position: relative;
  padding: 24px;
  overflow: hidden;
}

.hero-grid {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 8% 16%, rgba(17, 128, 255, 0.2), transparent 44%),
    radial-gradient(circle at 88% 84%, rgba(255, 124, 50, 0.18), transparent 40%),
    linear-gradient(120deg, rgba(17, 24, 39, 0.04) 1px, transparent 1px),
    linear-gradient(30deg, rgba(17, 24, 39, 0.04) 1px, transparent 1px);
  background-size: auto, auto, 28px 28px, 28px 28px;
}

.auth-card {
  width: min(460px, 100%);
  border-radius: 18px;
  z-index: 1;
}

.title {
  margin: 0;
  font-size: 30px;
}

.subtitle {
  margin: 8px 0 20px;
  color: var(--text-secondary);
}

.submit-button {
  width: 100%;
  margin-top: 8px;
}

.extra-row {
  margin-top: 16px;
  display: flex;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}
</style>
