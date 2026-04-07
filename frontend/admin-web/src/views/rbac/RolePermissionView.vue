<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchRolePermissions } from '@/services/repositories/futureRepositories'
import type { RolePermission } from '@/types/api'

const roles = ref<RolePermission[]>([])

onMounted(async () => {
  roles.value = await fetchRolePermissions()
})
</script>

<template>
  <section class="page-card">
    <h1 class="page-title">角色权限</h1>
    <p class="page-subtitle">RBAC 模块已预置结构，待后端接口补齐即可转真实读写</p>

    <el-table :data="roles" style="width: 100%">
      <el-table-column prop="roleID" label="ID" width="80" />
      <el-table-column prop="roleName" label="角色名" min-width="180" />
      <el-table-column prop="description" label="描述" min-width="220" />
      <el-table-column label="权限">
        <template #default="scope">
          <el-tag v-for="permission in scope.row.permissions" :key="permission" class="tag">{{ permission }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<style scoped>
.tag {
  margin-right: 8px;
}
</style>
