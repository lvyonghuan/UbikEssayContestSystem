<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import type { Author } from '@/types/api'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const form = reactive({
  authorName: '',
  authorEmail: '',
  penName: '',
  password: '',
  confirmPassword: '',
})

const loginRoute = computed(() => {
  const redirect = route.query.redirect
  if (redirect) {
    return { name: 'login', query: { redirect: String(redirect) } }
  }
  return { name: 'login' }
})

async function onSubmit() {
  if (!form.authorName.trim()) {
    ElMessage.warning('请输入用户名')
    return
  }
  const normalizedEmail = form.authorEmail.trim()
  if (!normalizedEmail) {
    ElMessage.warning('请输入邮箱')
    return
  }
  if (!form.password) {
    ElMessage.warning('请输入密码')
    return
  }
  if (form.password !== form.confirmPassword) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }

  const payload: Author = {
    authorName: form.authorName.trim(),
    authorEmail: normalizedEmail,
    penName: form.penName.trim() || undefined,
    password: form.password,
  }

  try {
    await authStore.signUpThenSignIn(payload)
    ElMessage.success('注册成功，已自动登录')
    const redirectTarget = String(route.query.redirect || '/dashboard')
    router.replace(redirectTarget)
  } catch {
    ElMessage.error(authStore.errorMessage || '注册失败，请稍后重试')
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="hero-grid" />
    <el-card class="auth-card">
      <h1 class="title">作者注册</h1>
      <p class="subtitle">注册后将自动登录并进入比赛看板</p>

      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="用户名">
          <el-input v-model="form.authorName" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="邮箱（必填）">
          <el-input v-model="form.authorEmail" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="笔名">
          <el-input v-model="form.penName" placeholder="可选，投稿展示名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input v-model="form.confirmPassword" type="password" show-password placeholder="请再次输入密码" />
        </el-form-item>
        <el-button class="submit-button" type="primary" :loading="authStore.loading" @click="onSubmit">
          注册并登录
        </el-button>
      </el-form>

      <div class="extra-row">
        <span>已有账号？</span>
        <router-link :to="loginRoute">返回登录</router-link>
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
    radial-gradient(circle at 6% 10%, rgba(11, 110, 242, 0.2), transparent 42%),
    radial-gradient(circle at 85% 88%, rgba(15, 157, 88, 0.18), transparent 36%),
    linear-gradient(130deg, rgba(17, 24, 39, 0.04) 1px, transparent 1px),
    linear-gradient(40deg, rgba(17, 24, 39, 0.04) 1px, transparent 1px);
  background-size: auto, auto, 24px 24px, 24px 24px;
}

.auth-card {
  width: min(480px, 100%);
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
