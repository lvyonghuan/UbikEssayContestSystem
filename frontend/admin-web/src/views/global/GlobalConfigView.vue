<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchGlobalConfig } from '@/services/repositories/futureRepositories'

const configForm = reactive({
  siteName: '',
  emailAddress: '',
  emailSmtpServer: '',
  emailSmtpPort: '',
})

onMounted(async () => {
  Object.assign(configForm, await fetchGlobalConfig())
})

function save() {
  ElMessage.success('当前为前端预置模块，等待后端接口补齐后可直接接入')
}
</script>

<template>
  <section class="page-card">
    <h1 class="page-title">全局配置</h1>
    <p class="page-subtitle">站点信息与邮件发送配置占位模块，可直接切换真实接口</p>

    <el-form label-position="top">
      <el-form-item label="站点名称">
        <el-input v-model="configForm.siteName" />
      </el-form-item>
      <el-form-item label="邮箱地址">
        <el-input v-model="configForm.emailAddress" />
      </el-form-item>
      <el-form-item label="SMTP 服务器">
        <el-input v-model="configForm.emailSmtpServer" />
      </el-form-item>
      <el-form-item label="SMTP 端口">
        <el-input v-model="configForm.emailSmtpPort" />
      </el-form-item>
      <el-button type="primary" @click="save">保存配置</el-button>
    </el-form>
  </section>
</template>
