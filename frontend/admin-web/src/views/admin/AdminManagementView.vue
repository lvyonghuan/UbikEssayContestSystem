<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchAdmins } from '@/services/repositories/futureRepositories'
import type { AdminProfile } from '@/types/api'

const admins = ref<AdminProfile[]>([])

onMounted(async () => {
  admins.value = await fetchAdmins()
})
</script>

<template>
  <section class="page-card">
    <h1 class="page-title">管理员管理</h1>
    <p class="page-subtitle">管理员模块已完成前端页与数据契约，后端接口到位即可联调</p>

    <el-table :data="admins" style="width: 100%">
      <el-table-column prop="adminID" label="ID" width="80" />
      <el-table-column prop="adminName" label="用户名" min-width="160" />
      <el-table-column prop="adminEmail" label="邮箱" min-width="220" />
      <el-table-column label="角色">
        <template #default="scope">
          <el-tag v-for="role in scope.row.roleNames" :key="role" class="role-tag">{{ role }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<style scoped>
.role-tag {
  margin-right: 8px;
}
</style>
