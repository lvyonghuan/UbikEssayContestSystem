<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createSubAdmin,
  disableSubAdmin,
  fetchSubAdmins,
  removeSubAdmin,
  updateSubAdminPermissions,
} from '@/services/repositories/subAdminRepository'
import { fetchRolePermissions } from '@/services/repositories/futureRepositories'
import type { CreateSubAdminRequest, RolePermission, SubAdminInfo } from '@/types/api'

const loading = ref(false)
const admins = ref<SubAdminInfo[]>([])
const roles = ref<RolePermission[]>([])

const createDialogVisible = ref(false)
const permissionDialogVisible = ref(false)
const selectedAdmin = ref<SubAdminInfo | null>(null)

const createForm = reactive<CreateSubAdminRequest>({
  adminName: '',
  adminEmail: '',
  permissionNames: [],
})

const permissionForm = reactive({
  permissionNames: [] as string[],
})

const allPermissionNames = computed(() => {
  const names = new Set<string>()
  roles.value.forEach((role) => {
    role.permissions.forEach((permission) => names.add(permission))
  })
  admins.value.forEach((admin) => {
    admin.permissionNames.forEach((permission) => names.add(permission))
  })
  return [...names]
})

async function loadData() {
  loading.value = true
  try {
    const [adminRows, roleRows] = await Promise.all([fetchSubAdmins(), fetchRolePermissions()])
    admins.value = adminRows
    roles.value = roleRows
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '子管理员列表加载失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  Object.assign(createForm, {
    adminName: '',
    adminEmail: '',
    permissionNames: [],
  })
  createDialogVisible.value = true
}

async function submitCreate() {
  if (!createForm.adminName.trim()) {
    ElMessage.warning('请填写子管理员姓名')
    return
  }
  if (!createForm.adminEmail.trim()) {
    ElMessage.warning('请填写子管理员邮箱')
    return
  }

  try {
    const result = await createSubAdmin({
      adminName: createForm.adminName.trim(),
      adminEmail: createForm.adminEmail.trim(),
      permissionNames: createForm.permissionNames || [],
    })
    const tempPasswordText = result.tempPassword ? `，临时密码：${result.tempPassword}` : ''
    ElMessage.success(`子管理员创建成功${tempPasswordText}`)
    createDialogVisible.value = false
    await loadData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建子管理员失败')
  }
}

function openPermissionDialog(admin: SubAdminInfo) {
  selectedAdmin.value = admin
  permissionForm.permissionNames = [...admin.permissionNames]
  permissionDialogVisible.value = true
}

async function submitPermissions() {
  if (!selectedAdmin.value) {
    return
  }

  try {
    await updateSubAdminPermissions(selectedAdmin.value.adminID, {
      permissionNames: [...permissionForm.permissionNames],
    })
    ElMessage.success('权限更新成功')
    permissionDialogVisible.value = false
    await loadData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '权限更新失败')
  }
}

async function disableAccount(admin: SubAdminInfo) {
  try {
    await ElMessageBox.confirm('禁用后该管理员将无法登录，确认继续吗？', '禁用子管理员', {
      type: 'warning',
    })
    await disableSubAdmin(admin.adminID)
    ElMessage.success('子管理员已禁用')
    await loadData()
  } catch {
    // canceled
  }
}

async function deleteAdmin(admin: SubAdminInfo) {
  try {
    await ElMessageBox.confirm('删除后不可恢复，确认继续吗？', '删除子管理员', {
      type: 'warning',
    })
    await removeSubAdmin(admin.adminID)
    ElMessage.success('子管理员已删除')
    await loadData()
  } catch {
    // canceled
  }
}

onMounted(loadData)
</script>

<template>
  <section class="page-card">
    <div class="header-row">
      <div>
        <h1 class="page-title">子管理员管理</h1>
        <p class="page-subtitle">首期支持列表、创建、权限更新、禁用与删除</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">创建子管理员</el-button>
    </div>

    <el-table :data="admins" style="width: 100%" v-loading="loading" empty-text="暂无子管理员">
      <el-table-column prop="adminName" label="姓名" min-width="160" />
      <el-table-column prop="adminEmail" label="邮箱" min-width="220" />
      <el-table-column label="状态" width="110">
        <template #default="scope">
          <el-tag :type="scope.row.isActive ? 'success' : 'info'">{{ scope.row.isActive ? '启用中' : '已禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="adminID" label="ID" width="80" />
      <el-table-column label="权限">
        <template #default="scope">
          <el-tag v-for="permission in scope.row.permissionNames" :key="permission" class="role-tag">
            {{ permission }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="scope">
          <el-space>
            <el-button link type="primary" @click="openPermissionDialog(scope.row)">权限</el-button>
            <el-button
              link
              type="warning"
              :disabled="!scope.row.isActive"
              @click="disableAccount(scope.row)"
            >
              禁用
            </el-button>
            <el-button link type="danger" @click="deleteAdmin(scope.row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createDialogVisible" title="创建子管理员" width="520px">
      <el-form label-position="top">
        <el-form-item label="姓名" required>
          <el-input v-model="createForm.adminName" />
        </el-form-item>
        <el-form-item label="邮箱" required>
          <el-input v-model="createForm.adminEmail" />
        </el-form-item>
        <el-form-item label="权限分配">
          <el-checkbox-group v-model="createForm.permissionNames">
            <el-checkbox v-for="permission in allPermissionNames" :key="permission" :label="permission">
              {{ permission }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="permissionDialogVisible"
      :title="`更新权限 - ${selectedAdmin?.adminName || ''}`"
      width="520px"
    >
      <el-form label-position="top">
        <el-form-item label="权限集合">
          <el-checkbox-group v-model="permissionForm.permissionNames">
            <el-checkbox v-for="permission in allPermissionNames" :key="permission" :label="permission">
              {{ permission }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="permissionDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitPermissions">保存权限</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped>
.role-tag {
  margin-right: 8px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}
</style>
