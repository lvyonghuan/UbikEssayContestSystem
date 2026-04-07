<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { sidebarMenu } from '@/navigation/menu'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const activeRoute = computed(() => route.name as string)

function navigate(routeName: string) {
  router.push({ name: routeName })
}

function logout() {
  authStore.signOut()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="layout-root">
    <aside class="sidebar">
      <h1 class="brand">Ubik Admin</h1>
      <p class="brand-subtitle">管理后台</p>
      <el-menu class="menu" :default-active="activeRoute" @select="navigate">
        <el-menu-item v-for="menuItem in sidebarMenu" :key="menuItem.routeName" :index="menuItem.routeName">
          {{ menuItem.label }}
        </el-menu-item>
      </el-menu>
    </aside>

    <main class="main-panel">
      <header class="topbar page-card">
        <div>
          <h2 class="topbar-title">{{ route.meta.title || '后台管理' }}</h2>
          <p class="topbar-hint">面向赛事管理与后续功能扩展的前端工作台</p>
        </div>
        <el-button type="danger" plain @click="logout">退出登录</el-button>
      </header>

      <section class="content-area">
        <transition name="fade-slide" mode="out-in">
          <router-view />
        </transition>
      </section>
    </main>
  </div>
</template>

<style scoped>
.layout-root {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 260px 1fr;
}

.sidebar {
  background: linear-gradient(180deg, var(--bg-panel) 0%, var(--bg-panel-soft) 100%);
  color: var(--text-on-dark);
  padding: 24px 16px;
}

.brand {
  margin: 0;
  font-size: 26px;
  letter-spacing: 0.6px;
}

.brand-subtitle {
  margin: 6px 0 24px;
  color: #9fb8df;
  font-size: 13px;
}

.menu {
  border-right: none;
  background: transparent;
}

.menu :deep(.el-menu-item) {
  border-radius: 10px;
  margin-bottom: 8px;
  color: #e5edfc;
}

.menu :deep(.el-menu-item.is-active) {
  background: rgba(11, 110, 242, 0.2);
  color: #ffffff;
}

.main-panel {
  padding: 18px;
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 16px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.topbar-title {
  margin: 0;
  font-size: 20px;
}

.topbar-hint {
  margin: 4px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.content-area {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

@media (max-width: 960px) {
  .layout-root {
    grid-template-columns: 1fr;
  }

  .sidebar {
    padding-bottom: 12px;
  }

  .menu {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .menu :deep(.el-menu-item) {
    margin: 0;
  }
}
</style>
